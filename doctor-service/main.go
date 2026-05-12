package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	doctorapp "med-go/internal/doctor/app"
	"med-go/internal/platform/bootstrap"
)

func main() {
	bootstrap.LoadDotEnv(".env")
	config := bootstrap.LoadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startupCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	doctorService, err := doctorapp.New(startupCtx, config.DoctorAddress, config.DatabaseURL, config.NATSURL, "migrations", config.RedisURL, config.CacheTTL, config.RateLimitRPM)
	if err != nil {
		log.Fatalf("failed to initialize doctor-service: %v", err)
	}
	defer doctorService.Close()

	if err := bootstrap.RunGRPCServices(ctx,
		bootstrap.Service{Name: "doctor-service", Address: doctorService.Address, Server: doctorService.Server},
	); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
