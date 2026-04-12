package repository

import (
	"context"
	"errors"
	"sync"

	"med-go/internal/doctor/model"
)

var ErrDoctorNotFound = errors.New("doctor not found")
var ErrDoctorEmailAlreadyExists = errors.New("doctor email already exists")

type Repository interface {
	Create(ctx context.Context, doctor model.Doctor) error
	List(ctx context.Context) ([]model.Doctor, error)
	GetByID(ctx context.Context, id string) (model.Doctor, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

type MemoryRepository struct {
	mu      sync.RWMutex
	doctors map[string]model.Doctor
	byEmail map[string]string
	order   []string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		doctors: make(map[string]model.Doctor),
		byEmail: make(map[string]string),
	}
}

func (r *MemoryRepository) Create(_ context.Context, doctor model.Doctor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byEmail[doctor.Email]; exists {
		return ErrDoctorEmailAlreadyExists
	}

	r.doctors[doctor.ID] = doctor
	r.byEmail[doctor.Email] = doctor.ID
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

func (r *MemoryRepository) ExistsByEmail(_ context.Context, email string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.byEmail[email]

	return exists, nil
}
