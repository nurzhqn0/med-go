package app

import (
	"context"

	doctorpb "med-go/internal/doctor/proto"
	"med-go/internal/doctor/repository"
	grpctransport "med-go/internal/doctor/transport/grpc"
	"med-go/internal/doctor/usecase"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"google.golang.org/grpc"
)

type App struct {
	Server  *grpc.Server
	Address string
}

func New(ctx context.Context, addr string, database *mongo.Database) (*App, error) {
	repo, err := repository.NewMongoRepository(ctx, database)
	if err != nil {
		return nil, err
	}

	service := usecase.NewService(repo)
	server := grpc.NewServer()
	doctorpb.RegisterDoctorServiceServer(server, grpctransport.NewServer(service))

	return &App{
		Server:  server,
		Address: addr,
	}, nil
}
