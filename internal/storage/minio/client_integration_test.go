//go:build integration

package minio_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/tdenkov123/file-metadata-service/internal/config"
	"github.com/tdenkov123/file-metadata-service/internal/storage/minio"
)

func TestClient_PresignUploadDownload(t *testing.T) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:RELEASE.2025-09-07T16-13-09Z",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadmin",
			"MINIO_ROOT_PASSWORD": "minioadmin",
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/live").WithPort("9000/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "9000/tcp")
	require.NoError(t, err)

	client, err := minio.NewClient(&config.Config{
		MinioEndpoint:  host + ":" + port.Port(),
		MinioAccessKey: "minioadmin",
		MinioSecretKey: "minioadmin",
		MinioBucket:    "files",
		MinioUseSSL:    false,
		PresignTTL:     5 * time.Minute,
	})
	require.NoError(t, err)
	require.NoError(t, client.EnsureBucket(ctx))

	uploadURL, _, err := client.PresignUpload(ctx, "user-1/obj.bin", "application/octet-stream")
	require.NoError(t, err)
	require.NotEmpty(t, uploadURL.String())

	downloadURL, _, err := client.PresignDownload(ctx, "user-1/obj.bin")
	require.NoError(t, err)
	require.NotEmpty(t, downloadURL.String())
}
