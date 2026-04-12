package main

import (
	"bufio"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	doctorapp "med-go/internal/doctor/app"
	"med-go/internal/platform/mongodb"
)

func main() {
	loadDotEnv(".env")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	mongoURI := getEnv("MONGODB_URI", "mongodb://localhost:27017")
	mongoDatabaseName := getEnv("MONGODB_DATABASE", "med_go")
	doctorAddress := getEnv("DOCTOR_SERVICE_ADDR", ":8081")

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

	doctorService, err := doctorapp.New(ctx, doctorAddress, mongoClient.Database(mongoDatabaseName))
	if err != nil {
		log.Fatalf("failed to initialize doctor-service: %v", err)
	}

	serverErrors := make(chan error, 1)
	go serve("doctor-service", doctorService.Server, serverErrors)

	select {
	case err := <-serverErrors:
		log.Fatalf("server exited with error: %v", err)
	case <-ctx.Done():
		log.Println("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := doctorService.Server.Shutdown(shutdownCtx); err != nil {
		log.Printf("doctor-service shutdown failed: %v", err)
	}
}

func serve(name string, server *http.Server, serverErrors chan<- error) {
	log.Printf("%s listening on %s", name, server.Addr)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
