package usecase

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"med-go/internal/doctor/model"
	"med-go/internal/doctor/repository"
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

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
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

	doctor := model.Doctor{
		ID:             bson.NewObjectID().Hex(),
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

	return doctor, nil
}

func (s *Service) ListDoctors(ctx context.Context) ([]model.Doctor, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetDoctor(ctx context.Context, id string) (model.Doctor, error) {
	doctor, err := s.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, repository.ErrDoctorNotFound) {
			return model.Doctor{}, repository.ErrDoctorNotFound
		}

		return model.Doctor{}, err
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
