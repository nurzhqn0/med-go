package jobqueue

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	statusDone     = "done"
	idempotencyTTL = 24 * time.Hour
)

type Queue struct {
	jobs        chan Job
	client      *goredis.Client
	gatewayURL  string
	httpClient  *http.Client
	stdout      io.Writer
	stderr      io.Writer
	maxAttempts int
	backoffs    []time.Duration
}

type Options struct {
	PoolSize    int
	MaxAttempts int
	Backoffs    []time.Duration
}

type Job struct {
	ID             string `json:"job_id"`
	AppointmentID  string `json:"appointment_id"`
	DoctorID       string `json:"doctor_id"`
	OccurredAt     string `json:"occurred_at"`
	Channel        string `json:"channel"`
	Recipient      string `json:"recipient"`
	Message        string `json:"message"`
	IdempotencyKey string `json:"idempotency_key"`
}

type LogLine struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	JobID   string `json:"job_id"`
	Attempt int    `json:"attempt"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

func New(client *goredis.Client, gatewayURL string, poolSize int) *Queue {
	return NewWithOptions(client, gatewayURL, Options{PoolSize: poolSize})
}

func NewWithOptions(client *goredis.Client, gatewayURL string, options Options) *Queue {
	poolSize := defaultInt(options.PoolSize, 3)
	maxAttempts := defaultInt(options.MaxAttempts, 3)
	backoffs := options.Backoffs
	if len(backoffs) == 0 {
		backoffs = []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
	}

	return &Queue{
		jobs:        make(chan Job, poolSize*10),
		client:      client,
		gatewayURL:  gatewayURL,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
		stdout:      os.Stdout,
		stderr:      os.Stderr,
		maxAttempts: maxAttempts,
		backoffs:    backoffs,
	}
}

func NewForTest(client *goredis.Client, gatewayURL string, poolSize int, httpClient *http.Client, stdout, stderr io.Writer) *Queue {
	q := New(client, gatewayURL, poolSize)
	if httpClient != nil {
		q.httpClient = httpClient
	}
	q.stdout = stdout
	q.stderr = stderr
	return q
}

func (q *Queue) Start(ctx context.Context, poolSize int) {
	if poolSize <= 0 {
		poolSize = 3
	}

	var wg sync.WaitGroup
	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.worker(ctx)
		}()
	}

	go func() {
		<-ctx.Done()
		wg.Wait()
	}()
}

func (q *Queue) EnqueueFromEvent(ctx context.Context, event map[string]any) {
	if eventString(event, "event_type") != "appointments.status_updated" || eventString(event, "new_status") != statusDone {
		return
	}

	job := Job{
		AppointmentID: eventString(event, "id"),
		DoctorID:      eventString(event, "doctor_id"),
		OccurredAt:    eventString(event, "occurred_at"),
		Channel:       "email",
		Recipient:     "patient@clinic.kz",
	}
	job.IdempotencyKey = idempotencyKey(eventString(event, "event_type"), job.AppointmentID, job.OccurredAt)
	job.ID = job.IdempotencyKey
	job.Message = fmt.Sprintf("Your appointment %s with doctor %s is complete.", job.AppointmentID, job.DoctorID)

	if q.client == nil {
		q.write(q.stderr, "warn", job.ID, 1, "dead_letter", "redis unavailable; job queue disabled")
		return
	}

	key := q.redisKey(job.ID)
	value, err := q.client.Get(ctx, key).Result()
	if err == nil && value == statusDone {
		q.write(q.stdout, "info", job.ID, 1, "duplicate", "")
		return
	}
	if err != nil && !errors.Is(err, goredis.Nil) {
		q.write(q.stderr, "warn", job.ID, 1, "retry", fmt.Sprintf("idempotency lookup failed: %v", err))
		return
	}

	claimed, err := q.client.SetNX(ctx, key, "processing", idempotencyTTL).Result()
	if err != nil {
		q.write(q.stderr, "warn", job.ID, 1, "retry", fmt.Sprintf("idempotency claim failed: %v", err))
		return
	}
	if !claimed {
		q.write(q.stdout, "info", job.ID, 1, "duplicate", "")
		return
	}

	select {
	case q.jobs <- job:
		q.write(q.stdout, "info", job.ID, 1, "enqueued", "")
	default:
		_ = q.client.Del(ctx, key).Err()
		q.write(q.stderr, "error", job.ID, 1, "dead_letter", "job queue is full")
	}
}

func (q *Queue) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-q.jobs:
			q.process(ctx, job)
		}
	}
}

func (q *Queue) process(ctx context.Context, job Job) {
	var lastErr error

	for attempt := 1; attempt <= q.maxAttempts; attempt++ {
		q.write(q.stdout, "info", job.ID, attempt, "processing", "")

		if err := q.notify(ctx, job); err != nil {
			lastErr = err
			q.write(q.stdout, "warn", job.ID, attempt, "retry", err.Error())
			timer := time.NewTimer(q.backoffFor(attempt))
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}

			continue
		}

		if err := q.client.Set(ctx, q.redisKey(job.ID), statusDone, idempotencyTTL).Err(); err != nil {
			log.Printf("failed to mark job done job_id=%s: %v", job.ID, err)
		}
		q.write(q.stdout, "info", job.ID, attempt, "success", "")
		return
	}

	if lastErr != nil {
		_ = q.client.Del(ctx, q.redisKey(job.ID)).Err()
		q.write(q.stderr, "error", job.ID, q.maxAttempts, "dead_letter", lastErr.Error())
	}
}

func (q *Queue) backoffFor(attempt int) time.Duration {
	if attempt <= 0 || len(q.backoffs) == 0 {
		return time.Second
	}
	if attempt > len(q.backoffs) {
		return q.backoffs[len(q.backoffs)-1]
	}

	return q.backoffs[attempt-1]
}

func (q *Queue) notify(ctx context.Context, job Job) error {
	body := struct {
		IdempotencyKey string `json:"idempotency_key"`
		Channel        string `json:"channel"`
		Recipient      string `json:"recipient"`
		Message        string `json:"message"`
	}{
		IdempotencyKey: job.IdempotencyKey,
		Channel:        job.Channel,
		Recipient:      job.Recipient,
		Message:        job.Message,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, q.gatewayURL+"/notify", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := q.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		return nil
	}
	if response.StatusCode == http.StatusServiceUnavailable {
		return fmt.Errorf("gateway returned 503")
	}

	return fmt.Errorf("gateway returned status %d", response.StatusCode)
}

func (q *Queue) write(writer io.Writer, level, jobID string, attempt int, statusValue, errMessage string) {
	line := LogLine{
		Time:    time.Now().UTC().Format(time.RFC3339),
		Level:   level,
		JobID:   jobID,
		Attempt: attempt,
		Status:  statusValue,
		Error:   errMessage,
	}

	if encodeErr := json.NewEncoder(writer).Encode(line); encodeErr != nil {
		log.Printf("failed to write job log job_id=%s: %v", jobID, encodeErr)
	}
}

func (q *Queue) redisKey(id string) string {
	return "notification:job:" + id
}

func idempotencyKey(eventType, id, occurredAt string) string {
	sum := sha256.Sum256([]byte(eventType + id + occurredAt))
	return hex.EncodeToString(sum[:])
}

func eventString(event map[string]any, key string) string {
	value, _ := event[key].(string)
	return value
}

func defaultInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}

	return value
}
