package repository

import (
	"context"
	"errors"
	"sync"

	"med-go/internal/doctor/model"
)

var ErrDoctorNotFound = errors.New("doctor not found")

type Repository interface {
	Create(ctx context.Context, doctor model.Doctor) error
	List(ctx context.Context) ([]model.Doctor, error)
	GetByID(ctx context.Context, id string) (model.Doctor, error)
}

type MemoryRepository struct {
	mu      sync.RWMutex
	doctors map[string]model.Doctor
	order   []string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		doctors: make(map[string]model.Doctor),
	}
}

func (r *MemoryRepository) Create(_ context.Context, doctor model.Doctor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.doctors[doctor.ID] = doctor
	r.order = append(r.order, doctor.ID)

	return nil
}

func (r *MemoryRepository) List(_ context.Context) ([]model.Doctor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	doctors := make([]model.Doctor, 0, len(r.order))
	for _, id := range r.order {
		doctors = append(doctors, r.doctors[id])
	}

	return doctors, nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id string) (model.Doctor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	doctor, ok := r.doctors[id]
	if !ok {
		return model.Doctor{}, ErrDoctorNotFound
	}

	return doctor, nil
}
