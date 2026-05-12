package middleware

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func TestRateLimiterReturnsResourceExhaustedAfterLimit(t *testing.T) {
	redisServer, err := miniredis.Run()
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("local TCP listeners are blocked in this environment: %v", err)
		}
		t.Fatalf("could not start miniredis: %v", err)
	}
	defer redisServer.Close()

	client := goredis.NewClient(&goredis.Options{Addr: redisServer.Addr()})
	defer client.Close()

	limiter := NewRateLimiter(client, "test-service", 2)
	interceptor := limiter.UnaryServerInterceptor()
	ctx := peer.NewContext(context.Background(), &peer.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}})
	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Get"}
	handler := func(context.Context, any) (any, error) {
		return "ok", nil
	}

	for i := 0; i < 2; i++ {
		if _, err := interceptor(ctx, nil, info, handler); err != nil {
			t.Fatalf("request %d returned error: %v", i+1, err)
		}
	}

	_, err = interceptor(ctx, nil, info, handler)
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("expected ResourceExhausted, got %s (%v)", status.Code(err), err)
	}
}
