//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/tdenkov123/file-metadata-service/internal/domain"
	pgrepo "github.com/tdenkov123/file-metadata-service/internal/repository/postgres"
)

func TestRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:18.4-alpine",
		tcpostgres.WithDatabase("fms"),
		tcpostgres.WithUsername("fms"),
		tcpostgres.WithPassword("fms"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgrepo.Connect(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })
	require.NoError(t, pgrepo.RunMigrations(ctx, pool))

	repo := pgrepo.NewRepository(pool)
	now := time.Now().UTC()
	id := uuid.NewString()
	file := domain.FileMetadata{
		ID:           id,
		OwnerID:      "user-1",
		Bucket:       "files",
		ObjectKey:    "user-1/" + id + "/doc.pdf",
		OriginalName: "doc.pdf",
		ContentType:  "application/pdf",
		SizeBytes:    1024,
		Status:       domain.FileStatusPending,
		UploadMode:   domain.UploadModeSingle,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	require.NoError(t, repo.Create(ctx, file))

	got, err := repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "user-1", got.OwnerID)

	checksum := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	ready, err := repo.Confirm(ctx, id, checksum)
	require.NoError(t, err)
	require.Equal(t, domain.FileStatusReady, ready.Status)

	require.NoError(t, repo.SoftDelete(ctx, id))
	got, err = repo.GetByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, domain.FileStatusDeleted, got.Status)
}
