package app

import (
	"net/http"
	"time"

	"med-go/internal/doctor/repository"
	httptransport "med-go/internal/doctor/transport/http"
	"med-go/internal/doctor/usecase"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type App struct {
	Server *http.Server
}

func New(addr string, database *mongo.Database) *App {
	repo := repository.NewMongoRepository(database)
	service := usecase.NewService(repo)
	router := httptransport.NewRouter(service)

	return &App{
		Server: &http.Server{
			Addr:              addr,
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}
