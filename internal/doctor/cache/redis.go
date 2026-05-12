package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"med-go/internal/doctor/model"

	goredis "github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewRedisCache(client *goredis.Client, ttl time.Duration) *RedisCache {
	return &RedisCache{client: client, ttl: ttl}
}

func (c *RedisCache) GetDoctor(ctx context.Context, id string) (model.Doctor, bool, error) {
	var doctor model.Doctor
	ok, err := c.getJSON(ctx, "doctor:"+id, &doctor)
	return doctor, ok, err
}

func (c *RedisCache) SetDoctor(ctx context.Context, doctor model.Doctor) error {
	return c.setJSON(ctx, "doctor:"+doctor.ID, doctor)
}

func (c *RedisCache) GetDoctors(ctx context.Context) ([]model.Doctor, bool, error) {
	var doctors []model.Doctor
	ok, err := c.getJSON(ctx, "doctors:list", &doctors)
	return doctors, ok, err
}

func (c *RedisCache) SetDoctors(ctx context.Context, doctors []model.Doctor) error {
	return c.setJSON(ctx, "doctors:list", doctors)
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
