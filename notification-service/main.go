package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

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

	redisClient, err := platformredis.Connect(ctx, config.RedisURL)
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

	if err := notifications.Run(ctx); err != nil {
		log.Fatalf("notification-service exited with error: %v", err)
	}
}
