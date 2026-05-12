package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	appointmentapp "med-go/internal/appointment/app"
	"med-go/internal/platform/bootstrap"
)

func main() {
	bootstrap.LoadDotEnv(".env")
	config := bootstrap.LoadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startupCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	appointmentService, err := appointmentapp.New(startupCtx, config.AppointmentAddress, config.DoctorServiceTarget, config.DatabaseURL, config.NATSURL, "../../appointment-service/migrations", config.RedisURL, config.CacheTTL, config.RateLimitRPM)
	if err != nil {
		log.Fatalf("failed to initialize appointment-service: %v", err)
	}
	defer appointmentService.Close()

	if err := bootstrap.RunGRPCServices(ctx,
		bootstrap.Service{Name: "appointment-service", Address: appointmentService.Address, Server: appointmentService.Server},
	); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
