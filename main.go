package main

import (
	"bufio"
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	appointmentapp "med-go/internal/appointment/app"
	doctorapp "med-go/internal/doctor/app"
	"med-go/internal/platform/mongodb"

	"google.golang.org/grpc"
)

func main() {
	loadDotEnv(".env")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mongoURI := getEnv("MONGODB_URI", "mongodb://localhost:27017")
	mongoDatabaseName := getEnv("MONGODB_DATABASE", "med_go")
	doctorAddress := getEnv("DOCTOR_SERVICE_ADDR", ":8081")
	appointmentAddress := getEnv("APPOINTMENT_SERVICE_ADDR", ":8082")
	doctorServiceTarget := getEnv("DOCTOR_SERVICE_GRPC_TARGET", "127.0.0.1:8081")

	connectCtx, connectCancel := context.WithTimeout(ctx, 10*time.Second)
	defer connectCancel()

	mongoClient, err := mongodb.Connect(connectCtx, mongoURI)
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

	database := mongoClient.Database(mongoDatabaseName)

	doctorService, err := doctorapp.New(ctx, doctorAddress, database)
	if err != nil {
		log.Fatalf("failed to initialize doctor-service: %v", err)
	}
	appointmentService, err := appointmentapp.New(appointmentAddress, doctorServiceTarget, database)
	if err != nil {
		log.Fatalf("failed to initialize appointment-service: %v", err)
	}

	serverErrors := make(chan error, 2)

	go serve("doctor-service", doctorService.Address, doctorService.Server, serverErrors)
	go serve("appointment-service", appointmentService.Address, appointmentService.Server, serverErrors)

	select {
	case err := <-serverErrors:
		log.Fatalf("server exited with error: %v", err)
	case <-ctx.Done():
		log.Println("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		doctorService.Server.GracefulStop()
		appointmentService.Server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-shutdownCtx.Done():
		appointmentService.Server.Stop()
		doctorService.Server.Stop()
	}
}

func serve(name, addr string, server *grpc.Server, serverErrors chan<- error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		serverErrors <- err
		return
	}

	log.Printf("%s listening on %s", name, addr)

	if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		serverErrors <- err
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}

		_ = os.Setenv(key, value)
	}
}
