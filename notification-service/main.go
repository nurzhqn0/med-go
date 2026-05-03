package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"med-go/internal/notification/subscriber"
	"med-go/internal/platform/bootstrap"
)

func main() {
	bootstrap.LoadDotEnv(".env")
	config := bootstrap.LoadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	notifications, err := subscriber.New(config.NATSURL)
	if err != nil {
		log.Fatalf("failed to initialize notification-service: %v", err)
	}

	if err := notifications.Run(ctx); err != nil {
		log.Fatalf("notification-service exited with error: %v", err)
	}
}
