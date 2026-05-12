package usecase

import (
	"context"
	"errors"
	"testing"

	"med-go/internal/doctor/model"
	"med-go/internal/doctor/repository"
)

type doctorCacheStub struct {
	doctors    map[string]model.Doctor
	list       []model.Doctor
	deleted    []string
	setDoctors int
}

type countingDoctorRepo struct {
	doctor model.Doctor
	gets   int
}

func (r *countingDoctorRepo) Create(context.Context, model.Doctor) error {
	return nil
}

func (r *countingDoctorRepo) List(context.Context) ([]model.Doctor, error) {
	return []model.Doctor{r.doctor}, nil
}

func (r *countingDoctorRepo) GetByID(_ context.Context, id string) (model.Doctor, error) {
	r.gets++
	if id != r.doctor.ID {
		return model.Doctor{}, errors.New("missing doctor")
	}

	return r.doctor, nil
}

func (r *countingDoctorRepo) ExistsByEmail(context.Context, string) (bool, error) {
	return false, nil
}

func (c *doctorCacheStub) GetDoctor(_ context.Context, id string) (model.Doctor, bool, error) {
	doctor, ok := c.doctors[id]
	return doctor, ok, nil
}

func (c *doctorCacheStub) SetDoctor(_ context.Context, doctor model.Doctor) error {
	c.doctors[doctor.ID] = doctor
	return nil
}

func (c *doctorCacheStub) GetDoctors(context.Context) ([]model.Doctor, bool, error) {
	return c.list, c.list != nil, nil
}

func (c *doctorCacheStub) SetDoctors(_ context.Context, doctors []model.Doctor) error {
	c.list = doctors
	c.setDoctors++
	return nil
}

func (c *doctorCacheStub) Delete(_ context.Context, keys ...string) error {
	c.deleted = append(c.deleted, keys...)
	return nil
}

func TestGetDoctorCachesDatabaseResult(t *testing.T) {
	ctx := context.Background()
	repo := &countingDoctorRepo{doctor: model.Doctor{ID: "doc-1", FullName: "Dr. Test", Email: "test@example.com"}}
	cache := &doctorCacheStub{doctors: make(map[string]model.Doctor)}
	service := NewService(repo)
	service.SetCache(cache)

	got, err := service.GetDoctor(ctx, "doc-1")
	if err != nil {
		t.Fatalf("GetDoctor returned error: %v", err)
	}
	if got.ID != "doc-1" {
		t.Fatalf("expected doc-1, got %q", got.ID)
	}
	if _, ok := cache.doctors["doc-1"]; !ok {
		t.Fatalf("expected doctor to be stored in cache")
	}

	if _, err := service.GetDoctor(ctx, "doc-1"); err != nil {
		t.Fatalf("second GetDoctor returned error: %v", err)
	}
	if repo.gets != 1 {
		t.Fatalf("expected one database read, got %d", repo.gets)
	}
}

func TestCreateDoctorInvalidatesListCache(t *testing.T) {
	ctx := context.Background()
	cache := &doctorCacheStub{doctors: make(map[string]model.Doctor)}
	service := NewService(repository.NewMemoryRepository())
	service.SetCache(cache)

	if _, err := service.CreateDoctor(ctx, CreateDoctorInput{
		FullName: "Dr. Cache",
		Email:    "cache@example.com",
	}); err != nil {
		t.Fatalf("CreateDoctor returned error: %v", err)
	}

	if len(cache.deleted) != 1 || cache.deleted[0] != "doctors:list" {
		t.Fatalf("expected doctors:list invalidation, got %#v", cache.deleted)
	}
}
