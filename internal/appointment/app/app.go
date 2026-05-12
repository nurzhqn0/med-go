package app

import (
	"context"
	"log"
	"time"

	appointmentcache "med-go/internal/appointment/cache"
	"med-go/internal/appointment/client"
	appointmentevent "med-go/internal/appointment/event"
	appointmentpb "med-go/internal/appointment/proto"
	"med-go/internal/appointment/repository"
	grpctransport "med-go/internal/appointment/transport/grpc"
	"med-go/internal/appointment/usecase"
	"med-go/internal/platform/middleware"
	"med-go/internal/platform/migrations"
	"med-go/internal/platform/postgres"
	platformredis "med-go/internal/platform/redis"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type App struct {
	Server  *grpc.Server
	Address string
	closers []func()
}

func New(ctx context.Context, addr, doctorServiceAddress, databaseURL, natsURL, migrationsPath, redisURL string, cacheTTL time.Duration, rateLimitRPM int) (*App, error) {
	if err := migrations.Up(databaseURL, migrationsPath); err != nil {
		return nil, err
	}

	pool, err := postgres.Connect(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	repo := repository.NewPostgresRepository(pool)
	doctorClient, err := client.NewDoctorService(doctorServiceAddress)
	if err != nil {
		pool.Close()
		return nil, err
	}

	publisher, err := appointmentevent.NewPublisher(natsURL)
	if err != nil {
		log.Printf("NATS unavailable for appointment-service, events will be skipped: %v", err)
	}

	var eventPublisher usecase.EventPublisher = appointmentevent.NoopPublisher{}
	closers := []func(){
		pool.Close,
		func() {
			if err := doctorClient.Close(); err != nil {
				log.Printf("doctor-service gRPC client close failed: %v", err)
			}
		},
	}
	if publisher != nil {
		eventPublisher = publisher
		closers = append(closers, func() {
			if err := publisher.Close(); err != nil {
				log.Printf("appointment-service NATS close failed: %v", err)
			}
		})
	}

	service := usecase.NewService(repo, doctorClient, eventPublisher)
	redisClient, err := platformredis.Connect(ctx, redisURL)
	if err != nil {
		log.Printf("Redis unavailable for appointment-service, cache and rate limiting disabled: %v", err)
	} else {
		service.SetCache(appointmentcache.NewRedisCache(redisClient, cacheTTL))
		closers = append(closers, func() {
			if err := redisClient.Close(); err != nil {
				log.Printf("appointment-service Redis close failed: %v", err)
			}
		})
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(middleware.NewRateLimiter(redisClient, "appointment-service", rateLimitRPM).UnaryServerInterceptor()))
	appointmentpb.RegisterAppointmentServiceServer(server, grpctransport.NewServer(service))
	reflection.Register(server)

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
