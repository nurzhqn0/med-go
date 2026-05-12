package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	appointmentapp "med-go/internal/appointment/app"
	doctorapp "med-go/internal/doctor/app"
	"med-go/internal/notification/jobqueue"
	notificationlogger "med-go/internal/notification/logger"
	"med-go/internal/notification/subscriber"
	"med-go/internal/platform/bootstrap"
	platformredis "med-go/internal/platform/redis"
)

func main() {
	bootstrap.LoadDotEnv(".env")
	config := bootstrap.LoadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startupCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	doctorService, err := doctorapp.New(startupCtx, config.DoctorAddress, config.DoctorDatabaseURL, config.NATSURL, "doctor-service/migrations", config.RedisURL, config.CacheTTL, config.RateLimitRPM)
	if err != nil {
		log.Fatalf("failed to initialize doctor-service: %v", err)
	}
	defer doctorService.Close()

	appointmentService, err := appointmentapp.New(startupCtx, config.AppointmentAddress, config.DoctorServiceTarget, config.AppointmentDatabaseURL, config.NATSURL, "appointment-service/migrations", config.RedisURL, config.CacheTTL, config.RateLimitRPM)
	if err != nil {
		log.Fatalf("failed to initialize appointment-service: %v", err)
	}
	defer appointmentService.Close()

	redisClient, err := platformredis.Connect(startupCtx, config.RedisURL)
	if err != nil {
		log.Printf("Redis unavailable for notification-service, job queue idempotency disabled: %v", err)
	} else {
		defer redisClient.Close()
	}

	queue := jobqueue.NewWithOptions(redisClient, config.GatewayURL, jobqueue.Options{
		PoolSize:    config.WorkerPoolSize,
		MaxAttempts: config.JobMaxRetries,
		Backoffs:    config.JobBackoffs,
	})
	queue.Start(ctx, config.WorkerPoolSize)

	notifications, err := subscriber.New(config.NATSURL, notificationlogger.New(), queue)
	if err != nil {
		log.Fatalf("failed to initialize notification-service: %v", err)
	}

	runtimeCtx, runtimeCancel := context.WithCancel(ctx)
	defer runtimeCancel()

	errs := make(chan error, 2)
	go func() {
		errs <- bootstrap.RunGRPCServices(runtimeCtx,
			bootstrap.Service{Name: "doctor-service", Address: doctorService.Address, Server: doctorService.Server},
			bootstrap.Service{Name: "appointment-service", Address: appointmentService.Address, Server: appointmentService.Server},
		)
	}()

	go func() {
		log.Println("notification-service subscribed to NATS subjects")
		errs <- notifications.Run(runtimeCtx)
	}()

	select {
	case err := <-errs:
		runtimeCancel()
		if err != nil {
			log.Fatalf("service exited with error: %v", err)
		}
	case <-ctx.Done():
		runtimeCancel()
		if err := <-errs; err != nil {
			log.Fatalf("shutdown failed: %v", err)
		}
	}
}
