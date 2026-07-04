package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdenkov123/file-metadata-service/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Setenv("APP_ENV", "development")

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, 50051, cfg.GRPCPort)
	assert.Equal(t, "localhost:6379", cfg.RedisAddr)
	assert.Equal(t, "files", cfg.MinioBucket)
	assert.Equal(t, 5*time.Minute, cfg.CacheTTL)
	assert.Equal(t, 15*time.Minute, cfg.PresignTTL)
	assert.Equal(t, []string{"localhost:9092"}, cfg.KafkaBrokers)
	assert.True(t, cfg.GRPCInsecure)
	assert.True(t, cfg.GRPCEnableReflection)
}

func TestLoad_InvalidGRPCPort(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Setenv("GRPC_PORT", "not-a-number")

	_, err := config.Load()
	require.Error(t, err)
}

func TestLoad_MultipartPartSizeTooSmall(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Setenv("MULTIPART_PART_SIZE", "1024")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MULTIPART_PART_SIZE")
}

func TestLoad_ProductionRejectsDefaultSecrets(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("POSTGRES_DSN", "postgres://fms:fms@localhost:5432/fms?sslmode=disable")
	t.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	t.Setenv("MINIO_SECRET_KEY", "minioadmin")

	_, err := config.Load()
	require.Error(t, err)
}

func TestLoad_JWTRequiresSecret(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Setenv("JWT_ENABLED", "true")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "jwt")
}

func TestLoad_TLSRequiredWhenNotInsecure(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://user:pass@localhost:5432/db?sslmode=disable")
	t.Setenv("GRPC_INSECURE", "false")

	_, err := config.Load()
	require.Error(t, err)
}
