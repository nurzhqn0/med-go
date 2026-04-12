package app

import (
	"context"
	"net/http"
	"time"

	"med-go/internal/doctor/repository"
	httptransport "med-go/internal/doctor/transport/http"
	"med-go/internal/doctor/usecase"
	"med-go/internal/platform/observability"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type App struct {
	Server *http.Server
}

func New(ctx context.Context, addr string, database *mongo.Database) (*App, error) {
	repo, err := repository.NewMongoRepository(ctx, database)
	if err != nil {
		return nil, err
	}

	service := usecase.NewService(repo)
	registry := observability.NewRegistry()
	metrics := observability.NewHTTPMetrics(registry)
	router := httptransport.NewRouter(service, registry, metrics)

	return &App{
		Server: &http.Server{
			Addr:              addr,
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}, nil
}
