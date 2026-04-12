package app

import (
	"med-go/internal/appointment/client"
	appointmentpb "med-go/internal/appointment/proto"
	"med-go/internal/appointment/repository"
	grpctransport "med-go/internal/appointment/transport/grpc"
	"med-go/internal/appointment/usecase"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"google.golang.org/grpc"
)

type App struct {
	Server  *grpc.Server
	Address string
}

func New(addr, doctorServiceAddress string, database *mongo.Database) (*App, error) {
	repo := repository.NewMongoRepository(database)
	doctorClient, err := client.NewDoctorService(doctorServiceAddress)
	if err != nil {
		return nil, err
	}

	service := usecase.NewService(repo, doctorClient)
	server := grpc.NewServer()
	appointmentpb.RegisterAppointmentServiceServer(server, grpctransport.NewServer(service))

	return &App{
		Server:  server,
		Address: addr,
	}, nil
}
