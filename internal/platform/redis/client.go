package redis

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

func Connect(ctx context.Context, url string) (*goredis.Client, error) {
	options, err := goredis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	client := goredis.NewClient(options)
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}
