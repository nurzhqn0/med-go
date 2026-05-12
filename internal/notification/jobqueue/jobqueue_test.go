package jobqueue

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func TestQueueProcessesDoneEventAndDropsDuplicate(t *testing.T) {
	redisServer := startMiniRedis(t)
	redisClient := goredis.NewClient(&goredis.Options{Addr: redisServer.Addr()})
	defer redisClient.Close()

	var calls int32
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"accepted"}`))
	}))
	defer gateway.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var stdout, stderr bytes.Buffer
	queue := NewForTest(redisClient, gateway.URL, 1, gateway.Client(), &stdout, &stderr)
	queue.Start(ctx, 1)

	event := map[string]any{
		"event_type":  "appointments.status_updated",
		"occurred_at": "2026-05-11T12:00:00Z",
		"id":          "appt-1",
		"doctor_id":   "doc-1",
		"new_status":  "done",
	}

	queue.EnqueueFromEvent(ctx, event)
	waitFor(t, func() bool { return atomic.LoadInt32(&calls) == 1 })

	queue.EnqueueFromEvent(ctx, event)
	time.Sleep(50 * time.Millisecond)
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected duplicate event to be dropped, got %d gateway calls", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr logs, got %s", stderr.String())
	}
}

func TestQueueRetriesAndDeadLetters(t *testing.T) {
	redisServer := startMiniRedis(t)
	redisClient := goredis.NewClient(&goredis.Options{Addr: redisServer.Addr()})
	defer redisClient.Close()

	var calls int32
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"unavailable"}`))
	}))
	defer gateway.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var stdout, stderr bytes.Buffer
	queue := NewForTest(redisClient, gateway.URL, 1, gateway.Client(), &stdout, &stderr)
	queue.Start(ctx, 1)

	queue.EnqueueFromEvent(ctx, map[string]any{
		"event_type":  "appointments.status_updated",
		"occurred_at": "2026-05-11T12:00:00Z",
		"id":          "appt-2",
		"doctor_id":   "doc-1",
		"new_status":  "done",
	})

	waitFor(t, func() bool { return atomic.LoadInt32(&calls) == 3 && stderr.Len() > 0 })
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
	if !bytes.Contains(stderr.Bytes(), []byte(`"status":"dead_letter"`)) {
		t.Fatalf("expected dead_letter log, got %s", stderr.String())
	}
}

func startMiniRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()

	redisServer, err := miniredis.Run()
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("local TCP listeners are blocked in this environment: %v", err)
		}
		t.Fatalf("could not start miniredis: %v", err)
	}
	t.Cleanup(redisServer.Close)

	return redisServer
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(9 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("condition was not met before timeout")
}
