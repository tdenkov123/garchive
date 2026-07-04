package middleware_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tdenkov123/file-metadata-service/internal/audit"
	"github.com/tdenkov123/file-metadata-service/internal/grpc/middleware"
)

func TestRecoveryInterceptor(t *testing.T) {
	interceptor := middleware.RecoveryInterceptor(slog.Default())
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, func(ctx context.Context, req any) (any, error) {
		panic("boom")
	})
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestRateLimiter(t *testing.T) {
	rl := middleware.NewRateLimiter(1000, 10)
	interceptor := rl.UnaryServerInterceptor()
	ok := 0
	for i := 0; i < 5; i++ {
		_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
			ok++
			return nil, nil
		})
		require.NoError(t, err)
	}
	require.Equal(t, 5, ok)
}

func TestAuditDeniedInterceptor(t *testing.T) {
	auditLog := audit.New(slog.Default())
	interceptor := middleware.AuditDeniedInterceptor(auditLog)
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.PermissionDenied, "nope")
	})
	require.Error(t, err)
}
