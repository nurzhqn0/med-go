package usecase

import (
	"context"
	"errors"
	"testing"

	"med-go/internal/doctor/repository"
)

func TestCreateDoctorAllowsEmptySpecialization(t *testing.T) {
	t.Parallel()

	service := NewService(repository.NewMemoryRepository())

	doctor, err := service.CreateDoctor(context.Background(), CreateDoctorInput{
		FullName: "Dr. Alice Brown",
		Email:    "alice.brown@example.com",
	})
	if err != nil {
		t.Fatalf("CreateDoctor returned error: %v", err)
	}
	if doctor.Specialization != "" {
		t.Fatalf("expected empty specialization, got %q", doctor.Specialization)
	}
}

func TestCreateDoctorRejectsDuplicateEmailCaseInsensitive(t *testing.T) {
	t.Parallel()

	service := NewService(repository.NewMemoryRepository())

	if _, err := service.CreateDoctor(context.Background(), CreateDoctorInput{
		FullName: "Dr. Alice Brown",
		Email:    "Alice.Brown@Example.com",
	}); err != nil {
		t.Fatalf("CreateDoctor returned error: %v", err)
	}

	_, err := service.CreateDoctor(context.Background(), CreateDoctorInput{
		FullName: "Dr. Bob Smith",
		Email:    "alice.brown@example.com",
	})
	if !errors.Is(err, ErrDoctorEmailAlreadyUsed) {
		t.Fatalf("expected ErrDoctorEmailAlreadyUsed, got %v", err)
	}
}
