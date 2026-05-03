package bootstrap

import (
	"bufio"
	"context"
	"errors"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
)

type Service struct {
	Name    string
	Address string
	Server  *grpc.Server
}

func LoadDotEnv(path string) {
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

func GetEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func RunGRPCServices(ctx context.Context, services ...Service) error {
	serverErrors := make(chan error, len(services))

	for _, service := range services {
		go serve(service, serverErrors)
	}

	select {
	case err := <-serverErrors:
		return err
	case <-ctx.Done():
		log.Println("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		for _, service := range services {
			service.Server.GracefulStop()
		}
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-shutdownCtx.Done():
		for _, service := range services {
			service.Server.Stop()
		}
	}

	return nil
}

func serve(service Service, serverErrors chan<- error) {
	listener, err := net.Listen("tcp", service.Address)
	if err != nil {
		serverErrors <- err
		return
	}

	log.Printf("%s listening on %s", service.Name, service.Address)

	if err := service.Server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		serverErrors <- err
	}
}
