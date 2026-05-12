package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

var subjects = []string{
	"doctors.created",
	"appointments.created",
	"appointments.status_updated",
}

type Subscriber struct {
	conn     *nats.Conn
	logger   EventLogger
	jobQueue JobQueue
}

type EventLogger interface {
	Log(subject string, event map[string]any)
}

type JobQueue interface {
	EnqueueFromEvent(ctx context.Context, event map[string]any)
}

func New(url string, logger EventLogger, jobQueue JobQueue) (*Subscriber, error) {
	var lastErr error
	backoff := time.Second

	for attempt := 1; attempt <= 5; attempt++ {
		conn, err := nats.Connect(url, nats.Name("notification-service"), nats.Timeout(3*time.Second))
		if err == nil {
			return &Subscriber{conn: conn, logger: logger, jobQueue: jobQueue}, nil
		}

		lastErr = err
		log.Printf("failed to connect to NATS attempt=%d: %v", attempt, err)
		time.Sleep(backoff)
		backoff *= 2
	}

	return nil, fmt.Errorf("failed to connect to NATS after retries: %w", lastErr)
}

func (s *Subscriber) Run(ctx context.Context) error {
	for _, subject := range subjects {
		subject := subject
		if _, err := s.conn.Subscribe(subject, func(message *nats.Msg) {
			s.handleMessage(ctx, subject, message.Data)
		}); err != nil {
			return err
		}
	}

	if err := s.conn.Flush(); err != nil {
		return err
	}

	<-ctx.Done()
	if err := s.conn.Drain(); err != nil {
		return err
	}
	s.conn.Close()

	return nil
}

func (s *Subscriber) handleMessage(ctx context.Context, subject string, data []byte) {
	var event map[string]any
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("failed to deserialize event subject=%s: %v", subject, err)
		return
	}

	if s.logger != nil {
		s.logger.Log(subject, event)
	}

	if s.jobQueue != nil {
		s.jobQueue.EnqueueFromEvent(ctx, event)
	}
}
