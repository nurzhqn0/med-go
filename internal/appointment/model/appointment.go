package model

import (
	"errors"
	"time"
)

var ErrInvalidStatus = errors.New("invalid appointment status")

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type Status string

func (s Status) IsValid() bool {
	switch s {
	case StatusNew, StatusInProgress, StatusDone:
		return true
	default:
		return false
	}
}

func ParseStatus(value string) (Status, error) {
	status := Status(value)
	if !status.IsValid() {
		return "", ErrInvalidStatus
	}

	return status, nil
}

func (s Status) CanTransitionTo(next Status) bool {
	if !next.IsValid() {
		return false
	}

	if s == StatusDone && next == StatusNew {
		return false
	}

	return true
}

type Appointment struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	DoctorID    string    `json:"doctor_id"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
