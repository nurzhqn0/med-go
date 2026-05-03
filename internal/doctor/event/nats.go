package event

import (
	"context"
	"encoding/json"
	"time"

	"med-go/internal/doctor/model"

	"github.com/nats-io/nats.go"
)

const DoctorsCreatedSubject = "doctors.created"

type Publisher struct {
	conn *nats.Conn
}

func NewPublisher(url string) (*Publisher, error) {
	conn, err := nats.Connect(url, nats.Name("doctor-service"), nats.Timeout(3*time.Second))
	if err != nil {
		return nil, err
	}

	return &Publisher{conn: conn}, nil
}

func (p *Publisher) PublishDoctorCreated(_ context.Context, doctor model.Doctor) error {
	payload := struct {
		EventType      string `json:"event_type"`
		OccurredAt     string `json:"occurred_at"`
		ID             string `json:"id"`
		FullName       string `json:"full_name"`
		Specialization string `json:"specialization"`
		Email          string `json:"email"`
	}{
		EventType:      DoctorsCreatedSubject,
		OccurredAt:     time.Now().UTC().Format(time.RFC3339),
		ID:             doctor.ID,
		FullName:       doctor.FullName,
		Specialization: doctor.Specialization,
		Email:          doctor.Email,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return p.conn.Publish(DoctorsCreatedSubject, data)
}

func (p *Publisher) Close() error {
	p.conn.Close()
	return nil
}

type NoopPublisher struct{}

func (NoopPublisher) PublishDoctorCreated(context.Context, model.Doctor) error {
	return nil
}
