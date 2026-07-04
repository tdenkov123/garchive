//go:build integration

package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/tdenkov123/file-metadata-service/internal/cache/redis"
	"github.com/tdenkov123/file-metadata-service/internal/config"
	"github.com/tdenkov123/file-metadata-service/internal/domain"
)

func TestCache_SetGetInvalidate(t *testing.T) {
	ctx := context.Background()

	container, err := tcredis.Run(ctx, "redis:8.8.0-alpine")
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	addr, err := container.Endpoint(ctx, "")
	require.NoError(t, err)

	cache := redis.NewCache(&config.Config{
		RedisAddr: addr,
		CacheTTL:  time.Minute,
	})
	require.NoError(t, cache.Ping(ctx))

	file := domain.FileMetadata{
		ID:        uuid.NewString(),
		OwnerID:   "user-1",
		Status:    domain.FileStatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	require.NoError(t, cache.SetFile(ctx, file))

	got, ok, err := cache.GetFile(ctx, file.ID)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, file.ID, got.ID)

	require.NoError(t, cache.InvalidateFile(ctx, file.ID))
	_, ok, err = cache.GetFile(ctx, file.ID)
	require.NoError(t, err)
	require.False(t, ok)
}
