package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"med-go/internal/appointment/model"
	"med-go/internal/appointment/repository"
	"med-go/internal/platform/id"
)

var (
	ErrInvalidAppointmentInput  = errors.New("invalid appointment input")
	ErrInvalidStatusTransition  = errors.New("invalid appointment status transition")
	ErrDoctorNotFound           = errors.New("doctor not found")
	ErrDoctorServiceUnavailable = errors.New("doctor service unavailable")
)

type CreateAppointmentInput struct {
	Title       string
	Description string
	DoctorID    string
}

type Repository interface {
	Create(ctx context.Context, appointment model.Appointment) error
	List(ctx context.Context) ([]model.Appointment, error)
	GetByID(ctx context.Context, id string) (model.Appointment, error)
	Update(ctx context.Context, appointment model.Appointment) error
	UpdateStatus(ctx context.Context, id string, status model.Status, updatedAt time.Time) (model.Appointment, model.Status, error)
}

type DoctorLookup interface {
	Exists(ctx context.Context, id string) (bool, error)
}

type EventPublisher interface {
	PublishAppointmentCreated(ctx context.Context, appointment model.Appointment) error
	PublishAppointmentStatusUpdated(ctx context.Context, appointment model.Appointment, oldStatus model.Status) error
}

type CacheRepository interface {
	GetAppointment(ctx context.Context, id string) (model.Appointment, bool, error)
	SetAppointment(ctx context.Context, appointment model.Appointment) error
	GetAppointments(ctx context.Context) ([]model.Appointment, bool, error)
	SetAppointments(ctx context.Context, appointments []model.Appointment) error
	Delete(ctx context.Context, keys ...string) error
}

type Service struct {
	repo         Repository
	doctorLookup DoctorLookup
	publisher    EventPublisher
	cache        CacheRepository
}

func NewService(repo Repository, doctorLookup DoctorLookup, publishers ...EventPublisher) *Service {
	var publisher EventPublisher
	if len(publishers) > 0 {
		publisher = publishers[0]
	}

	return &Service{
		repo:         repo,
		doctorLookup: doctorLookup,
		publisher:    publisher,
	}
}

func (s *Service) SetCache(cache CacheRepository) {
	s.cache = cache
}

func (s *Service) CreateAppointment(ctx context.Context, input CreateAppointmentInput) (model.Appointment, error) {
	title := strings.TrimSpace(input.Title)
	description := strings.TrimSpace(input.Description)
	doctorID := strings.TrimSpace(input.DoctorID)

	if title == "" || doctorID == "" {
		return model.Appointment{}, ErrInvalidAppointmentInput
	}

	exists, err := s.doctorLookup.Exists(ctx, doctorID)
	if err != nil {
		log.Printf("doctor lookup failed for doctor_id=%s: %v", doctorID, err)
		return model.Appointment{}, fmt.Errorf("%w: %v", ErrDoctorServiceUnavailable, err)
	}

	if !exists {
		return model.Appointment{}, ErrDoctorNotFound
	}

	now := time.Now().UTC()
	appointmentID, err := id.New()
	if err != nil {
		return model.Appointment{}, err
	}

	appointment := model.Appointment{
		ID:          appointmentID,
		Title:       title,
		Description: description,
		DoctorID:    doctorID,
		Status:      model.StatusNew,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, appointment); err != nil {
		return model.Appointment{}, err
	}

	if s.cache != nil {
		if err := s.cache.Delete(ctx, "appointments:list"); err != nil {
			log.Printf("failed to invalidate appointments:list after create appointment_id=%s: %v", appointment.ID, err)
		}
	}

	if s.publisher != nil {
		if err := s.publisher.PublishAppointmentCreated(ctx, appointment); err != nil {
			log.Printf("failed to publish appointments.created appointment_id=%s: %v", appointment.ID, err)
		}
	}

	return appointment, nil
}

func (s *Service) ListAppointments(ctx context.Context) ([]model.Appointment, error) {
	if s.cache != nil {
		appointments, ok, err := s.cache.GetAppointments(ctx)
		if err != nil {
			log.Printf("appointment list cache read failed: %v", err)
		}
		if ok {
			return appointments, nil
		}
	}

	appointments, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		if err := s.cache.SetAppointments(ctx, appointments); err != nil {
			log.Printf("appointment list cache write failed: %v", err)
		}
	}

	return appointments, nil
}

func (s *Service) GetAppointment(ctx context.Context, id string) (model.Appointment, error) {
	appointmentID := strings.TrimSpace(id)
	if s.cache != nil {
		appointment, ok, err := s.cache.GetAppointment(ctx, appointmentID)
		if err != nil {
			log.Printf("appointment cache read failed appointment_id=%s: %v", appointmentID, err)
		}
		if ok {
			return appointment, nil
		}
	}

	appointment, err := s.repo.GetByID(ctx, appointmentID)
	if err != nil {
		if errors.Is(err, repository.ErrAppointmentNotFound) {
			return model.Appointment{}, repository.ErrAppointmentNotFound
		}

		return model.Appointment{}, err
	}

	if s.cache != nil {
		if err := s.cache.SetAppointment(ctx, appointment); err != nil {
			log.Printf("appointment cache write failed appointment_id=%s: %v", appointment.ID, err)
		}
	}

	return appointment, nil
}

func (s *Service) UpdateStatus(ctx context.Context, id string, rawStatus string) (model.Appointment, error) {
	appointmentID := strings.TrimSpace(id)
	appointment, err := s.repo.GetByID(ctx, appointmentID)
	if err != nil {
		if errors.Is(err, repository.ErrAppointmentNotFound) {
			return model.Appointment{}, repository.ErrAppointmentNotFound
		}

		return model.Appointment{}, err
	}

	status, err := model.ParseStatus(strings.TrimSpace(rawStatus))
	if err != nil {
		return model.Appointment{}, err
	}

	if !appointment.Status.CanTransitionTo(status) {
		return model.Appointment{}, fmt.Errorf("%w: %s -> %s", ErrInvalidStatusTransition, appointment.Status, status)
	}

	updatedAppointment, oldStatus, err := s.repo.UpdateStatus(ctx, appointmentID, status, time.Now().UTC())
	if err != nil {
		if errors.Is(err, repository.ErrAppointmentNotFound) {
			return model.Appointment{}, repository.ErrAppointmentNotFound
		}

		return model.Appointment{}, err
	}

	if s.cache != nil {
		if err := s.cache.SetAppointment(ctx, updatedAppointment); err != nil {
			log.Printf("appointment cache write failed appointment_id=%s: %v", updatedAppointment.ID, err)
		}
		if err := s.cache.Delete(ctx, "appointments:list"); err != nil {
			log.Printf("failed to invalidate appointments:list after status update appointment_id=%s: %v", updatedAppointment.ID, err)
		}
	}

	if s.publisher != nil {
		if err := s.publisher.PublishAppointmentStatusUpdated(ctx, updatedAppointment, oldStatus); err != nil {
			log.Printf("failed to publish appointments.status_updated appointment_id=%s: %v", updatedAppointment.ID, err)
		}
	}

	return updatedAppointment, nil
}
