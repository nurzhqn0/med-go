package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var ErrDoctorServiceUnavailable = errors.New("doctor service unavailable")

type DoctorService struct {
	baseURL    string
	httpClient *http.Client
}

func NewDoctorService(baseURL string) *DoctorService {
	return &DoctorService{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (c *DoctorService) Exists(ctx context.Context, id string) (bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/doctors/%s", c.baseURL, id), nil)
	if err != nil {
		return false, err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return false, fmt.Errorf("%w: request failed: %v", ErrDoctorServiceUnavailable, err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	case http.StatusBadRequest:
		return false, fmt.Errorf("doctor-service rejected lookup for doctor id %q", id)
	default:
		if response.StatusCode >= http.StatusInternalServerError {
			return false, fmt.Errorf("%w: status %d", ErrDoctorServiceUnavailable, response.StatusCode)
		}

		return false, fmt.Errorf("unexpected doctor-service response status %d", response.StatusCode)
	}
}
