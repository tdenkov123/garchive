package service

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/tdenkov123/file-metadata-service/internal/domain"
)

type FileRepository interface {
	Create(ctx context.Context, file domain.FileMetadata) error
	GetByID(ctx context.Context, id string) (domain.FileMetadata, error)
	Confirm(ctx context.Context, id, checksum string) (domain.FileMetadata, error)
	List(ctx context.Context, filter domain.ListFilter) (domain.ListResult, error)
	SoftDelete(ctx context.Context, id string) error
}

type ObjectStorage interface {
	Bucket() string
	PresignUpload(ctx context.Context, objectKey, contentType string) (url string, expiresIn time.Duration, err error)
	PresignDownload(ctx context.Context, objectKey string) (url string, expiresIn time.Duration, err error)
	RemoveObject(ctx context.Context, objectKey string) error
}

type FileCache interface {
	GetFile(ctx context.Context, id string) (domain.FileMetadata, bool, error)
	SetFile(ctx context.Context, file domain.FileMetadata) error
	InvalidateFile(ctx context.Context, id string) error
}

type EventPublisher interface {
	PublishFileCreated(ctx context.Context, file domain.FileMetadata) error
	PublishFileReady(ctx context.Context, file domain.FileMetadata) error
	PublishFileDeleted(ctx context.Context, file domain.FileMetadata) error
}

type FileService struct {
	repo    FileRepository
	storage ObjectStorage
	cache   FileCache
	events  EventPublisher
}

func NewFileService(repo FileRepository, storage ObjectStorage, cache FileCache, events EventPublisher) *FileService {
	return &FileService{
		repo:    repo,
		storage: storage,
		cache:   cache,
		events:  events,
	}
}

type CreateUploadResult struct {
	Metadata  domain.FileMetadata
	UploadURL string
	ExpiresIn time.Duration
}

func (s *FileService) CreateUpload(ctx context.Context, ownerID, originalName, contentType string, sizeBytes int64) (CreateUploadResult, error) {
	if err := validateCreateInput(ownerID, originalName, contentType, sizeBytes); err != nil {
		return CreateUploadResult{}, err
	}

	now := time.Now().UTC()
	id := uuid.NewString()
	objectKey := path.Join(ownerID, id, sanitizeFilename(originalName))

	file := domain.FileMetadata{
		ID:           id,
		OwnerID:      ownerID,
		Bucket:       s.storage.Bucket(),
		ObjectKey:    objectKey,
		OriginalName: originalName,
		ContentType:  contentType,
		SizeBytes:    sizeBytes,
		Status:       domain.FileStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	uploadURL, expiresIn, err := s.storage.PresignUpload(ctx, objectKey, contentType)
	if err != nil {
		return CreateUploadResult{}, err
	}

	if err := s.repo.Create(ctx, file); err != nil {
		return CreateUploadResult{}, err
	}

	_ = s.cache.SetFile(ctx, file)
	_ = s.events.PublishFileCreated(ctx, file)

	return CreateUploadResult{
		Metadata:  file,
		UploadURL: uploadURL,
		ExpiresIn: expiresIn,
	}, nil
}

func (s *FileService) ConfirmUpload(ctx context.Context, id, ownerID, checksum string) (domain.FileMetadata, error) {
	if id == "" || ownerID == "" {
		return domain.FileMetadata{}, domain.ErrInvalidInput
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.FileMetadata{}, err
	}
	if !existing.IsOwnedBy(ownerID) {
		return domain.FileMetadata{}, domain.ErrAccessDenied
	}
	if existing.Status == domain.FileStatusReady {
		return domain.FileMetadata{}, domain.ErrAlreadyExists
	}
	if existing.Status == domain.FileStatusDeleted {
		return domain.FileMetadata{}, domain.ErrNotFound
	}

	file, err := s.repo.Confirm(ctx, id, checksum)
	if err != nil {
		return domain.FileMetadata{}, err
	}

	_ = s.cache.SetFile(ctx, file)
	_ = s.events.PublishFileReady(ctx, file)

	return file, nil
}

func (s *FileService) GetFile(ctx context.Context, id, ownerID string) (domain.FileMetadata, error) {
	if id == "" || ownerID == "" {
		return domain.FileMetadata{}, domain.ErrInvalidInput
	}

	if cached, ok, err := s.cache.GetFile(ctx, id); err == nil && ok {
		if !cached.IsOwnedBy(ownerID) {
			return domain.FileMetadata{}, domain.ErrAccessDenied
		}
		if cached.Status != domain.FileStatusDeleted {
			return cached, nil
		}
	}

	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.FileMetadata{}, err
	}
	if !file.IsOwnedBy(ownerID) {
		return domain.FileMetadata{}, domain.ErrAccessDenied
	}
	if file.Status == domain.FileStatusDeleted {
		return domain.FileMetadata{}, domain.ErrNotFound
	}

	_ = s.cache.SetFile(ctx, file)
	return file, nil
}

func (s *FileService) ListFiles(ctx context.Context, filter domain.ListFilter) (domain.ListResult, error) {
	if filter.OwnerID == "" {
		return domain.ListResult{}, domain.ErrInvalidInput
	}
	return s.repo.List(ctx, filter)
}

type DownloadURLResult struct {
	URL       string
	ExpiresIn time.Duration
}

func (s *FileService) GetDownloadURL(ctx context.Context, id, ownerID string) (DownloadURLResult, error) {
	file, err := s.GetFile(ctx, id, ownerID)
	if err != nil {
		return DownloadURLResult{}, err
	}
	if !file.CanDownload() {
		return DownloadURLResult{}, fmt.Errorf("%w: file is not ready", domain.ErrInvalidInput)
	}

	url, expiresIn, err := s.storage.PresignDownload(ctx, file.ObjectKey)
	if err != nil {
		return DownloadURLResult{}, err
	}
	return DownloadURLResult{URL: url, ExpiresIn: expiresIn}, nil
}

func (s *FileService) DeleteFile(ctx context.Context, id, ownerID string) error {
	file, err := s.GetFile(ctx, id, ownerID)
	if err != nil {
		return err
	}

	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return err
	}

	_ = s.storage.RemoveObject(ctx, file.ObjectKey)
	_ = s.cache.InvalidateFile(ctx, id)

	file.Status = domain.FileStatusDeleted
	file.UpdatedAt = time.Now().UTC()
	_ = s.events.PublishFileDeleted(ctx, file)

	return nil
}

func validateCreateInput(ownerID, originalName, contentType string, sizeBytes int64) error {
	if ownerID == "" || originalName == "" || contentType == "" {
		return domain.ErrInvalidInput
	}
	if sizeBytes <= 0 {
		return domain.ErrInvalidInput
	}
	return nil
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.TrimSpace(name)
	if name == "" {
		return "file"
	}
	return name
}
