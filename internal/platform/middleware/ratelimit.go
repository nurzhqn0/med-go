package middleware

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type RateLimiter struct {
	client  *goredis.Client
	service string
	limit   int
	window  time.Duration
}

func NewRateLimiter(client *goredis.Client, service string, limit int) *RateLimiter {
	if limit <= 0 {
		limit = 100
	}

	return &RateLimiter{
		client:  client,
		service: service,
		limit:   limit,
		window:  time.Minute,
	}
}

func (r *RateLimiter) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if r == nil || r.client == nil {
			return handler(ctx, req)
		}

		allowed, retryAfter, err := r.allow(ctx, clientIP(ctx))
		if err != nil {
			log.Printf("rate limiter failed service=%s method=%s: %v", r.service, info.FullMethod, err)
			return handler(ctx, req)
		}
		if !allowed {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded, retry after %d seconds", int(retryAfter.Seconds())+1)
		}

		return handler(ctx, req)
	}
}

func (r *RateLimiter) allow(ctx context.Context, ip string) (bool, time.Duration, error) {
	now := time.Now().UTC()
	windowStart := now.Add(-r.window)
	key := fmt.Sprintf("rate:%s:%s", r.service, ip)
	member := strconv.FormatInt(now.UnixNano(), 10)

	pipe := r.client.TxPipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart.UnixMilli(), 10))
	pipe.ZAdd(ctx, key, goredis.Z{Score: float64(now.UnixMilli()), Member: member})
	countCmd := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, r.window)
	firstCmd := pipe.ZRangeWithScores(ctx, key, 0, 0)

	if _, err := pipe.Exec(ctx); err != nil {
		return true, 0, err
	}

	count := countCmd.Val()
	if count <= int64(r.limit) {
		return true, 0, nil
	}

	first := firstCmd.Val()
	if len(first) == 0 {
		return false, r.window, nil
	}

	elapsed := now.Sub(time.UnixMilli(int64(first[0].Score)))
	retryAfter := r.window - elapsed
	if retryAfter < 0 {
		retryAfter = 0
	}

	return false, retryAfter, nil
}

func clientIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr == nil {
		return "unknown"
	}

	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil || host == "" {
		return p.Addr.String()
	}

	return host
}
