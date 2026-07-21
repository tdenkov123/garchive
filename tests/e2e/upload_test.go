//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	filev1 "github.com/tdenkov123/file-metadata-service/api/gen/file/v1"
	"github.com/tdenkov123/file-metadata-service/internal/auth"
	"github.com/tdenkov123/file-metadata-service/internal/config"
)

func setupEnv(t *testing.T) (grpcAddr string, jwtSecret string) {
	t.Helper()
	ctx := context.Background()

	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("fms"),
		tcpostgres.WithUsername("fms"),
		tcpostgres.WithPassword("fms"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })
	pgDSN, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	rd, err := tcredis.Run(ctx, "redis:8.8.0-alpine")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rd.Terminate(ctx) })
	redisAddr, err := rd.Endpoint(ctx, "")
	require.NoError(t, err)

	kf, err := tckafka.Run(ctx, "confluentinc/confluent-local:7.5.0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = kf.Terminate(ctx) })
	brokers, err := kf.Brokers(ctx)
	require.NoError(t, err)

	minioC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "minio/minio:RELEASE.2025-09-07T16-13-09Z",
			ExposedPorts: []string{"9000/tcp"},
			Env: map[string]string{
				"MINIO_ROOT_USER":     "minioadmin",
				"MINIO_ROOT_PASSWORD": "minioadmin",
			},
			Cmd:        []string{"server", "/data"},
			WaitingFor: wait.ForHTTP("/minio/health/ready").WithPort("9000/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = minioC.Terminate(ctx) })
	minioHost, err := minioC.Host(ctx)
	require.NoError(t, err)
	minioPort, err := minioC.MappedPort(ctx, "9000/tcp")
	require.NoError(t, err)

	grpcPort := freePort(t)
	metricsPort := freePort(t)
	jwtSecret = "e2e-test-secret"

	t.Setenv("APP_ENV", "development")
	t.Setenv("GRPC_PORT", itoa(grpcPort))
	t.Setenv("METRICS_PORT", itoa(metricsPort))
	t.Setenv("POSTGRES_DSN", pgDSN)
	t.Setenv("REDIS_ADDR", redisAddr)
	t.Setenv("MINIO_ENDPOINT", minioHost+":"+minioPort.Port())
	t.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	t.Setenv("MINIO_SECRET_KEY", "minioadmin")
	t.Setenv("MINIO_BUCKET", "files")
	t.Setenv("MINIO_USE_SSL", "false")
	t.Setenv("KAFKA_BROKERS", strings.Join(brokers, ","))
	t.Setenv("KAFKA_TOPIC", "file.events")
	t.Setenv("JWT_ENABLED", "false")
	t.Setenv("GRPC_INSECURE", "true")

	cfg, err := config.Load()
	require.NoError(t, err)
	startApp(t, cfg)

	return fmt.Sprintf("127.0.0.1:%d", grpcPort), jwtSecret
}

func dial(t *testing.T, addr string, token string) filev1.FileServiceClient {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	if token != "" {
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
		_ = ctx
	}
	return filev1.NewFileServiceClient(conn)
}

func TestE2E_SingleUploadFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e in short mode")
	}
	addr, _ := setupEnv(t)
	client := dial(t, addr, "")

	payload := []byte("hello garchive")
	createResp, err := client.CreateUpload(context.Background(), &filev1.CreateUploadRequest{
		OwnerId:      "user-1",
		OriginalName: "hello.txt",
		ContentType:  "text/plain",
		SizeBytes:    int64(len(payload)),
	})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, createResp.GetUploadUrl(), bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "text/plain")
	httpResp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, httpResp.Body)
	httpResp.Body.Close()
	require.True(t, httpResp.StatusCode >= 200 && httpResp.StatusCode < 300)

	checksum := strings.Repeat("e", 64)
	_, err = client.ConfirmUpload(context.Background(), &filev1.ConfirmUploadRequest{
		Id:             createResp.GetMetadata().GetId(),
		OwnerId:        "user-1",
		ChecksumSha256: checksum,
	})
	require.NoError(t, err)

	dl, err := client.GetDownloadURL(context.Background(), &filev1.GetDownloadURLRequest{
		Id:      createResp.GetMetadata().GetId(),
		OwnerId: "user-1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, dl.GetDownloadUrl())

	_, err = client.DeleteFile(context.Background(), &filev1.DeleteFileRequest{
		Id:      createResp.GetMetadata().GetId(),
		OwnerId: "user-1",
	})
	require.NoError(t, err)
}

func TestE2E_DoubleConfirmUpload(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e in short mode")
	}
	addr, _ := setupEnv(t)
	client := dial(t, addr, "")

	createResp, err := client.CreateUpload(context.Background(), &filev1.CreateUploadRequest{
		OwnerId:      "user-1",
		OriginalName: "dup.txt",
		ContentType:  "text/plain",
		SizeBytes:    10,
	})
	require.NoError(t, err)

	checksum := strings.Repeat("a", 64)
	_, err = client.ConfirmUpload(context.Background(), &filev1.ConfirmUploadRequest{
		Id: createResp.GetMetadata().GetId(), OwnerId: "user-1", ChecksumSha256: checksum,
	})
	require.NoError(t, err)

	_, err = client.ConfirmUpload(context.Background(), &filev1.ConfirmUploadRequest{
		Id: createResp.GetMetadata().GetId(), OwnerId: "user-1", ChecksumSha256: checksum,
	})
	require.Error(t, err)
}

func TestE2E_MultipartUploadFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e in short mode")
	}
	addr, _ := setupEnv(t)
	client := dial(t, addr, "")

	partSize := int64(5 * 1024 * 1024)
	size := partSize + 1024
	created, err := client.CreateMultipartUpload(context.Background(), &filev1.CreateMultipartUploadRequest{
		OwnerId:      "user-1",
		OriginalName: "big.bin",
		ContentType:  "application/octet-stream",
		SizeBytes:    size,
	})
	require.NoError(t, err)
	require.Equal(t, int32(2), created.GetTotalParts())

	for _, part := range []int32{1, 2} {
		partURL, err := client.GetPartUploadURL(context.Background(), &filev1.GetPartUploadURLRequest{
			Id:         created.GetMetadata().GetId(),
			OwnerId:    "user-1",
			PartNumber: part,
		})
		require.NoError(t, err)

		body := bytes.Repeat([]byte("x"), int(partURL.GetPartSizeBytes()))
		req, err := http.NewRequest(http.MethodPut, partURL.GetUploadUrl(), bytes.NewReader(body))
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		resp.Body.Close()
		require.True(t, resp.StatusCode >= 200 && resp.StatusCode < 300)

		_, err = client.ReportPartUploaded(context.Background(), &filev1.ReportPartUploadedRequest{
			Id:         created.GetMetadata().GetId(),
			OwnerId:    "user-1",
			PartNumber: part,
			Etag:       `"etag"` ,
		})
		require.NoError(t, err)
	}

	_, err = client.CompleteMultipartUpload(context.Background(), &filev1.CompleteMultipartUploadRequest{
		Id:             created.GetMetadata().GetId(),
		OwnerId:        "user-1",
		ChecksumSha256: strings.Repeat("f", 64),
	})
	require.NoError(t, err)
}

func TestE2E_AuthDeniedWithoutToken(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e in short mode")
	}

	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("fms"),
		tcpostgres.WithUsername("fms"),
		tcpostgres.WithPassword("fms"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })
	pgDSN, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	rd, err := tcredis.Run(ctx, "redis:8.8.0-alpine")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rd.Terminate(ctx) })
	redisAddr, err := rd.Endpoint(ctx, "")
	require.NoError(t, err)

	kf, err := tckafka.Run(ctx, "confluentinc/confluent-local:7.5.0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = kf.Terminate(ctx) })
	brokers, err := kf.Brokers(ctx)
	require.NoError(t, err)

	minioC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "minio/minio:RELEASE.2025-09-07T16-13-09Z",
			ExposedPorts: []string{"9000/tcp"},
			Env: map[string]string{
				"MINIO_ROOT_USER":     "minioadmin",
				"MINIO_ROOT_PASSWORD": "minioadmin",
			},
			Cmd:        []string{"server", "/data"},
			WaitingFor: wait.ForHTTP("/minio/health/ready").WithPort("9000/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = minioC.Terminate(ctx) })
	minioHost, err := minioC.Host(ctx)
	require.NoError(t, err)
	minioPort, err := minioC.MappedPort(ctx, "9000/tcp")
	require.NoError(t, err)

	grpcPort := freePort(t)
	metricsPort := freePort(t)
	secret := "auth-e2e-secret"

	t.Setenv("APP_ENV", "development")
	t.Setenv("GRPC_PORT", itoa(grpcPort))
	t.Setenv("METRICS_PORT", itoa(metricsPort))
	t.Setenv("POSTGRES_DSN", pgDSN)
	t.Setenv("REDIS_ADDR", redisAddr)
	t.Setenv("MINIO_ENDPOINT", minioHost+":"+minioPort.Port())
	t.Setenv("MINIO_ACCESS_KEY", "minioadmin")
	t.Setenv("MINIO_SECRET_KEY", "minioadmin")
	t.Setenv("KAFKA_BROKERS", strings.Join(brokers, ","))
	t.Setenv("JWT_ENABLED", "true")
	t.Setenv("JWT_HMAC_SECRET", secret)
	t.Setenv("GRPC_INSECURE", "true")

	cfg, err := config.Load()
	require.NoError(t, err)
	startApp(t, cfg)

	addr := "127.0.0.1:" + itoa(grpcPort)
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })
	client := filev1.NewFileServiceClient(conn)

	_, err = client.GetFile(context.Background(), &filev1.GetFileRequest{Id: "x", OwnerId: "user-1"})
	require.Error(t, err)

	token, err := auth.GenerateDevToken(secret, cfg.JWTIssuer, cfg.JWTAudience, "user-1", time.Hour)
	require.NoError(t, err)
	mdCtx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+token))
	_, err = client.GetFile(mdCtx, &filev1.GetFileRequest{Id: "missing", OwnerId: "user-1"})
	require.Error(t, err)
}
