package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	const q = `
		CREATE TABLE IF NOT EXISTS files (
			id UUID PRIMARY KEY,
			owner_id TEXT NOT NULL,
			bucket TEXT NOT NULL,
			object_key TEXT NOT NULL UNIQUE,
			original_name TEXT NOT NULL,
			content_type TEXT NOT NULL,
			size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
			checksum_sha256 TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL CHECK (status IN ('pending', 'ready', 'deleted')),
			upload_mode TEXT NOT NULL DEFAULT 'single' CHECK (upload_mode IN ('single', 'multipart')),
			upload_id TEXT NOT NULL DEFAULT '',
			part_size BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_files_owner_created ON files (owner_id, created_at DESC)
			WHERE status != 'deleted';
		ALTER TABLE files ADD COLUMN IF NOT EXISTS upload_mode TEXT NOT NULL DEFAULT 'single';
		ALTER TABLE files ADD COLUMN IF NOT EXISTS upload_id TEXT NOT NULL DEFAULT '';
		ALTER TABLE files ADD COLUMN IF NOT EXISTS part_size BIGINT NOT NULL DEFAULT 0;
		CREATE TABLE IF NOT EXISTS upload_parts (
			file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
			part_number INT NOT NULL CHECK (part_number > 0),
			etag TEXT NOT NULL,
			uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (file_id, part_number)
		);`
	_, err := pool.Exec(ctx, q)
	return err
}
