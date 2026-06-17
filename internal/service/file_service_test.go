package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdenkov123/file-metadata-service/internal/domain"
	"github.com/tdenkov123/file-metadata-service/internal/service"
)

type mockRepo struct {
	files map[string]domain.FileMetadata
}

func newMockRepo() *mockRepo {
	return &mockRepo{files: make(map[string]domain.FileMetadata)}
}

func (m *mockRepo) Create(_ context.Context, file domain.FileMetadata) error {
	m.files[file.ID] = file
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id string) (domain.FileMetadata, error) {
	file, ok := m.files[id]
	if !ok {
		return domain.FileMetadata{}, domain.ErrNotFound
	}
	return file, nil
}

func (m *mockRepo) Confirm(_ context.Context, id, checksum string) (domain.FileMetadata, error) {
	file, ok := m.files[id]
	if !ok || file.Status != domain.FileStatusPending {
		return domain.FileMetadata{}, domain.ErrNotFound
	}
	file.Status = domain.FileStatusReady
	file.ChecksumSHA256 = checksum
	file.UpdatedAt = time.Now().UTC()
	m.files[id] = file
	return file, nil
}

func (m *mockRepo) List(_ context.Context, filter domain.ListFilter) (domain.ListResult, error) {
	var files []domain.FileMetadata
	for _, f := range m.files {
		if f.OwnerID == filter.OwnerID && f.Status != domain.FileStatusDeleted {
			files = append(files, f)
		}
	}
	return domain.ListResult{Files: files}, nil
}

func (m *mockRepo) SoftDelete(_ context.Context, id string) error {
	file, ok := m.files[id]
	if !ok || file.Status == domain.FileStatusDeleted {
		return domain.ErrNotFound
	}
	file.Status = domain.FileStatusDeleted
	m.files[id] = file
	return nil
}

type mockStorage struct {
	bucket string
}

func (m *mockStorage) Bucket() string { return m.bucket }

func (m *mockStorage) PresignUpload(_ context.Context, _, _ string) (string, time.Duration, error) {
	return "https://upload.example/presigned", 15 * time.Minute, nil
}

func (m *mockStorage) PresignDownload(_ context.Context, _ string) (string, time.Duration, error) {
	return "https://download.example/presigned", 15 * time.Minute, nil
}

func (m *mockStorage) RemoveObject(_ context.Context, _ string) error { return nil }

type mockCache struct{}

func (m *mockCache) GetFile(_ context.Context, _ string) (domain.FileMetadata, bool, error) {
	return domain.FileMetadata{}, false, nil
}

func (m *mockCache) SetFile(_ context.Context, _ domain.FileMetadata) error { return nil }

func (m *mockCache) InvalidateFile(_ context.Context, _ string) error { return nil }

type mockEvents struct{}

func (m *mockEvents) PublishFileCreated(_ context.Context, _ domain.FileMetadata) error { return nil }
func (m *mockEvents) PublishFileReady(_ context.Context, _ domain.FileMetadata) error   { return nil }
func (m *mockEvents) PublishFileDeleted(_ context.Context, _ domain.FileMetadata) error { return nil }

func TestFileService_CreateAndConfirmUpload(t *testing.T) {
	repo := newMockRepo()
	svc := service.NewFileService(repo, &mockStorage{bucket: "files"}, &mockCache{}, &mockEvents{})

	result, err := svc.CreateUpload(context.Background(), "user-1", "doc.pdf", "application/pdf", 1024)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Metadata.ID)
	assert.Equal(t, domain.FileStatusPending, result.Metadata.Status)
	assert.Contains(t, result.UploadURL, "https://")

	confirmed, err := svc.ConfirmUpload(context.Background(), result.Metadata.ID, "user-1", "abc123")
	require.NoError(t, err)
	assert.Equal(t, domain.FileStatusReady, confirmed.Status)
	assert.Equal(t, "abc123", confirmed.ChecksumSHA256)
}

func TestFileService_GetFile_AccessDenied(t *testing.T) {
	repo := newMockRepo()
	svc := service.NewFileService(repo, &mockStorage{bucket: "files"}, &mockCache{}, &mockEvents{})

	result, err := svc.CreateUpload(context.Background(), "user-1", "doc.pdf", "application/pdf", 1024)
	require.NoError(t, err)

	_, err = svc.GetFile(context.Background(), result.Metadata.ID, "user-2")
	require.ErrorIs(t, err, domain.ErrAccessDenied)
}

func TestFileService_GetDownloadURL_RequiresReady(t *testing.T) {
	repo := newMockRepo()
	svc := service.NewFileService(repo, &mockStorage{bucket: "files"}, &mockCache{}, &mockEvents{})

	result, err := svc.CreateUpload(context.Background(), "user-1", "doc.pdf", "application/pdf", 1024)
	require.NoError(t, err)

	_, err = svc.GetDownloadURL(context.Background(), result.Metadata.ID, "user-1")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestFileService_DeleteFile(t *testing.T) {
	repo := newMockRepo()
	svc := service.NewFileService(repo, &mockStorage{bucket: "files"}, &mockCache{}, &mockEvents{})

	result, err := svc.CreateUpload(context.Background(), "user-1", "doc.pdf", "application/pdf", 1024)
	require.NoError(t, err)

	_, err = svc.ConfirmUpload(context.Background(), result.Metadata.ID, "user-1", "hash")
	require.NoError(t, err)

	err = svc.DeleteFile(context.Background(), result.Metadata.ID, "user-1")
	require.NoError(t, err)

	_, err = svc.GetFile(context.Background(), result.Metadata.ID, "user-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}
