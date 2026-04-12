package grpctransport

import (
	"context"
	"errors"

	"med-go/internal/doctor/model"
	doctorpb "med-go/internal/doctor/proto"
	"med-go/internal/doctor/repository"
	"med-go/internal/doctor/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service interface {
	CreateDoctor(ctx context.Context, input usecase.CreateDoctorInput) (model.Doctor, error)
	ListDoctors(ctx context.Context) ([]model.Doctor, error)
	GetDoctor(ctx context.Context, id string) (model.Doctor, error)
}

type Server struct {
	doctorpb.UnimplementedDoctorServiceServer
	service Service
}

func NewServer(service Service) *Server {
	return &Server{service: service}
}

func (s *Server) CreateDoctor(ctx context.Context, request *doctorpb.CreateDoctorRequest) (*doctorpb.DoctorResponse, error) {
	doctor, err := s.service.CreateDoctor(ctx, usecase.CreateDoctorInput{
		FullName:       request.GetFullName(),
		Specialization: request.GetSpecialization(),
		Email:          request.GetEmail(),
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrInvalidDoctorInput):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, usecase.ErrDoctorEmailAlreadyUsed):
			return nil, status.Error(codes.AlreadyExists, err.Error())
		default:
			return nil, status.Error(codes.Internal, "failed to create doctor")
		}
	}

	return newDoctorResponse(doctor), nil
}

func (s *Server) GetDoctor(ctx context.Context, request *doctorpb.GetDoctorRequest) (*doctorpb.DoctorResponse, error) {
	doctor, err := s.service.GetDoctor(ctx, request.GetId())
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDoctorNotFound):
			return nil, status.Error(codes.NotFound, err.Error())
		default:
			return nil, status.Error(codes.Internal, "failed to get doctor")
		}
	}

	return newDoctorResponse(doctor), nil
}

func (s *Server) ListDoctors(ctx context.Context, _ *doctorpb.ListDoctorsRequest) (*doctorpb.ListDoctorsResponse, error) {
	doctors, err := s.service.ListDoctors(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list doctors")
	}

	response := &doctorpb.ListDoctorsResponse{
		Doctors: make([]*doctorpb.DoctorResponse, 0, len(doctors)),
	}
	for _, doctor := range doctors {
		response.Doctors = append(response.Doctors, newDoctorResponse(doctor))
	}

	return response, nil
}

func newDoctorResponse(doctor model.Doctor) *doctorpb.DoctorResponse {
	return &doctorpb.DoctorResponse{
		Id:             doctor.ID,
		FullName:       doctor.FullName,
		Specialization: doctor.Specialization,
		Email:          doctor.Email,
	}
}
