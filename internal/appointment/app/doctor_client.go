package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type DoctorClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewDoctorClient(baseURL string) *DoctorClient {
	return &DoctorClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (c *DoctorClient) Exists(ctx context.Context, id string) (bool, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/doctors/%s", c.baseURL, id), nil)
	if err != nil {
		return false, err
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, errors.New("unexpected doctor-service response")
	}
}
