package app

import (
	"context"
	"log"

	doctorevent "med-go/internal/doctor/event"
	doctorpb "med-go/internal/doctor/proto"
	"med-go/internal/doctor/repository"
	grpctransport "med-go/internal/doctor/transport/grpc"
	"med-go/internal/doctor/usecase"
	"med-go/internal/platform/migrations"
	"med-go/internal/platform/postgres"

	"google.golang.org/grpc"
)

type App struct {
	Server  *grpc.Server
	Address string
	closers []func()
}

func New(ctx context.Context, addr, databaseURL, natsURL, migrationsPath string) (*App, error) {
	if err := migrations.Up(databaseURL, migrationsPath); err != nil {
		return nil, err
	}

	pool, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	repo := repository.NewPostgresRepository(pool)
	publisher, err := doctorevent.NewPublisher(natsURL)
	if err != nil {
		log.Printf("NATS unavailable for doctor-service, events will be skipped: %v", err)
	}

	var eventPublisher usecase.EventPublisher = doctorevent.NoopPublisher{}
	closers := []func(){pool.Close}
	if publisher != nil {
		eventPublisher = publisher
		closers = append(closers, func() {
			if err := publisher.Close(); err != nil {
				log.Printf("doctor-service NATS close failed: %v", err)
			}
		})
	}

	service := usecase.NewService(repo, eventPublisher)
	server := grpc.NewServer()
	doctorpb.RegisterDoctorServiceServer(server, grpctransport.NewServer(service))

	return &App{
		Server:  server,
		Address: addr,
		closers: closers,
	}, nil
}

func (a *App) Close() {
	for _, closer := range a.closers {
		closer()
	}
}
