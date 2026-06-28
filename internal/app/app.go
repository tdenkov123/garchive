package app

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tdenkov123/file-metadata-service/internal/cache/redis"
	"github.com/tdenkov123/file-metadata-service/internal/config"
	"github.com/tdenkov123/file-metadata-service/internal/events/kafka"
	grpchandler "github.com/tdenkov123/file-metadata-service/internal/grpc"
	"github.com/tdenkov123/file-metadata-service/internal/repository/postgres"
	"github.com/tdenkov123/file-metadata-service/internal/service"
	"github.com/tdenkov123/file-metadata-service/internal/storage/minio"
)

type App struct {
	cfg        *config.Config
	grpcServer *grpc.Server
	pgPool     *pgxpool.Pool
	kafkaPub   *kafka.Publisher
	logger     *slog.Logger
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
	)

	grpcServer := grpc.NewServer()
	grpchandler.Register(grpcServer, fileSvc, logger)
	reflection.Register(grpcServer)

	return &App{
		cfg:        cfg,
		grpcServer: grpcServer,
		pgPool:     pool,
		kafkaPub:   kafkaPub,
		logger:     logger,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", a.cfg.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	a.logger.Info("starting gRPC server", "addr", addr)

	errCh := make(chan error, 1)
	go func() {
		errCh <- a.grpcServer.Serve(lis)
	}()

	select {
	case <-ctx.Done():
		a.logger.Info("shutting down")
		a.grpcServer.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}

func (a *App) Close() {
	if a.grpcServer != nil {
		a.grpcServer.GracefulStop()
	}
	if a.kafkaPub != nil {
		_ = a.kafkaPub.Close()
	}
	if a.pgPool != nil {
		a.pgPool.Close()
	}
}
