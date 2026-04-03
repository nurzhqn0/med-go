package app

import (
	"net/http"
	"time"

	"med-go/internal/doctor/repository"
	httptransport "med-go/internal/doctor/transport/http"
	"med-go/internal/doctor/usecase"
)

type App struct {
	Server *http.Server
}

func New(addr string) *App {
	repo := repository.NewMemoryRepository()
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
