package grpctransport

import (
	"context"
	"errors"

	"med-go/internal/appointment/model"
	appointmentpb "med-go/internal/appointment/proto"
	"med-go/internal/appointment/repository"
	"med-go/internal/appointment/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service interface {
	CreateAppointment(ctx context.Context, input usecase.CreateAppointmentInput) (model.Appointment, error)
	ListAppointments(ctx context.Context) ([]model.Appointment, error)
	GetAppointment(ctx context.Context, id string) (model.Appointment, error)
	UpdateStatus(ctx context.Context, id string, rawStatus string) (model.Appointment, error)
}

type Server struct {
	appointmentpb.UnimplementedAppointmentServiceServer
	service Service
}

func NewServer(service Service) *Server {
	return &Server{service: service}
}

func (s *Server) CreateAppointment(ctx context.Context, request *appointmentpb.CreateAppointmentRequest) (*appointmentpb.AppointmentResponse, error) {
	appointment, err := s.service.CreateAppointment(ctx, usecase.CreateAppointmentInput{
		Title:       request.GetTitle(),
		Description: request.GetDescription(),
		DoctorID:    request.GetDoctorId(),
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidAppointmentInput):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, usecase.ErrDoctorNotFound):
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		case errors.Is(err, usecase.ErrDoctorServiceUnavailable):
			return nil, status.Error(codes.Unavailable, err.Error())
		default:
			return nil, status.Error(codes.Internal, "failed to create appointment")
		}
	}

	return newAppointmentResponse(appointment), nil
}

func (s *Server) GetAppointment(ctx context.Context, request *appointmentpb.GetAppointmentRequest) (*appointmentpb.AppointmentResponse, error) {
	appointment, err := s.service.GetAppointment(ctx, request.GetId())
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrAppointmentNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		default:
			return nil, status.Error(codes.Internal, "failed to get appointment")
		}
	}

	return newAppointmentResponse(appointment), nil
}

func (s *Server) ListAppointments(ctx context.Context, _ *appointmentpb.ListAppointmentsRequest) (*appointmentpb.ListAppointmentsResponse, error) {
	appointments, err := s.service.ListAppointments(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list appointments")
	}

	response := &appointmentpb.ListAppointmentsResponse{
		Appointments: make([]*appointmentpb.AppointmentResponse, 0, len(appointments)),
	}
	for _, appointment := range appointments {
		response.Appointments = append(response.Appointments, newAppointmentResponse(appointment))
	}

	return response, nil
}

func (s *Server) UpdateAppointmentStatus(ctx context.Context, request *appointmentpb.UpdateStatusRequest) (*appointmentpb.AppointmentResponse, error) {
	appointment, err := s.service.UpdateStatus(ctx, request.GetId(), request.GetStatus())
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrAppointmentNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		case errors.Is(err, model.ErrInvalidStatus):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, usecase.ErrInvalidStatusTransition):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		default:
			return nil, status.Error(codes.Internal, "failed to update appointment")
		}
	}

	return newAppointmentResponse(appointment), nil
}

func newAppointmentResponse(appointment model.Appointment) *appointmentpb.AppointmentResponse {
	return &appointmentpb.AppointmentResponse{
		Id:          appointment.ID,
		Title:       appointment.Title,
		Description: appointment.Description,
		DoctorId:    appointment.DoctorID,
		Status:      string(appointment.Status),
		CreatedAt:   appointment.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		UpdatedAt:   appointment.UpdatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
	}
}
