package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	doctorapp "med-go/internal/doctor/app"
	"med-go/internal/platform/bootstrap"
	"med-go/internal/platform/mongodb"
)

func main() {
	bootstrap.LoadDotEnv(".env")
	config := bootstrap.LoadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	connectCtx, connectCancel := context.WithTimeout(ctx, 10*time.Second)
	defer connectCancel()

	mongoClient, err := mongodb.Connect(connectCtx, config.MongoURI)
	if err != nil {
		log.Fatalf("failed to connect to MongoDB: %v", err)
	}
	defer func() {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer disconnectCancel()

		if err := mongoClient.Disconnect(disconnectCtx); err != nil {
			log.Printf("mongo disconnect failed: %v", err)
		}
	}()

	doctorService, err := doctorapp.New(ctx, config.DoctorAddress, mongoClient.Database(config.MongoDatabaseName))
	if err != nil {
		log.Fatalf("failed to initialize doctor-service: %v", err)
	}

	if err := bootstrap.RunGRPCServices(ctx,
		bootstrap.Service{Name: "doctor-service", Address: doctorService.Address, Server: doctorService.Server},
	); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
