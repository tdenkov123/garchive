package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	GRPCPort int

	PostgresDSN string

	RedisAddr     string
	RedisPassword string
	RedisDB       int
	CacheTTL      time.Duration

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool
	PresignTTL        time.Duration
	MultipartPartSize int64

	KafkaBrokers []string
	KafkaTopic   string
}

func Load() (*Config, error) {
	grpcPort, err := getEnvInt("GRPC_PORT", 50051)
	if err != nil {
		return nil, err
	}

	redisDB, err := getEnvInt("REDIS_DB", 0)
	if err != nil {
		return nil, err
	}

	cacheTTL, err := getEnvDuration("CACHE_TTL", 5*time.Minute)
	if err != nil {
		return nil, err
	}

	presignTTL, err := getEnvDuration("PRESIGN_TTL", 15*time.Minute)
	if err != nil {
		return nil, err
	}

	multipartPartSize, err := getEnvInt64("MULTIPART_PART_SIZE", 5*1024*1024)
	if err != nil {
		return nil, err
	}
	if multipartPartSize < 5*1024*1024 {
		return nil, fmt.Errorf("MULTIPART_PART_SIZE must be at least 5MiB")
	}

	cfg := &Config{
		GRPCPort:       grpcPort,
		PostgresDSN:    getEnv("POSTGRES_DSN", "postgres://fms:fms@localhost:5432/fms?sslmode=disable"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		RedisDB:        redisDB,
		CacheTTL:       cacheTTL,
		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioBucket:    getEnv("MINIO_BUCKET", "files"),
		MinioUseSSL:    getEnvBool("MINIO_USE_SSL", false),
		PresignTTL:        presignTTL,
		MultipartPartSize: multipartPartSize,
		KafkaBrokers:      splitCSV(getEnv("KAFKA_BROKERS", "localhost:9092")),
		KafkaTopic:     getEnv("KAFKA_TOPIC", "file.events"),
	}

	if cfg.PostgresDSN == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) (int64, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return n, nil
}

func getEnvInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return n, nil
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return d, nil
}

func splitCSV(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := s[start:i]
			if part != "" {
				out = append(out, part)
			}
			start = i + 1
		}
	}
	return out
}
