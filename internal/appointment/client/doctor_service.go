package client

import (
	"context"
	"fmt"
	"time"

	doctorpb "med-go/internal/doctor/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type DoctorService struct {
	client doctorpb.DoctorServiceClient
}

func NewDoctorService(target string) (*DoctorService, error) {
	connection, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &DoctorService{
		client: doctorpb.NewDoctorServiceClient(connection),
	}, nil
}

func (c *DoctorService) Exists(ctx context.Context, id string) (bool, error) {
	callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := c.client.GetDoctor(callCtx, &doctorpb.GetDoctorRequest{Id: id})
	if err == nil {
		return true, nil
	}

	statusErr, ok := status.FromError(err)
	if !ok {
		return false, fmt.Errorf("doctor service call failed: %w", err)
	}

	switch statusErr.Code() {
	case codes.NotFound:
		return false, nil
	case codes.Unavailable, codes.DeadlineExceeded:
		return false, fmt.Errorf("doctor service unavailable: %w", err)
	default:
		return false, fmt.Errorf("doctor service request failed with %s: %w", statusErr.Code(), err)
	}
}
