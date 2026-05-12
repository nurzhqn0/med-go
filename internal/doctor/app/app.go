package app

import (
	"context"
	"log"
	"time"

	doctorcache "med-go/internal/doctor/cache"
	doctorevent "med-go/internal/doctor/event"
	doctorpb "med-go/internal/doctor/proto"
	"med-go/internal/doctor/repository"
	grpctransport "med-go/internal/doctor/transport/grpc"
	"med-go/internal/doctor/usecase"
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

func New(ctx context.Context, addr, databaseURL, natsURL, migrationsPath, redisURL string, cacheTTL time.Duration, rateLimitRPM int) (*App, error) {
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
	redisClient, err := platformredis.Connect(ctx, redisURL)
	if err != nil {
		log.Printf("Redis unavailable for doctor-service, cache and rate limiting disabled: %v", err)
	} else {
		service.SetCache(doctorcache.NewRedisCache(redisClient, cacheTTL))
		closers = append(closers, func() {
			if err := redisClient.Close(); err != nil {
				log.Printf("doctor-service Redis close failed: %v", err)
			}
		})
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(middleware.NewRateLimiter(redisClient, "doctor-service", rateLimitRPM).UnaryServerInterceptor()))
	doctorpb.RegisterDoctorServiceServer(server, grpctransport.NewServer(service))
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
