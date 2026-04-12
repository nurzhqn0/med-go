package httptransport

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	"med-go/internal/appointment/model"
	"med-go/internal/appointment/repository"
	"med-go/internal/appointment/usecase"
	"med-go/internal/platform/observability"
)

type createAppointmentRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DoctorID    string `json:"doctor_id"`
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

type appointmentResponse struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DoctorID    string    `json:"doctor_id"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Service interface {
	CreateAppointment(ctx context.Context, input usecase.CreateAppointmentInput) (model.Appointment, error)
	ListAppointments(ctx context.Context) ([]model.Appointment, error)
	GetAppointment(ctx context.Context, id string) (model.Appointment, error)
	UpdateStatus(ctx context.Context, id string, rawStatus string) (model.Appointment, error)
}

func NewRouter(doctorServiceBaseURL string, service Service, registry *prometheus.Registry, metrics *observability.HTTPMetrics) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(metrics.Middleware("appointment-service"))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"doctor_service_url": doctorServiceBaseURL,
			"service":            "appointment-service",
			"status":             "ok",
		})
	})
	router.GET("/metrics", observability.MetricsHandler(registry))

	appointments := router.Group("/appointments")
	appointments.POST("", func(c *gin.Context) {
		var request createAppointmentRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		appointment, err := service.CreateAppointment(c.Request.Context(), usecase.CreateAppointmentInput{
			Title:       request.Title,
			Description: request.Description,
			DoctorID:    request.DoctorID,
		})
		if err != nil {
			switch {
			case errors.Is(err, usecase.ErrInvalidAppointmentInput):
				c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			case errors.Is(err, usecase.ErrDoctorNotFound):
				c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			case errors.Is(err, usecase.ErrDoctorServiceUnavailable):
				c.JSON(http.StatusServiceUnavailable, gin.H{"message": err.Error()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create appointment"})
			}
			return
		}

		c.JSON(http.StatusCreated, newAppointmentResponse(appointment))
	})
	appointments.GET("", func(c *gin.Context) {
		appointments, err := service.ListAppointments(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list appointments"})
			return
		}

		c.JSON(http.StatusOK, newAppointmentResponses(appointments))
	})
	appointments.GET("/:id", func(c *gin.Context) {
		appointment, err := service.GetAppointment(c.Request.Context(), c.Param("id"))
		if err != nil {
			if errors.Is(err, repository.ErrAppointmentNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
				return
			}

			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get appointment"})
			return
		}

		c.JSON(http.StatusOK, newAppointmentResponse(appointment))
	})
	appointments.PATCH("/:id/status", func(c *gin.Context) {
		var request updateStatusRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		appointment, err := service.UpdateStatus(c.Request.Context(), c.Param("id"), request.Status)
		if err != nil {
			switch {
			case errors.Is(err, repository.ErrAppointmentNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
			case errors.Is(err, model.ErrInvalidStatus):
				c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			case errors.Is(err, usecase.ErrInvalidStatusTransition):
				c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update appointment"})
			}
			return
		}

		c.JSON(http.StatusOK, newAppointmentResponse(appointment))
	})

	return router
}

func newAppointmentResponse(appointment model.Appointment) appointmentResponse {
	return appointmentResponse{
		ID:          appointment.ID,
		Title:       appointment.Title,
		Description: appointment.Description,
		DoctorID:    appointment.DoctorID,
		Status:      string(appointment.Status),
		CreatedAt:   appointment.CreatedAt,
		UpdatedAt:   appointment.UpdatedAt,
	}
}

func newAppointmentResponses(appointments []model.Appointment) []appointmentResponse {
	responses := make([]appointmentResponse, 0, len(appointments))
	for _, appointment := range appointments {
		responses = append(responses, newAppointmentResponse(appointment))
	}

	return responses
}
