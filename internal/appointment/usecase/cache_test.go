package usecase

import (
	"context"
	"testing"
	"time"

	"med-go/internal/appointment/model"
	"med-go/internal/appointment/repository"
)

type appointmentCacheStub struct {
	appointments map[string]model.Appointment
	list         []model.Appointment
	deleted      []string
}

func (c *appointmentCacheStub) GetAppointment(_ context.Context, id string) (model.Appointment, bool, error) {
	appointment, ok := c.appointments[id]
	return appointment, ok, nil
}

func (c *appointmentCacheStub) SetAppointment(_ context.Context, appointment model.Appointment) error {
	c.appointments[appointment.ID] = appointment
	return nil
}

func (c *appointmentCacheStub) GetAppointments(context.Context) ([]model.Appointment, bool, error) {
	return c.list, c.list != nil, nil
}

func (c *appointmentCacheStub) SetAppointments(_ context.Context, appointments []model.Appointment) error {
	c.list = appointments
	return nil
}

func (c *appointmentCacheStub) Delete(_ context.Context, keys ...string) error {
	c.deleted = append(c.deleted, keys...)
	return nil
}

type alwaysDoctorLookup struct{}

func (alwaysDoctorLookup) Exists(context.Context, string) (bool, error) {
	return true, nil
}

func TestGetAppointmentCachesDatabaseResult(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemoryRepository()
	cache := &appointmentCacheStub{appointments: make(map[string]model.Appointment)}
	service := NewService(repo, alwaysDoctorLookup{})
	service.SetCache(cache)

	appointment := model.Appointment{
		ID:        "appt-1",
		Title:     "Visit",
		DoctorID:  "doc-1",
		Status:    model.StatusNew,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := repo.Create(ctx, appointment); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if _, err := service.GetAppointment(ctx, "appt-1"); err != nil {
		t.Fatalf("GetAppointment returned error: %v", err)
	}
	if _, ok := cache.appointments["appt-1"]; !ok {
		t.Fatalf("expected appointment to be stored in cache")
	}

	if err := repo.Update(ctx, model.Appointment{
		ID:        "appt-1",
		Title:     "Changed",
		DoctorID:  "doc-1",
		Status:    model.StatusNew,
		CreatedAt: appointment.CreatedAt,
		UpdatedAt: appointment.UpdatedAt,
	}); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}

	got, err := service.GetAppointment(ctx, "appt-1")
	if err != nil {
		t.Fatalf("second GetAppointment returned error: %v", err)
	}
	if got.Title != "Visit" {
		t.Fatalf("expected cached title Visit, got %q", got.Title)
	}
}

func TestUpdateStatusRefreshesAppointmentCacheAndInvalidatesList(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemoryRepository()
	cache := &appointmentCacheStub{appointments: make(map[string]model.Appointment)}
	service := NewService(repo, alwaysDoctorLookup{})
	service.SetCache(cache)

	now := time.Now().UTC()
	appointment := model.Appointment{
		ID:        "appt-1",
		Title:     "Visit",
		DoctorID:  "doc-1",
		Status:    model.StatusNew,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Create(ctx, appointment); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	updated, err := service.UpdateStatus(ctx, "appt-1", string(model.StatusDone))
	if err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}

	if cached := cache.appointments["appt-1"]; cached.Status != model.StatusDone || cached.ID != updated.ID {
		t.Fatalf("expected updated appointment in cache, got %#v", cached)
	}
	if len(cache.deleted) != 1 || cache.deleted[0] != "appointments:list" {
		t.Fatalf("expected appointments:list invalidation, got %#v", cache.deleted)
	}
}
