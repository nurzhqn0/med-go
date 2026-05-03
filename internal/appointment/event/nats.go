package event

import (
	"context"
	"encoding/json"
	"time"

	"med-go/internal/appointment/model"

	"github.com/nats-io/nats.go"
)

const (
	AppointmentsCreatedSubject       = "appointments.created"
	AppointmentsStatusUpdatedSubject = "appointments.status_updated"
)

type Publisher struct {
	conn *nats.Conn
}

func NewPublisher(url string) (*Publisher, error) {
	conn, err := nats.Connect(url, nats.Name("appointment-service"), nats.Timeout(3*time.Second))
	if err != nil {
		return nil, err
	}

	return &Publisher{conn: conn}, nil
}

func (p *Publisher) PublishAppointmentCreated(_ context.Context, appointment model.Appointment) error {
	payload := struct {
		EventType  string `json:"event_type"`
		OccurredAt string `json:"occurred_at"`
		ID         string `json:"id"`
		Title      string `json:"title"`
		DoctorID   string `json:"doctor_id"`
		Status     string `json:"status"`
	}{
		EventType:  AppointmentsCreatedSubject,
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		ID:         appointment.ID,
		Title:      appointment.Title,
		DoctorID:   appointment.DoctorID,
		Status:     string(appointment.Status),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return p.conn.Publish(AppointmentsCreatedSubject, data)
}

func (p *Publisher) PublishAppointmentStatusUpdated(_ context.Context, appointment model.Appointment, oldStatus model.Status) error {
	payload := struct {
		EventType  string `json:"event_type"`
		OccurredAt string `json:"occurred_at"`
		ID         string `json:"id"`
		OldStatus  string `json:"old_status"`
		NewStatus  string `json:"new_status"`
	}{
		EventType:  AppointmentsStatusUpdatedSubject,
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		ID:         appointment.ID,
		OldStatus:  string(oldStatus),
		NewStatus:  string(appointment.Status),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return p.conn.Publish(AppointmentsStatusUpdatedSubject, data)
}

func (p *Publisher) Close() error {
	p.conn.Close()
	return nil
}

type NoopPublisher struct{}

func (NoopPublisher) PublishAppointmentCreated(context.Context, model.Appointment) error {
	return nil
}

func (NoopPublisher) PublishAppointmentStatusUpdated(context.Context, model.Appointment, model.Status) error {
	return nil
}
