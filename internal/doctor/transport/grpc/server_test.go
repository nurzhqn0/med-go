package grpctransport

import (
	"context"
	"testing"

	"med-go/internal/doctor/model"
	doctorpb "med-go/internal/doctor/proto"
	"med-go/internal/doctor/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubDoctorService struct {
	createDoctor func(context.Context, usecase.CreateDoctorInput) (model.Doctor, error)
}

func (s stubDoctorService) CreateDoctor(ctx context.Context, input usecase.CreateDoctorInput) (model.Doctor, error) {
	return s.createDoctor(ctx, input)
}

func (s stubDoctorService) ListDoctors(context.Context) ([]model.Doctor, error) {
	return nil, nil
}

func (s stubDoctorService) GetDoctor(context.Context, string) (model.Doctor, error) {
	return model.Doctor{}, nil
}

func TestCreateDoctorMapsDuplicateEmailToAlreadyExists(t *testing.T) {
	t.Parallel()

	server := NewServer(stubDoctorService{
		createDoctor: func(context.Context, usecase.CreateDoctorInput) (model.Doctor, error) {
			return model.Doctor{}, usecase.ErrDoctorEmailAlreadyUsed
		},
	})

	_, err := server.CreateDoctor(context.Background(), &doctorpb.CreateDoctorRequest{
		FullName: "Dr. Test",
		Email:    "test@example.com",
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("expected %s, got %s", codes.AlreadyExists, status.Code(err))
	}
}

func TestCreateDoctorMapsInvalidInputToInvalidArgument(t *testing.T) {
	t.Parallel()

	server := NewServer(stubDoctorService{
		createDoctor: func(context.Context, usecase.CreateDoctorInput) (model.Doctor, error) {
			return model.Doctor{}, usecase.ErrInvalidDoctorInput
		},
	})

	_, err := server.CreateDoctor(context.Background(), &doctorpb.CreateDoctorRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected %s, got %s", codes.InvalidArgument, status.Code(err))
	}
}
