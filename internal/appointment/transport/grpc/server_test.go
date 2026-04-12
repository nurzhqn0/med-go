package grpctransport

import (
	"context"
	"fmt"
	"testing"

	"med-go/internal/appointment/model"
	appointmentpb "med-go/internal/appointment/proto"
	"med-go/internal/appointment/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubAppointmentService struct {
	createAppointment func(context.Context, usecase.CreateAppointmentInput) (model.Appointment, error)
}

func (s stubAppointmentService) CreateAppointment(ctx context.Context, input usecase.CreateAppointmentInput) (model.Appointment, error) {
	return s.createAppointment(ctx, input)
}

func (s stubAppointmentService) ListAppointments(context.Context) ([]model.Appointment, error) {
	return nil, nil
}

func (s stubAppointmentService) GetAppointment(context.Context, string) (model.Appointment, error) {
	return model.Appointment{}, nil
}

func (s stubAppointmentService) UpdateStatus(context.Context, string, string) (model.Appointment, error) {
	return model.Appointment{}, nil
}

func TestCreateAppointmentMapsRemoteDoctorMissingToFailedPrecondition(t *testing.T) {
	t.Parallel()

	server := NewServer(stubAppointmentService{
		createAppointment: func(context.Context, usecase.CreateAppointmentInput) (model.Appointment, error) {
			return model.Appointment{}, usecase.ErrDoctorNotFound
		},
	})

	_, err := server.CreateAppointment(context.Background(), &appointmentpb.CreateAppointmentRequest{
		Title:    "Visit",
		DoctorId: "doc-1",
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected %s, got %s", codes.FailedPrecondition, status.Code(err))
	}
}

func TestCreateAppointmentMapsUnavailableDoctorServiceToUnavailable(t *testing.T) {
	t.Parallel()

	server := NewServer(stubAppointmentService{
		createAppointment: func(context.Context, usecase.CreateAppointmentInput) (model.Appointment, error) {
			return model.Appointment{}, fmt.Errorf("%w: connection refused", usecase.ErrDoctorServiceUnavailable)
		},
	})

	_, err := server.CreateAppointment(context.Background(), &appointmentpb.CreateAppointmentRequest{
		Title:    "Visit",
		DoctorId: "doc-1",
	})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("expected %s, got %s", codes.Unavailable, status.Code(err))
	}
}
