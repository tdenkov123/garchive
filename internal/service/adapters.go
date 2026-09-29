package service

import (
	"context"
	"io"
	"time"

	"github.com/tdenkov123/file-metadata-service/internal/domain"
)

type StorageAdapter struct {
	bucketFn          func() string
	presignUpload     func(ctx context.Context, objectKey, contentType string) (string, time.Duration, error)
	presignDown       func(ctx context.Context, objectKey string) (string, time.Duration, error)
	remove            func(ctx context.Context, objectKey string) error
	putObject         func(ctx context.Context, objectKey, contentType string, r io.Reader, size int64) error
	getObject         func(ctx context.Context, objectKey string) (io.ReadCloser, error)
	createMultipart   func(ctx context.Context, objectKey, contentType string) (string, error)
	presignUploadPart func(ctx context.Context, objectKey, uploadID string, partNumber int32) (string, time.Duration, error)
	completeMultipart func(ctx context.Context, objectKey, uploadID string, parts []domain.CompletedPart) error
	abortMultipart    func(ctx context.Context, objectKey, uploadID string) error
}

func NewStorageAdapter(
	bucketFn func() string,
	presignUpload func(ctx context.Context, objectKey, contentType string) (string, time.Duration, error),
	presignDown func(ctx context.Context, objectKey string) (string, time.Duration, error),
	remove func(ctx context.Context, objectKey string) error,
	putObject func(ctx context.Context, objectKey, contentType string, r io.Reader, size int64) error,
	getObject func(ctx context.Context, objectKey string) (io.ReadCloser, error),
	createMultipart func(ctx context.Context, objectKey, contentType string) (string, error),
	presignUploadPart func(ctx context.Context, objectKey, uploadID string, partNumber int32) (string, time.Duration, error),
	completeMultipart func(ctx context.Context, objectKey, uploadID string, parts []domain.CompletedPart) error,
	abortMultipart func(ctx context.Context, objectKey, uploadID string) error,
) *StorageAdapter {
	return &StorageAdapter{
		bucketFn:          bucketFn,
		presignUpload:     presignUpload,
		presignDown:       presignDown,
		remove:            remove,
		putObject:         putObject,
		getObject:         getObject,
		createMultipart:   createMultipart,
		presignUploadPart: presignUploadPart,
		completeMultipart: completeMultipart,
		abortMultipart:    abortMultipart,
	}
}

func (a *StorageAdapter) Bucket() string {
	return a.bucketFn()
}

func (a *StorageAdapter) PresignUpload(ctx context.Context, objectKey, contentType string) (string, time.Duration, error) {
	return a.presignUpload(ctx, objectKey, contentType)
}

func (a *StorageAdapter) PresignDownload(ctx context.Context, objectKey string) (string, time.Duration, error) {
	return a.presignDown(ctx, objectKey)
}

func (a *StorageAdapter) RemoveObject(ctx context.Context, objectKey string) error {
	return a.remove(ctx, objectKey)
}

func (a *StorageAdapter) PutObject(ctx context.Context, objectKey, contentType string, r io.Reader, size int64) error {
	return a.putObject(ctx, objectKey, contentType, r, size)
}

func (a *StorageAdapter) GetObject(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	return a.getObject(ctx, objectKey)
}

func (a *StorageAdapter) CreateMultipartUpload(ctx context.Context, objectKey, contentType string) (string, error) {
	return a.createMultipart(ctx, objectKey, contentType)
}

func (a *StorageAdapter) PresignUploadPart(ctx context.Context, objectKey, uploadID string, partNumber int32) (string, time.Duration, error) {
	return a.presignUploadPart(ctx, objectKey, uploadID, partNumber)
}

func (a *StorageAdapter) CompleteMultipartUpload(ctx context.Context, objectKey, uploadID string, parts []domain.CompletedPart) error {
	return a.completeMultipart(ctx, objectKey, uploadID, parts)
}

func (a *StorageAdapter) AbortMultipartUpload(ctx context.Context, objectKey, uploadID string) error {
	return a.abortMultipart(ctx, objectKey, uploadID)
}
