package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tdenkov123/file-metadata-service/internal/audit"
	"github.com/tdenkov123/file-metadata-service/internal/auth"
	"github.com/tdenkov123/file-metadata-service/internal/cache/redis"
	"github.com/tdenkov123/file-metadata-service/internal/config"
	"github.com/tdenkov123/file-metadata-service/internal/events/kafka"
	grpchandler "github.com/tdenkov123/file-metadata-service/internal/grpc"
	grpcmw "github.com/tdenkov123/file-metadata-service/internal/grpc/middleware"
	"github.com/tdenkov123/file-metadata-service/internal/repository/postgres"
	"github.com/tdenkov123/file-metadata-service/internal/service"
	"github.com/tdenkov123/file-metadata-service/internal/storage/minio"
	"github.com/tdenkov123/file-metadata-service/internal/tlsutil"
)

type App struct {
	cfg          *config.Config
	grpcServer   *grpc.Server
	metricsSrv   *http.Server
	pgPool       *pgxpool.Pool
	kafkaPub     *kafka.Publisher
	logger       *slog.Logger
	healthServer *health.Server
}

func New(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*App, error) {
	pool, err := postgres.Connect(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}
	if err := postgres.RunMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	minioClient, err := minio.NewClient(cfg)
	if err != nil {
		pool.Close()
		return nil, err
	}
	if err := minioClient.EnsureBucket(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	cache := redis.NewCache(cfg)
	if err := cache.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	kafkaPub := kafka.NewPublisher(cfg)
	repo := postgres.NewRepository(pool)

	storage := service.NewStorageAdapter(
		minioClient.Bucket,
		func(ctx context.Context, objectKey, contentType string) (string, time.Duration, error) {
			u, ttl, err := minioClient.PresignUpload(ctx, objectKey, contentType)
			if err != nil {
				return "", 0, err
			}
			return u.String(), ttl, nil
		},
		func(ctx context.Context, objectKey string) (string, time.Duration, error) {
			u, ttl, err := minioClient.PresignDownload(ctx, objectKey)
			if err != nil {
				return "", 0, err
			}
			return u.String(), ttl, nil
		},
		minioClient.RemoveObject,
		minioClient.CreateMultipartUpload,
		func(ctx context.Context, objectKey, uploadID string, partNumber int32) (string, time.Duration, error) {
			u, ttl, err := minioClient.PresignUploadPart(ctx, objectKey, uploadID, partNumber)
			if err != nil {
				return "", 0, err
			}
			return u.String(), ttl, nil
		},
		minioClient.CompleteMultipartUpload,
		minioClient.AbortMultipartUpload,
	)

	fileSvc := service.NewFileService(
		repo,
		storage,
		cache,
		service.NewKafkaEventPublisher(kafkaPub),
		cfg.MultipartPartSize,
		cfg.MaxFileSizeBytes,
	)

	auditLog := audit.New(logger)
	rateLimiter := grpcmw.NewRateLimiter(cfg.RateLimitRPS, int(cfg.RateLimitRPS)+1)

	var serverOpts []grpc.ServerOption
	if !cfg.GRPCInsecure {
		creds, err := tlsutil.LoadServerCredentials(cfg.GRPCTLSCertFile, cfg.GRPCTLSKeyFile)
		if err != nil {
			pool.Close()
			return nil, err
		}
		serverOpts = append(serverOpts, grpc.Creds(creds))
	}

	serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(
		grpcmw.RecoveryInterceptor(logger),
		grpcmw.LoggingInterceptor(logger),
		grpcmw.MetricsInterceptor(),
		auth.UnaryServerInterceptor(auth.Config{
			Enabled:    cfg.JWTEnabled,
			HMACSecret: cfg.JWTHMACSecret,
			Issuer:     cfg.JWTIssuer,
			Audience:   cfg.JWTAudience,
		}),
		rateLimiter.UnaryServerInterceptor(),
		grpcmw.AuditDeniedInterceptor(auditLog),
	))

	grpcServer := grpc.NewServer(serverOpts...)
	grpchandler.Register(grpcServer, fileSvc, logger, auditLog)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	if cfg.GRPCEnableReflection {
		reflection.Register(grpcServer)
	}

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsSrv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.MetricsPort),
		Handler:           metricsMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{
		cfg:          cfg,
		grpcServer:   grpcServer,
		metricsSrv:   metricsSrv,
		pgPool:       pool,
		kafkaPub:     kafkaPub,
		logger:       logger,
		healthServer: healthServer,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	go func() {
		a.logger.Info("starting metrics server", "addr", a.metricsSrv.Addr)
		if err := a.metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("metrics server", "error", err)
		}
	}()

	addr := fmt.Sprintf(":%d", a.cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	a.logger.Info("starting gRPC server", "addr", addr, "jwt_enabled", a.cfg.JWTEnabled, "tls", !a.cfg.GRPCInsecure)

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.grpcServer.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		a.logger.Info("shutting down")
		a.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
		defer cancel()

		stopped := make(chan struct{})
		go func() {
			a.grpcServer.GracefulStop()
			close(stopped)
		}()

		select {
		case <-stopped:
		case <-shutdownCtx.Done():
			a.grpcServer.Stop()
		}

		_ = a.metricsSrv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}

func (a *App) Close() {
	if a.healthServer != nil {
		a.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	}
	if a.grpcServer != nil {
		a.grpcServer.GracefulStop()
	}
	if a.metricsSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = a.metricsSrv.Shutdown(ctx)
	}
	if a.kafkaPub != nil {
		_ = a.kafkaPub.Close()
	}
	if a.pgPool != nil {
		a.pgPool.Close()
	}
}
