package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tdenkov123/file-metadata-service/internal/domain"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, file domain.FileMetadata) error {
	const q = `
		INSERT INTO files (
			id, owner_id, bucket, object_key, original_name,
			content_type, size_bytes, checksum_sha256, status, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`

	_, err := r.pool.Exec(ctx, q,
		file.ID, file.OwnerID, file.Bucket, file.ObjectKey, file.OriginalName,
		file.ContentType, file.SizeBytes, file.ChecksumSHA256, string(file.Status),
		file.CreatedAt, file.UpdatedAt,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (domain.FileMetadata, error) {
	const q = `
		SELECT id, owner_id, bucket, object_key, original_name, content_type,
		       size_bytes, checksum_sha256, status, created_at, updated_at
		FROM files WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id)
	return scanFile(row)
}

func (r *Repository) Confirm(ctx context.Context, id, checksum string) (domain.FileMetadata, error) {
	const q = `
		UPDATE files
		SET status = 'ready', checksum_sha256 = $2, updated_at = $3
		WHERE id = $1 AND status = 'pending'
		RETURNING id, owner_id, bucket, object_key, original_name, content_type,
		          size_bytes, checksum_sha256, status, created_at, updated_at`

	row := r.pool.QueryRow(ctx, q, id, checksum, time.Now().UTC())
	file, err := scanFile(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.FileMetadata{}, domain.ErrNotFound
		}
		return domain.FileMetadata{}, err
	}
	return file, nil
}

func (r *Repository) List(ctx context.Context, filter domain.ListFilter) (domain.ListResult, error) {
	pageSize := filter.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	var cursorTime time.Time
	var cursorID uuid.UUID
	if filter.PageToken != "" {
		t, id, err := decodePageToken(filter.PageToken)
		if err != nil {
			return domain.ListResult{}, fmt.Errorf("%w: invalid page token", domain.ErrInvalidInput)
		}
		cursorTime = t
		cursorID = id
	}

	const q = `
		SELECT id, owner_id, bucket, object_key, original_name, content_type,
		       size_bytes, checksum_sha256, status, created_at, updated_at
		FROM files
		WHERE owner_id = $1 AND status != 'deleted'
		  AND ($2::timestamptz IS NULL OR (created_at, id) < ($2, $3))
		ORDER BY created_at DESC, id DESC
		LIMIT $4`

	var cursorTimeArg *time.Time
	if !cursorTime.IsZero() {
		cursorTimeArg = &cursorTime
	}

	rows, err := r.pool.Query(ctx, q, filter.OwnerID, cursorTimeArg, cursorID, pageSize+1)
	if err != nil {
		return domain.ListResult{}, err
	}
	defer rows.Close()

	var files []domain.FileMetadata
	for rows.Next() {
		file, err := scanFile(rows)
		if err != nil {
			return domain.ListResult{}, err
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return domain.ListResult{}, err
	}

	var nextToken string
	if len(files) > int(pageSize) {
		last := files[pageSize-1]
		nextToken = encodePageToken(last.CreatedAt, last.ID)
		files = files[:pageSize]
	}

	return domain.ListResult{Files: files, NextPageToken: nextToken}, nil
}

func (r *Repository) SoftDelete(ctx context.Context, id string) error {
	const q = `
		UPDATE files SET status = 'deleted', updated_at = $2
		WHERE id = $1 AND status != 'deleted'`

	tag, err := r.pool.Exec(ctx, q, id, time.Now().UTC())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanFile(row scannable) (domain.FileMetadata, error) {
	var file domain.FileMetadata
	var status string
	err := row.Scan(
		&file.ID, &file.OwnerID, &file.Bucket, &file.ObjectKey, &file.OriginalName,
		&file.ContentType, &file.SizeBytes, &file.ChecksumSHA256, &status,
		&file.CreatedAt, &file.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.FileMetadata{}, domain.ErrNotFound
		}
		return domain.FileMetadata{}, err
	}
	file.Status = domain.FileStatus(status)
	return file, nil
}

func encodePageToken(t time.Time, id string) string {
	return fmt.Sprintf("%d|%s", t.UTC().UnixNano(), id)
}

func decodePageToken(token string) (time.Time, uuid.UUID, error) {
	var ns int64
	var idStr string
	_, err := fmt.Sscanf(token, "%d|%s", &ns, &idStr)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	return time.Unix(0, ns), id, nil
}
