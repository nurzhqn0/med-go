package httptransport

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"med-go/internal/appointment/model"
	"med-go/internal/appointment/usecase"
	"med-go/internal/platform/observability"
)

type stubService struct {
	createAppointment func(ctx context.Context, input usecase.CreateAppointmentInput) (model.Appointment, error)
}

func (s stubService) CreateAppointment(ctx context.Context, input usecase.CreateAppointmentInput) (model.Appointment, error) {
	return s.createAppointment(ctx, input)
}

func (s stubService) ListAppointments(context.Context) ([]model.Appointment, error) {
	return nil, nil
}

func (s stubService) GetAppointment(context.Context, string) (model.Appointment, error) {
	return model.Appointment{}, nil
}

func (s stubService) UpdateStatus(context.Context, string, string) (model.Appointment, error) {
	return model.Appointment{}, nil
}

func TestCreateAppointmentReturnsServiceUnavailableWhenDoctorServiceFails(t *testing.T) {
	t.Parallel()

	registry := observability.NewRegistry()
	metrics := observability.NewHTTPMetrics(registry)
	router := NewRouter("http://doctor-service", stubService{
		createAppointment: func(context.Context, usecase.CreateAppointmentInput) (model.Appointment, error) {
			return model.Appointment{}, fmt.Errorf("%w: timeout", usecase.ErrDoctorServiceUnavailable)
		},
	}, registry, metrics)

	request := httptest.NewRequest(http.MethodPost, "/appointments", bytes.NewBufferString(`{"title":"Visit","doctor_id":"doc-1"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}
}
