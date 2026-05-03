package repository

import (
	"context"
	"errors"
	"sync"
	"time"

	"med-go/internal/appointment/model"
)

var ErrAppointmentNotFound = errors.New("appointment not found")

type Repository interface {
	Create(ctx context.Context, appointment model.Appointment) error
	List(ctx context.Context) ([]model.Appointment, error)
	GetByID(ctx context.Context, id string) (model.Appointment, error)
	Update(ctx context.Context, appointment model.Appointment) error
}

type MemoryRepository struct {
	mu           sync.RWMutex
	appointments map[string]model.Appointment
	order        []string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		appointments: make(map[string]model.Appointment),
	}
}

func (r *MemoryRepository) Create(_ context.Context, appointment model.Appointment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.appointments[appointment.ID] = appointment
	r.order = append(r.order, appointment.ID)

	return nil
}

func (r *MemoryRepository) List(_ context.Context) ([]model.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	appointments := make([]model.Appointment, 0, len(r.order))
	for _, id := range r.order {
		appointments = append(appointments, r.appointments[id])
	}

	return appointments, nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id string) (model.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	appointment, ok := r.appointments[id]
	if !ok {
		return model.Appointment{}, ErrAppointmentNotFound
	}

	return appointment, nil
}

func (r *MemoryRepository) Update(_ context.Context, appointment model.Appointment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.appointments[appointment.ID]; !ok {
		return ErrAppointmentNotFound
	}

	r.appointments[appointment.ID] = appointment

	return nil
}

func (r *MemoryRepository) UpdateStatus(_ context.Context, id string, status model.Status, updatedAt time.Time) (model.Appointment, model.Status, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	appointment, ok := r.appointments[id]
	if !ok {
		return model.Appointment{}, "", ErrAppointmentNotFound
	}

	oldStatus := appointment.Status
	appointment.Status = status
	appointment.UpdatedAt = updatedAt
	r.appointments[id] = appointment

	return appointment, oldStatus, nil
}
