package usecase

import (
	"context"
	"errors"
	"log"
	"net/mail"
	"strings"

	"med-go/internal/doctor/model"
	"med-go/internal/doctor/repository"
	"med-go/internal/platform/id"
)

var (
	ErrInvalidDoctorInput     = errors.New("invalid doctor input")
	ErrDoctorEmailAlreadyUsed = errors.New("doctor email already exists")
)

type CreateDoctorInput struct {
	FullName       string
	Specialization string
	Email          string
}

type Repository interface {
	Create(ctx context.Context, doctor model.Doctor) error
	List(ctx context.Context) ([]model.Doctor, error)
	GetByID(ctx context.Context, id string) (model.Doctor, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

type EventPublisher interface {
	PublishDoctorCreated(ctx context.Context, doctor model.Doctor) error
}

type CacheRepository interface {
	GetDoctor(ctx context.Context, id string) (model.Doctor, bool, error)
	SetDoctor(ctx context.Context, doctor model.Doctor) error
	GetDoctors(ctx context.Context) ([]model.Doctor, bool, error)
	SetDoctors(ctx context.Context, doctors []model.Doctor) error
	Delete(ctx context.Context, keys ...string) error
}

type Service struct {
	repo      Repository
	publisher EventPublisher
	cache     CacheRepository
}

func NewService(repo Repository, publishers ...EventPublisher) *Service {
	var publisher EventPublisher
	if len(publishers) > 0 {
		publisher = publishers[0]
	}

	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) SetCache(cache CacheRepository) {
	s.cache = cache
}

func (s *Service) CreateDoctor(ctx context.Context, input CreateDoctorInput) (model.Doctor, error) {
	fullName := strings.TrimSpace(input.FullName)
	specialization := strings.TrimSpace(input.Specialization)
	email := strings.ToLower(strings.TrimSpace(input.Email))

	if fullName == "" || !isValidEmail(email) {
		return model.Doctor{}, ErrInvalidDoctorInput
	}

	exists, err := s.repo.ExistsByEmail(ctx, email)
	if err != nil {
		return model.Doctor{}, err
	}
	if exists {
		return model.Doctor{}, ErrDoctorEmailAlreadyUsed
	}

	doctorID, err := id.New()
	if err != nil {
		return model.Doctor{}, err
	}

	doctor := model.Doctor{
		ID:             doctorID,
		FullName:       fullName,
		Specialization: specialization,
		Email:          email,
	}

	if err := s.repo.Create(ctx, doctor); err != nil {
		if errors.Is(err, repository.ErrDoctorEmailAlreadyExists) {
			return model.Doctor{}, ErrDoctorEmailAlreadyUsed
		}

		return model.Doctor{}, err
	}

	if s.cache != nil {
		if err := s.cache.Delete(ctx, "doctors:list"); err != nil {
			log.Printf("failed to invalidate doctors:list after create doctor_id=%s: %v", doctor.ID, err)
		}
	}

	if s.publisher != nil {
		if err := s.publisher.PublishDoctorCreated(ctx, doctor); err != nil {
			log.Printf("failed to publish doctors.created doctor_id=%s: %v", doctor.ID, err)
		}
	}

	return doctor, nil
}

func (s *Service) ListDoctors(ctx context.Context) ([]model.Doctor, error) {
	if s.cache != nil {
		doctors, ok, err := s.cache.GetDoctors(ctx)
		if err != nil {
			log.Printf("doctor list cache read failed: %v", err)
		}
		if ok {
			return doctors, nil
		}
	}

	doctors, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		if err := s.cache.SetDoctors(ctx, doctors); err != nil {
			log.Printf("doctor list cache write failed: %v", err)
		}
	}

	return doctors, nil
}

func (s *Service) GetDoctor(ctx context.Context, id string) (model.Doctor, error) {
	doctorID := strings.TrimSpace(id)
	if s.cache != nil {
		doctor, ok, err := s.cache.GetDoctor(ctx, doctorID)
		if err != nil {
			log.Printf("doctor cache read failed doctor_id=%s: %v", doctorID, err)
		}
		if ok {
			return doctor, nil
		}
	}

	doctor, err := s.repo.GetByID(ctx, doctorID)
	if err != nil {
		if errors.Is(err, repository.ErrDoctorNotFound) {
			return model.Doctor{}, repository.ErrDoctorNotFound
		}

		return model.Doctor{}, err
	}

	if s.cache != nil {
		if err := s.cache.SetDoctor(ctx, doctor); err != nil {
			log.Printf("doctor cache write failed doctor_id=%s: %v", doctor.ID, err)
		}
	}

	return doctor, nil
}

func isValidEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	if err != nil {
		return false
	}

	return address.Address == value
}
