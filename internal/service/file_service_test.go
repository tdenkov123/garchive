package service_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdenkov123/file-metadata-service/internal/domain"
	"github.com/tdenkov123/file-metadata-service/internal/service"
)

const testPartSize = 5 * 1024 * 1024

type mockRepo struct {
	files map[string]domain.FileMetadata
	parts map[string][]domain.UploadPart
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		files: make(map[string]domain.FileMetadata),
		parts: make(map[string][]domain.UploadPart),
	}
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
	file.UploadMode = domain.UploadModeSingle
	file.UploadID = ""
	file.PartSize = 0
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

func (m *mockRepo) SaveUploadPart(_ context.Context, fileID string, part domain.UploadPart) error {
	for i, p := range m.parts[fileID] {
		if p.PartNumber == part.PartNumber {
			m.parts[fileID][i] = part
			return nil
		}
	}
	m.parts[fileID] = append(m.parts[fileID], part)
	return nil
}

func (m *mockRepo) ListUploadParts(_ context.Context, fileID string) ([]domain.UploadPart, error) {
	return m.parts[fileID], nil
}

func (m *mockRepo) DeleteUploadParts(_ context.Context, fileID string) error {
	delete(m.parts, fileID)
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

func (m *mockStorage) CreateMultipartUpload(_ context.Context, _, _ string) (string, error) {
	return "upload-id-1", nil
}

func (m *mockStorage) PresignUploadPart(_ context.Context, _, _ string, partNumber int32) (string, time.Duration, error) {
	return fmt.Sprintf("https://upload.example/part/%d", partNumber), 15 * time.Minute, nil
}

func (m *mockStorage) CompleteMultipartUpload(_ context.Context, _, _ string, _ []domain.CompletedPart) error {
	return nil
}

func (m *mockStorage) AbortMultipartUpload(_ context.Context, _, _ string) error { return nil }

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

func newTestService(repo *mockRepo) *service.FileService {
	return service.NewFileService(repo, &mockStorage{bucket: "files"}, &mockCache{}, &mockEvents{}, testPartSize, 5*1024*1024*1024)
}

func TestFileService_CreateAndConfirmUpload(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	result, err := svc.CreateUpload(context.Background(), "user-1", "doc.pdf", "application/pdf", 1024)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Metadata.ID)
	assert.Equal(t, domain.FileStatusPending, result.Metadata.Status)
	assert.Contains(t, result.UploadURL, "https://")

	checksum := strings.Repeat("a", 64)
	confirmed, err := svc.ConfirmUpload(context.Background(), result.Metadata.ID, "user-1", checksum)
	require.NoError(t, err)
	assert.Equal(t, domain.FileStatusReady, confirmed.Status)
	assert.Equal(t, checksum, confirmed.ChecksumSHA256)
}

func TestFileService_GetFile_AccessDenied(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	result, err := svc.CreateUpload(context.Background(), "user-1", "doc.pdf", "application/pdf", 1024)
	require.NoError(t, err)

	_, err = svc.GetFile(context.Background(), result.Metadata.ID, "user-2")
	require.ErrorIs(t, err, domain.ErrAccessDenied)
}

func TestFileService_GetDownloadURL_RequiresReady(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	result, err := svc.CreateUpload(context.Background(), "user-1", "doc.pdf", "application/pdf", 1024)
	require.NoError(t, err)

	_, err = svc.GetDownloadURL(context.Background(), result.Metadata.ID, "user-1")
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestFileService_DeleteFile(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	result, err := svc.CreateUpload(context.Background(), "user-1", "doc.pdf", "application/pdf", 1024)
	require.NoError(t, err)

	_, err = svc.ConfirmUpload(context.Background(), result.Metadata.ID, "user-1", strings.Repeat("c", 64))
	require.NoError(t, err)

	err = svc.DeleteFile(context.Background(), result.Metadata.ID, "user-1")
	require.NoError(t, err)

	_, err = svc.GetFile(context.Background(), result.Metadata.ID, "user-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestFileService_MultipartUploadFlow(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	size := int64(testPartSize + 1024)
	created, err := svc.CreateMultipartUpload(context.Background(), "user-1", "big.bin", "application/octet-stream", size)
	require.NoError(t, err)
	assert.Equal(t, domain.UploadModeMultipart, created.Metadata.UploadMode)
	assert.Equal(t, int32(2), created.TotalParts)
	assert.Equal(t, "upload-id-1", created.UploadID)

	partURL, err := svc.GetPartUploadURL(context.Background(), created.Metadata.ID, "user-1", 1)
	require.NoError(t, err)
	assert.Contains(t, partURL.URL, "/part/1")
	assert.Equal(t, int64(testPartSize), partURL.PartSize)

	_, err = svc.ReportPartUploaded(context.Background(), created.Metadata.ID, "user-1", 1, `"etag-1"`)
	require.NoError(t, err)
	_, err = svc.ReportPartUploaded(context.Background(), created.Metadata.ID, "user-1", 2, "etag-2")
	require.NoError(t, err)

	listed, err := svc.ListUploadParts(context.Background(), created.Metadata.ID, "user-1")
	require.NoError(t, err)
	assert.Len(t, listed.Parts, 2)

	ready, err := svc.CompleteMultipartUpload(context.Background(), created.Metadata.ID, "user-1", strings.Repeat("d", 64))
	require.NoError(t, err)
	assert.Equal(t, domain.FileStatusReady, ready.Status)
}

func TestFileService_CreateMultipartUpload_RejectsSmallFile(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	_, err := svc.CreateMultipartUpload(context.Background(), "user-1", "small.bin", "application/octet-stream", 1024)
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestFileService_ListUploadParts_Resume(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	size := int64(testPartSize * 3)
	created, err := svc.CreateMultipartUpload(context.Background(), "user-1", "big.bin", "application/octet-stream", size)
	require.NoError(t, err)
	assert.Equal(t, int32(3), created.TotalParts)

	_, err = svc.ReportPartUploaded(context.Background(), created.Metadata.ID, "user-1", 1, "etag-1")
	require.NoError(t, err)

	listed, err := svc.ListUploadParts(context.Background(), created.Metadata.ID, "user-1")
	require.NoError(t, err)
	assert.Len(t, listed.Parts, 1)
	assert.Equal(t, int32(1), listed.Parts[0].PartNumber)
}

type trackingCache struct {
	hits  int
	store map[string]domain.FileMetadata
}

func (c *trackingCache) GetFile(_ context.Context, id string) (domain.FileMetadata, bool, error) {
	if file, ok := c.store[id]; ok {
		c.hits++
		return file, true, nil
	}
	return domain.FileMetadata{}, false, nil
}

func (c *trackingCache) SetFile(_ context.Context, file domain.FileMetadata) error {
	if c.store == nil {
		c.store = make(map[string]domain.FileMetadata)
	}
	c.store[file.ID] = file
	return nil
}

func (c *trackingCache) InvalidateFile(_ context.Context, id string) error {
	delete(c.store, id)
	return nil
}

type trackingEvents struct {
	created int
	ready   int
	deleted int
}

func (e *trackingEvents) PublishFileCreated(_ context.Context, _ domain.FileMetadata) error {
	e.created++
	return nil
}

func (e *trackingEvents) PublishFileReady(_ context.Context, _ domain.FileMetadata) error {
	e.ready++
	return nil
}

func (e *trackingEvents) PublishFileDeleted(_ context.Context, _ domain.FileMetadata) error {
	e.deleted++
	return nil
}

func TestFileService_ListFiles(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	_, err := svc.CreateUpload(context.Background(), "user-1", "a.pdf", "application/pdf", 100)
	require.NoError(t, err)
	_, err = svc.CreateUpload(context.Background(), "user-1", "b.pdf", "application/pdf", 200)
	require.NoError(t, err)

	result, err := svc.ListFiles(context.Background(), domain.ListFilter{OwnerID: "user-1"})
	require.NoError(t, err)
	assert.Len(t, result.Files, 2)
}

func TestFileService_GetDownloadURL_Success(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	created, err := svc.CreateUpload(context.Background(), "user-1", "doc.pdf", "application/pdf", 1024)
	require.NoError(t, err)
	_, err = svc.ConfirmUpload(context.Background(), created.Metadata.ID, "user-1", strings.Repeat("a", 64))
	require.NoError(t, err)

	dl, err := svc.GetDownloadURL(context.Background(), created.Metadata.ID, "user-1")
	require.NoError(t, err)
	assert.Contains(t, dl.URL, "https://")
}

func TestFileService_AbortMultipartUpload(t *testing.T) {
	repo := newMockRepo()
	svc := newTestService(repo)

	created, err := svc.CreateMultipartUpload(context.Background(), "user-1", "big.bin", "application/octet-stream", int64(testPartSize+1))
	require.NoError(t, err)

	err = svc.AbortMultipartUpload(context.Background(), created.Metadata.ID, "user-1")
	require.NoError(t, err)

	_, err = svc.GetFile(context.Background(), created.Metadata.ID, "user-1")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestFileService_GetFile_CacheHit(t *testing.T) {
	repo := newMockRepo()
	cache := &trackingCache{store: make(map[string]domain.FileMetadata)}
	svc := service.NewFileService(repo, &mockStorage{bucket: "files"}, cache, &mockEvents{}, testPartSize, 5*1024*1024*1024)

	created, err := svc.CreateUpload(context.Background(), "user-1", "doc.pdf", "application/pdf", 1024)
	require.NoError(t, err)

	_, err = svc.GetFile(context.Background(), created.Metadata.ID, "user-1")
	require.NoError(t, err)
	_, err = svc.GetFile(context.Background(), created.Metadata.ID, "user-1")
	require.NoError(t, err)
	assert.Equal(t, 2, cache.hits)
}

func TestFileService_EventsPublished(t *testing.T) {
	repo := newMockRepo()
	events := &trackingEvents{}
	svc := service.NewFileService(repo, &mockStorage{bucket: "files"}, &mockCache{}, events, testPartSize, 5*1024*1024*1024)

	created, err := svc.CreateUpload(context.Background(), "user-1", "doc.pdf", "application/pdf", 1024)
	require.NoError(t, err)
	assert.Equal(t, 1, events.created)

	_, err = svc.ConfirmUpload(context.Background(), created.Metadata.ID, "user-1", strings.Repeat("b", 64))
	require.NoError(t, err)
	assert.Equal(t, 1, events.ready)

	err = svc.DeleteFile(context.Background(), created.Metadata.ID, "user-1")
	require.NoError(t, err)
	assert.Equal(t, 1, events.deleted)
}
