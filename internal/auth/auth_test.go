package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/tdenkov123/file-metadata-service/internal/auth"
)

func TestGenerateDevTokenAndInterceptor(t *testing.T) {
	secret := "test-secret"
	token, err := auth.GenerateDevToken(secret, "garchive", "garchive-api", "user-1", time.Hour)
	require.NoError(t, err)

	interceptor := auth.UnaryServerInterceptor(auth.Config{
		Enabled:    true,
		HMACSecret: secret,
		Issuer:     "garchive",
		Audience:   "garchive-api",
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	called := false
	_, err = interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/file.v1.FileService/GetFile"}, func(ctx context.Context, req any) (any, error) {
		called = true
		ownerID, ok := auth.OwnerIDFromContext(ctx)
		require.True(t, ok)
		require.Equal(t, "user-1", ownerID)
		return nil, nil
	})
	require.NoError(t, err)
	require.True(t, called)
}

func TestInterceptor_MissingToken(t *testing.T) {
	interceptor := auth.UnaryServerInterceptor(auth.Config{
		Enabled:    true,
		HMACSecret: "secret",
		Issuer:     "garchive",
		Audience:   "garchive-api",
	})

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/file.v1.FileService/GetFile"}, func(ctx context.Context, req any) (any, error) {
		return nil, nil
	})
	require.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestResolveOwnerID_Mismatch(t *testing.T) {
	ctx := auth.WithOwnerID(context.Background(), "user-1")
	_, err := auth.ResolveOwnerID(ctx, "user-2")
	require.Error(t, err)
}

func TestResolveOwnerID_FromRequestWhenAuthDisabled(t *testing.T) {
	ownerID, err := auth.ResolveOwnerID(context.Background(), "user-1")
	require.NoError(t, err)
	require.Equal(t, "user-1", ownerID)
}
