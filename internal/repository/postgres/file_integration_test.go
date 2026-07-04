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

func connectPostgres(t *testing.T, ctx context.Context, connStr string) *pgrepo.Repository {
	t.Helper()

	var poolErr error
	var repo *pgrepo.Repository
	for attempt := range 15 {
		pool, err := pgrepo.Connect(ctx, connStr)
		if err != nil {
			poolErr = err
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}
		if err := pgrepo.RunMigrations(ctx, pool); err != nil {
			pool.Close()
			poolErr = err
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
			continue
		}
		repo = pgrepo.NewRepository(pool)
		t.Cleanup(func() { pool.Close() })
		return repo
	}
	require.NoError(t, poolErr)
	return repo
}

func TestRepository_CRUD(t *testing.T) {
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("fms"),
		tcpostgres.WithUsername("fms"),
		tcpostgres.WithPassword("fms"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	repo := connectPostgres(t, ctx, connStr)

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
