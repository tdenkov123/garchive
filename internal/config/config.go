package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMaxFileSize = 5 * 1024 * 1024 * 1024 // 5 GiB
)

type Config struct {
	AppEnv string
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
	MaxFileSizeBytes  int64

	KafkaBrokers []string
	KafkaTopic   string

	GRPCInsecure         bool
	GRPCEnableReflection bool
	GRPCTLSCertFile      string
	GRPCTLSKeyFile       string

	JWTEnabled   bool
	JWTHMACSecret string
	JWTIssuer    string
	JWTAudience  string

	RateLimitRPS float64
	MetricsPort  int
	ShutdownTimeout time.Duration
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

	maxFileSize, err := getEnvInt64("MAX_FILE_SIZE_BYTES", defaultMaxFileSize)
	if err != nil {
		return nil, err
	}

	metricsPort, err := getEnvInt("METRICS_PORT", 9090)
	if err != nil {
		return nil, err
	}

	rateLimitRPS, err := getEnvFloat("RATE_LIMIT_RPS", 50)
	if err != nil {
		return nil, err
	}

	shutdownTimeout, err := getEnvDuration("SHUTDOWN_TIMEOUT", 15*time.Second)
	if err != nil {
		return nil, err
	}

	appEnv := getEnv("APP_ENV", "development")
	isProd := appEnv == "production"

	postgresDSN := getEnv("POSTGRES_DSN", "postgres://fms:fms@localhost:5432/fms?sslmode=disable")
	minioAccessKey := getEnv("MINIO_ACCESS_KEY", "minioadmin")
	minioSecretKey := getEnv("MINIO_SECRET_KEY", "minioadmin")

	if isProd {
		if err := rejectDefaultSecrets(postgresDSN, minioAccessKey, minioSecretKey); err != nil {
			return nil, err
		}
	}

	cfg := &Config{
		AppEnv:               appEnv,
		GRPCPort:             grpcPort,
		PostgresDSN:          postgresDSN,
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:        getEnv("REDIS_PASSWORD", ""),
		RedisDB:              redisDB,
		CacheTTL:             cacheTTL,
		MinioEndpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:       minioAccessKey,
		MinioSecretKey:       minioSecretKey,
		MinioBucket:          getEnv("MINIO_BUCKET", "files"),
		MinioUseSSL:          getEnvBool("MINIO_USE_SSL", false),
		PresignTTL:           presignTTL,
		MultipartPartSize:    multipartPartSize,
		MaxFileSizeBytes:     maxFileSize,
		KafkaBrokers:         splitCSV(getEnv("KAFKA_BROKERS", "localhost:9092")),
		KafkaTopic:           getEnv("KAFKA_TOPIC", "file.events"),
		GRPCInsecure:         getEnvBool("GRPC_INSECURE", !isProd),
		GRPCEnableReflection: getEnvBool("GRPC_ENABLE_REFLECTION", !isProd),
		GRPCTLSCertFile:      getEnv("GRPC_TLS_CERT", ""),
		GRPCTLSKeyFile:       getEnv("GRPC_TLS_KEY", ""),
		JWTEnabled:           getEnvBool("JWT_ENABLED", false),
		JWTHMACSecret:        getEnv("JWT_HMAC_SECRET", ""),
		JWTIssuer:            getEnv("JWT_ISSUER", "garchive"),
		JWTAudience:          getEnv("JWT_AUDIENCE", "garchive-api"),
		RateLimitRPS:         rateLimitRPS,
		MetricsPort:          metricsPort,
		ShutdownTimeout:      shutdownTimeout,
	}

	if cfg.PostgresDSN == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is required")
	}

	if cfg.JWTEnabled && cfg.JWTHMACSecret == "" {
		return nil, fmt.Errorf("JWT_HMAC_SECRET is required when JWT_ENABLED=true")
	}

	if !cfg.GRPCInsecure && (cfg.GRPCTLSCertFile == "" || cfg.GRPCTLSKeyFile == "") {
		return nil, fmt.Errorf("GRPC_TLS_CERT and GRPC_TLS_KEY are required when GRPC_INSECURE=false")
	}

	return cfg, nil
}

func rejectDefaultSecrets(postgresDSN, minioAccessKey, minioSecretKey string) error {
	if strings.Contains(postgresDSN, "fms:fms@") {
		return fmt.Errorf("default postgres credentials are not allowed in production")
	}
	if minioAccessKey == "minioadmin" || minioSecretKey == "minioadmin" {
		return fmt.Errorf("default minio credentials are not allowed in production")
	}
	return nil
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

func getEnvFloat(key string, fallback float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.ParseFloat(v, 64)
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
