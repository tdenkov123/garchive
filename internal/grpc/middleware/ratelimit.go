package middleware

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tdenkov123/file-metadata-service/internal/auth"
)

type RateLimiter struct {
	rps   rate.Limit
	burst int
	mu    sync.Mutex
	limits map[string]*rate.Limiter
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
	if burst < 1 {
		burst = 1
	}
	return &RateLimiter{
		rps:    rate.Limit(rps),
		burst:  burst,
		limits: make(map[string]*rate.Limiter),
	}
}

func (rl *RateLimiter) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		key := "anonymous"
		if ownerID, ok := auth.OwnerIDFromContext(ctx); ok {
			key = ownerID
		}

		if !rl.get(key).Allow() {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}

func (rl *RateLimiter) get(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	l, ok := rl.limits[key]
	if !ok {
		l = rate.NewLimiter(rl.rps, rl.burst)
		rl.limits[key] = l
	}
	return l
}
