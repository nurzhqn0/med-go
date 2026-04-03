package httptransport

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"med-go/internal/appointment/model"
	"med-go/internal/appointment/repository"
	"med-go/internal/appointment/usecase"
)

type createAppointmentRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DoctorID    string `json:"doctor_id"`
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

func NewRouter(doctorServiceBaseURL string, service *usecase.Service) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"doctor_service_url": doctorServiceBaseURL,
			"service":            "appointment-service",
			"status":             "ok",
		})
	})

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
				c.JSON(http.StatusBadGateway, gin.H{"message": err.Error()})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create appointment"})
			}
			return
		}

		c.JSON(http.StatusCreated, appointment)
	})
	appointments.GET("", func(c *gin.Context) {
		appointments, err := service.ListAppointments(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list appointments"})
			return
		}

		c.JSON(http.StatusOK, appointments)
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

		c.JSON(http.StatusOK, appointment)
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

		c.JSON(http.StatusOK, appointment)
	})

	return router
}
