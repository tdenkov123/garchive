package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdenkov123/file-metadata-service/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://user:pass@localhost:5432/db?sslmode=disable")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, 50051, cfg.GRPCPort)
	assert.Equal(t, "localhost:6379", cfg.RedisAddr)
	assert.Equal(t, "files", cfg.MinioBucket)
	assert.Equal(t, 5*time.Minute, cfg.CacheTTL)
	assert.Equal(t, 15*time.Minute, cfg.PresignTTL)
	assert.Equal(t, []string{"localhost:9092"}, cfg.KafkaBrokers)
}
