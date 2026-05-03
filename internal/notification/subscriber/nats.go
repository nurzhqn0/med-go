package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
)

var subjects = []string{
	"doctors.created",
	"appointments.created",
	"appointments.status_updated",
}

type Subscriber struct {
	conn *nats.Conn
}

func New(url string) (*Subscriber, error) {
	var lastErr error
	backoff := time.Second

	for attempt := 1; attempt <= 5; attempt++ {
		conn, err := nats.Connect(url, nats.Name("notification-service"), nats.Timeout(3*time.Second))
		if err == nil {
			return &Subscriber{conn: conn}, nil
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
			s.handleMessage(subject, message.Data)
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

func (s *Subscriber) handleMessage(subject string, data []byte) {
	var event map[string]any
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("failed to deserialize event subject=%s: %v", subject, err)
		return
	}

	line := struct {
		Time    string         `json:"time"`
		Subject string         `json:"subject"`
		Event   map[string]any `json:"event"`
	}{
		Time:    time.Now().UTC().Format(time.RFC3339),
		Subject: subject,
		Event:   event,
	}

	if err := json.NewEncoder(os.Stdout).Encode(line); err != nil {
		log.Printf("failed to write notification log subject=%s: %v", subject, err)
	}
}
