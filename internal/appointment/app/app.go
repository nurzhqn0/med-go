package app

import (
	"net/http"
	"time"

	"med-go/internal/appointment/repository"
	httptransport "med-go/internal/appointment/transport/http"
	"med-go/internal/appointment/usecase"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type App struct {
	Server *http.Server
}

func New(addr, doctorServiceBaseURL string, database *mongo.Database) *App {
	repo := repository.NewMongoRepository(database)
	doctorClient := NewDoctorClient(doctorServiceBaseURL)
	service := usecase.NewService(repo, doctorClient)
	router := httptransport.NewRouter(doctorServiceBaseURL, service)

	return &App{
		Server: &http.Server{
			Addr:              addr,
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}
