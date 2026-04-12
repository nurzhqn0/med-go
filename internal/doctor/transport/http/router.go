package httptransport

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	"med-go/internal/doctor/model"
	"med-go/internal/doctor/repository"
	"med-go/internal/doctor/usecase"
	"med-go/internal/platform/observability"
)

type createDoctorRequest struct {
	FullName       string `json:"full_name"`
	Specialization string `json:"specialization"`
	Email          string `json:"email"`
}

type doctorResponse struct {
	ID             string `json:"id"`
	FullName       string `json:"full_name"`
	Specialization string `json:"specialization"`
	Email          string `json:"email"`
}

type Service interface {
	CreateDoctor(ctx context.Context, input usecase.CreateDoctorInput) (model.Doctor, error)
	ListDoctors(ctx context.Context) ([]model.Doctor, error)
	GetDoctor(ctx context.Context, id string) (model.Doctor, error)
}

func NewRouter(service Service, registry *prometheus.Registry, metrics *observability.HTTPMetrics) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(metrics.Middleware("doctor-service"))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "doctor-service",
			"status":  "ok",
		})
	})
	router.GET("/metrics", observability.MetricsHandler(registry))

	doctors := router.Group("/doctors")
	doctors.POST("", func(c *gin.Context) {
		var request createDoctorRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		doctor, err := service.CreateDoctor(c.Request.Context(), usecase.CreateDoctorInput{
			FullName:       request.FullName,
			Specialization: request.Specialization,
			Email:          request.Email,
		})
		if err != nil {
			if errors.Is(err, usecase.ErrInvalidDoctorInput) {
				c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
				return
			}
			if errors.Is(err, usecase.ErrDoctorEmailAlreadyUsed) {
				c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create doctor"})
			return
		}

		c.JSON(http.StatusCreated, newDoctorResponse(doctor))
	})
	doctors.GET("", func(c *gin.Context) {
		doctors, err := service.ListDoctors(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list doctors"})
			return
		}

		c.JSON(http.StatusOK, newDoctorResponses(doctors))
	})
	doctors.GET("/:id", func(c *gin.Context) {
		doctor, err := service.GetDoctor(c.Request.Context(), c.Param("id"))
		if err != nil {
			if errors.Is(err, repository.ErrDoctorNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get doctor"})
			return
		}

		c.JSON(http.StatusOK, newDoctorResponse(doctor))
	})

	return router
}

func newDoctorResponse(doctor model.Doctor) doctorResponse {
	return doctorResponse{
		ID:             doctor.ID,
		FullName:       doctor.FullName,
		Specialization: doctor.Specialization,
		Email:          doctor.Email,
	}
}

func newDoctorResponses(doctors []model.Doctor) []doctorResponse {
	responses := make([]doctorResponse, 0, len(doctors))
	for _, doctor := range doctors {
		responses = append(responses, newDoctorResponse(doctor))
	}

	return responses
}
