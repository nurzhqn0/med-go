package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	appointmentapp "med-go/internal/appointment/app"
	doctorapp "med-go/internal/doctor/app"
)

func main() {
	doctorService := doctorapp.New(":8081")
	appointmentService := appointmentapp.New(":8082", "http://localhost:8081")

	serverErrors := make(chan error, 2)

	go serve("doctor-service", doctorService.Server, serverErrors)
	go serve("appointment-service", appointmentService.Server, serverErrors)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrors:
		log.Fatalf("server exited with error: %v", err)
	case <-ctx.Done():
		log.Println("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := appointmentService.Server.Shutdown(shutdownCtx); err != nil {
		log.Printf("appointment-service shutdown failed: %v", err)
	}

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
