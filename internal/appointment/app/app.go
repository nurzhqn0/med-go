package app

import (
	"net/http"
	"time"

	"med-go/internal/appointment/repository"
	httptransport "med-go/internal/appointment/transport/http"
	"med-go/internal/appointment/usecase"
	"med-go/internal/platform/observability"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type App struct {
	Server *http.Server
}

func New(addr, doctorServiceBaseURL string, database *mongo.Database) *App {
	repo := repository.NewMongoRepository(database)
	doctorClient := NewDoctorClient(doctorServiceBaseURL)
	service := usecase.NewService(repo, doctorClient)
	registry := observability.NewRegistry()
	metrics := observability.NewHTTPMetrics(registry)
	router := httptransport.NewRouter(doctorServiceBaseURL, service, registry, metrics)

	return &App{
		Server: &http.Server{
			Addr:              addr,
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}
