package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"med-go/internal/appointment/model"

	goredis "github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewRedisCache(client *goredis.Client, ttl time.Duration) *RedisCache {
	return &RedisCache{client: client, ttl: ttl}
}

func (c *RedisCache) GetAppointment(ctx context.Context, id string) (model.Appointment, bool, error) {
	var appointment model.Appointment
	ok, err := c.getJSON(ctx, "appointment:"+id, &appointment)
	return appointment, ok, err
}

func (c *RedisCache) SetAppointment(ctx context.Context, appointment model.Appointment) error {
	return c.setJSON(ctx, "appointment:"+appointment.ID, appointment)
}

func (c *RedisCache) GetAppointments(ctx context.Context) ([]model.Appointment, bool, error) {
	var appointments []model.Appointment
	ok, err := c.getJSON(ctx, "appointments:list", &appointments)
	return appointments, ok, err
}

func (c *RedisCache) SetAppointments(ctx context.Context, appointments []model.Appointment) error {
	return c.setJSON(ctx, "appointments:list", appointments)
}

func (c *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if c == nil || c.client == nil || len(keys) == 0 {
		return nil
	}

	return c.client.Del(ctx, keys...).Err()
}

func (c *RedisCache) getJSON(ctx context.Context, key string, dest any) (bool, error) {
	if c == nil || c.client == nil {
		return false, nil
	}

	data, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, goredis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return false, err
	}

	return true, nil
}

func (c *RedisCache) setJSON(ctx context.Context, key string, value any) error {
	if c == nil || c.client == nil {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, key, data, c.ttl).Err()
}
