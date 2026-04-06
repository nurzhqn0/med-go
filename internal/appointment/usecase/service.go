package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	"med-go/internal/appointment/model"
	"med-go/internal/appointment/repository"
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
}

type DoctorLookup interface {
	Exists(ctx context.Context, id string) (bool, error)
}

type Service struct {
	repo         Repository
	doctorLookup DoctorLookup
}

func NewService(repo Repository, doctorLookup DoctorLookup) *Service {
	return &Service{
		repo:         repo,
		doctorLookup: doctorLookup,
	}
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
		return model.Appointment{}, fmt.Errorf("%w: %v", ErrDoctorServiceUnavailable, err)
	}

	if !exists {
		return model.Appointment{}, ErrDoctorNotFound
	}

	now := time.Now().UTC()
	appointment := model.Appointment{
		ID:          bson.NewObjectID().Hex(),
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

	return appointment, nil
}

func (s *Service) ListAppointments(ctx context.Context) ([]model.Appointment, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetAppointment(ctx context.Context, id string) (model.Appointment, error) {
	appointment, err := s.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, repository.ErrAppointmentNotFound) {
			return model.Appointment{}, repository.ErrAppointmentNotFound
		}

		return model.Appointment{}, err
	}

	return appointment, nil
}

func (s *Service) UpdateStatus(ctx context.Context, id string, rawStatus string) (model.Appointment, error) {
	appointment, err := s.repo.GetByID(ctx, strings.TrimSpace(id))
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

	appointment.Status = status
	appointment.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, appointment); err != nil {
		if errors.Is(err, repository.ErrAppointmentNotFound) {
			return model.Appointment{}, repository.ErrAppointmentNotFound
		}

		return model.Appointment{}, err
	}

	return appointment, nil
}
