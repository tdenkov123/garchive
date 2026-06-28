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

type CreateMultipartUploadResult struct {
	Metadata   domain.FileMetadata
	UploadID   string
	PartSize   int64
	TotalParts int32
}

type PartUploadURLResult struct {
	URL         string
	ExpiresIn   time.Duration
	PartNumber  int32
	PartSize    int64
}

type ListUploadPartsResult struct {
	UploadID   string
	PartSize   int64
	TotalParts int32
	Parts      []domain.UploadPart
}

func (s *FileService) CreateMultipartUpload(ctx context.Context, ownerID, originalName, contentType string, sizeBytes int64) (CreateMultipartUploadResult, error) {
	if err := validateCreateInput(ownerID, originalName, contentType, sizeBytes); err != nil {
		return CreateMultipartUploadResult{}, err
	}
	if sizeBytes <= s.multipartPartSize {
		return CreateMultipartUploadResult{}, fmt.Errorf("%w: use CreateUpload for files <= part size", domain.ErrInvalidInput)
	}

	now := time.Now().UTC()
	id := uuid.NewString()
	objectKey := path.Join(ownerID, id, sanitizeFilename(originalName))

	uploadID, err := s.storage.CreateMultipartUpload(ctx, objectKey, contentType)
	if err != nil {
		return CreateMultipartUploadResult{}, err
	}

	file := domain.FileMetadata{
		ID:           id,
		OwnerID:      ownerID,
		Bucket:       s.storage.Bucket(),
		ObjectKey:    objectKey,
		OriginalName: originalName,
		ContentType:  contentType,
		SizeBytes:    sizeBytes,
		Status:       domain.FileStatusPending,
		UploadMode:   domain.UploadModeMultipart,
		UploadID:     uploadID,
		PartSize:     s.multipartPartSize,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, file); err != nil {
		_ = s.storage.AbortMultipartUpload(ctx, objectKey, uploadID)
		return CreateMultipartUploadResult{}, err
	}

	_ = s.cache.SetFile(ctx, file)
	_ = s.events.PublishFileCreated(ctx, file)

	return CreateMultipartUploadResult{
		Metadata:   file,
		UploadID:   uploadID,
		PartSize:   s.multipartPartSize,
		TotalParts: file.TotalParts(),
	}, nil
}

func (s *FileService) GetPartUploadURL(ctx context.Context, id, ownerID string, partNumber int32) (PartUploadURLResult, error) {
	file, err := s.getPendingMultipart(ctx, id, ownerID)
	if err != nil {
		return PartUploadURLResult{}, err
	}
	if err := validatePartNumber(partNumber, file.TotalParts()); err != nil {
		return PartUploadURLResult{}, err
	}

	url, expiresIn, err := s.storage.PresignUploadPart(ctx, file.ObjectKey, file.UploadID, partNumber)
	if err != nil {
		return PartUploadURLResult{}, err
	}

	partSize := partSizeForNumber(file, partNumber)
	return PartUploadURLResult{
		URL:        url,
		ExpiresIn:  expiresIn,
		PartNumber: partNumber,
		PartSize:   partSize,
	}, nil
}

func (s *FileService) ReportPartUploaded(ctx context.Context, id, ownerID string, partNumber int32, etag string) (domain.UploadPart, error) {
	file, err := s.getPendingMultipart(ctx, id, ownerID)
	if err != nil {
		return domain.UploadPart{}, err
	}
	if err := validatePartNumber(partNumber, file.TotalParts()); err != nil {
		return domain.UploadPart{}, err
	}
	etag = normalizeETag(etag)
	if etag == "" {
		return domain.UploadPart{}, domain.ErrInvalidInput
	}

	part := domain.UploadPart{
		PartNumber: partNumber,
		ETag:       etag,
		UploadedAt: time.Now().UTC(),
	}
	if err := s.repo.SaveUploadPart(ctx, file.ID, part); err != nil {
		return domain.UploadPart{}, err
	}
	return part, nil
}

func (s *FileService) ListUploadParts(ctx context.Context, id, ownerID string) (ListUploadPartsResult, error) {
	file, err := s.getPendingMultipart(ctx, id, ownerID)
	if err != nil {
		return ListUploadPartsResult{}, err
	}

	parts, err := s.repo.ListUploadParts(ctx, file.ID)
	if err != nil {
		return ListUploadPartsResult{}, err
	}

	return ListUploadPartsResult{
		UploadID:   file.UploadID,
		PartSize:   file.PartSize,
		TotalParts: file.TotalParts(),
		Parts:      parts,
	}, nil
}

func (s *FileService) CompleteMultipartUpload(ctx context.Context, id, ownerID, checksum string) (domain.FileMetadata, error) {
	if id == "" || ownerID == "" {
		return domain.FileMetadata{}, domain.ErrInvalidInput
	}

	file, err := s.getPendingMultipart(ctx, id, ownerID)
	if err != nil {
		return domain.FileMetadata{}, err
	}

	parts, err := s.repo.ListUploadParts(ctx, file.ID)
	if err != nil {
		return domain.FileMetadata{}, err
	}
	totalParts := file.TotalParts()
	if int32(len(parts)) != totalParts {
		return domain.FileMetadata{}, fmt.Errorf("%w: expected %d parts, got %d", domain.ErrInvalidInput, totalParts, len(parts))
	}

	completed := make([]domain.CompletedPart, len(parts))
	seen := make(map[int32]struct{}, len(parts))
	for i, p := range parts {
		if err := validatePartNumber(p.PartNumber, totalParts); err != nil {
			return domain.FileMetadata{}, err
		}
		if _, dup := seen[p.PartNumber]; dup {
			return domain.FileMetadata{}, fmt.Errorf("%w: duplicate part %d", domain.ErrInvalidInput, p.PartNumber)
		}
		seen[p.PartNumber] = struct{}{}
		completed[i] = domain.CompletedPart{PartNumber: p.PartNumber, ETag: p.ETag}
	}

	if err := s.storage.CompleteMultipartUpload(ctx, file.ObjectKey, file.UploadID, completed); err != nil {
		return domain.FileMetadata{}, err
	}

	confirmed, err := s.repo.Confirm(ctx, id, checksum)
	if err != nil {
		return domain.FileMetadata{}, err
	}
	_ = s.repo.DeleteUploadParts(ctx, id)

	_ = s.cache.SetFile(ctx, confirmed)
	_ = s.events.PublishFileReady(ctx, confirmed)

	return confirmed, nil
}

func (s *FileService) AbortMultipartUpload(ctx context.Context, id, ownerID string) error {
	file, err := s.getPendingMultipart(ctx, id, ownerID)
	if err != nil {
		return err
	}

	_ = s.storage.AbortMultipartUpload(ctx, file.ObjectKey, file.UploadID)
	_ = s.repo.DeleteUploadParts(ctx, file.ID)

	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return err
	}
	_ = s.cache.InvalidateFile(ctx, id)

	file.Status = domain.FileStatusDeleted
	file.UpdatedAt = time.Now().UTC()
	_ = s.events.PublishFileDeleted(ctx, file)

	return nil
}

func (s *FileService) getPendingMultipart(ctx context.Context, id, ownerID string) (domain.FileMetadata, error) {
	if id == "" || ownerID == "" {
		return domain.FileMetadata{}, domain.ErrInvalidInput
	}

	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.FileMetadata{}, err
	}
	if !file.IsOwnedBy(ownerID) {
		return domain.FileMetadata{}, domain.ErrAccessDenied
	}
	if file.Status != domain.FileStatusPending {
		return domain.FileMetadata{}, domain.ErrNotFound
	}
	if !file.IsMultipart() || file.UploadID == "" {
		return domain.FileMetadata{}, fmt.Errorf("%w: not a multipart upload", domain.ErrInvalidInput)
	}
	return file, nil
}

func validatePartNumber(partNumber, totalParts int32) error {
	if partNumber < 1 || partNumber > totalParts {
		return fmt.Errorf("%w: part_number must be between 1 and %d", domain.ErrInvalidInput, totalParts)
	}
	return nil
}

func partSizeForNumber(file domain.FileMetadata, partNumber int32) int64 {
	if partNumber < file.TotalParts() {
		return file.PartSize
	}
	remainder := file.SizeBytes % file.PartSize
	if remainder == 0 {
		return file.PartSize
	}
	return remainder
}

func normalizeETag(etag string) string {
	etag = strings.TrimSpace(etag)
	return strings.Trim(etag, `"`)
}
