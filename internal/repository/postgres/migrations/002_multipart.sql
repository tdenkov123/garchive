-- +goose Up
CREATE TABLE IF NOT EXISTS upload_parts (
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    part_number INT NOT NULL CHECK (part_number > 0),
    etag TEXT NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (file_id, part_number)
);

-- +goose Down
DROP TABLE IF EXISTS upload_parts;
