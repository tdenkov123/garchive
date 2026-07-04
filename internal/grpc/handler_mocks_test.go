package grpc_test

import (
	"context"
	"time"

	"github.com/tdenkov123/file-metadata-service/internal/domain"
)

type handlerMockRepo struct {
	files map[string]domain.FileMetadata
}

func newHandlerMockRepo() *handlerMockRepo {
	return &handlerMockRepo{files: make(map[string]domain.FileMetadata)}
}

func (m *handlerMockRepo) Create(_ context.Context, file domain.FileMetadata) error {
	m.files[file.ID] = file
	return nil
}

func (m *handlerMockRepo) GetByID(_ context.Context, id string) (domain.FileMetadata, error) {
	file, ok := m.files[id]
	if !ok {
		return domain.FileMetadata{}, domain.ErrNotFound
	}
	return file, nil
}

func (m *handlerMockRepo) Confirm(_ context.Context, id, checksum string) (domain.FileMetadata, error) {
	file, ok := m.files[id]
	if !ok {
		return domain.FileMetadata{}, domain.ErrNotFound
	}
	file.Status = domain.FileStatusReady
	file.ChecksumSHA256 = checksum
	m.files[id] = file
	return file, nil
}

func (m *handlerMockRepo) List(_ context.Context, filter domain.ListFilter) (domain.ListResult, error) {
	var files []domain.FileMetadata
	for _, f := range m.files {
		if f.OwnerID == filter.OwnerID && f.Status != domain.FileStatusDeleted {
			files = append(files, f)
		}
	}
	return domain.ListResult{Files: files}, nil
}

func (m *handlerMockRepo) SoftDelete(_ context.Context, id string) error {
	file, ok := m.files[id]
	if !ok {
		return domain.ErrNotFound
	}
	file.Status = domain.FileStatusDeleted
	m.files[id] = file
	return nil
}

func (m *handlerMockRepo) SaveUploadPart(_ context.Context, _ string, _ domain.UploadPart) error {
	return nil
}

func (m *handlerMockRepo) ListUploadParts(_ context.Context, _ string) ([]domain.UploadPart, error) {
	return nil, nil
}

func (m *handlerMockRepo) DeleteUploadParts(_ context.Context, _ string) error { return nil }

type handlerMockStorage struct{ bucket string }

func (m *handlerMockStorage) Bucket() string { return m.bucket }

func (m *handlerMockStorage) PresignUpload(_ context.Context, _, _ string) (string, time.Duration, error) {
	return "https://upload.example", 15 * time.Minute, nil
}

func (m *handlerMockStorage) PresignDownload(_ context.Context, _ string) (string, time.Duration, error) {
	return "https://download.example", 15 * time.Minute, nil
}

func (m *handlerMockStorage) RemoveObject(_ context.Context, _ string) error { return nil }

func (m *handlerMockStorage) CreateMultipartUpload(_ context.Context, _, _ string) (string, error) {
	return "upload-id", nil
}

func (m *handlerMockStorage) PresignUploadPart(_ context.Context, _, _ string, _ int32) (string, time.Duration, error) {
	return "https://part.example", 15 * time.Minute, nil
}

func (m *handlerMockStorage) CompleteMultipartUpload(_ context.Context, _, _ string, _ []domain.CompletedPart) error {
	return nil
}

func (m *handlerMockStorage) AbortMultipartUpload(_ context.Context, _, _ string) error { return nil }

type handlerMockCache struct{}

func (m *handlerMockCache) GetFile(_ context.Context, _ string) (domain.FileMetadata, bool, error) {
	return domain.FileMetadata{}, false, nil
}

func (m *handlerMockCache) SetFile(_ context.Context, _ domain.FileMetadata) error { return nil }

func (m *handlerMockCache) InvalidateFile(_ context.Context, _ string) error { return nil }

type handlerMockEvents struct{}

func (m *handlerMockEvents) PublishFileCreated(_ context.Context, _ domain.FileMetadata) error { return nil }
func (m *handlerMockEvents) PublishFileReady(_ context.Context, _ domain.FileMetadata) error   { return nil }
func (m *handlerMockEvents) PublishFileDeleted(_ context.Context, _ domain.FileMetadata) error { return nil }
