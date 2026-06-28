-- +goose Up
ALTER TABLE files ADD COLUMN IF NOT EXISTS upload_mode TEXT NOT NULL DEFAULT 'single'
    CHECK (upload_mode IN ('single', 'multipart'));
ALTER TABLE files ADD COLUMN IF NOT EXISTS upload_id TEXT NOT NULL DEFAULT '';
ALTER TABLE files ADD COLUMN IF NOT EXISTS part_size BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS upload_parts (
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    part_number INT NOT NULL CHECK (part_number > 0),
    etag TEXT NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (file_id, part_number)
);

-- +goose Down
DROP TABLE IF EXISTS upload_parts;
ALTER TABLE files DROP COLUMN IF EXISTS part_size;
ALTER TABLE files DROP COLUMN IF EXISTS upload_id;
ALTER TABLE files DROP COLUMN IF EXISTS upload_mode;
