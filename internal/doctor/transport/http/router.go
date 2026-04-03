package httptransport

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"med-go/internal/doctor/repository"
	"med-go/internal/doctor/usecase"
)

type createDoctorRequest struct {
	FullName       string `json:"full_name"`
	Specialization string `json:"specialization"`
	Email          string `json:"email"`
}

func NewRouter(service *usecase.Service) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "doctor-service",
			"status":  "ok",
		})
	})

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

			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create doctor"})
			return
		}

		c.JSON(http.StatusCreated, doctor)
	})
	doctors.GET("", func(c *gin.Context) {
		doctors, err := service.ListDoctors(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to list doctors"})
			return
		}

		c.JSON(http.StatusOK, doctors)
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

		c.JSON(http.StatusOK, doctor)
	})

	return router
}
