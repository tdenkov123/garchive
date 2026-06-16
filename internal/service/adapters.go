package service

import (
	"context"
	"time"
)

type StorageAdapter struct {
	bucketFn      func() string
	presignUpload func(ctx context.Context, objectKey, contentType string) (string, time.Duration, error)
	presignDown   func(ctx context.Context, objectKey string) (string, time.Duration, error)
	remove        func(ctx context.Context, objectKey string) error
}

func NewStorageAdapter(
	bucketFn func() string,
	presignUpload func(ctx context.Context, objectKey, contentType string) (string, time.Duration, error),
	presignDown func(ctx context.Context, objectKey string) (string, time.Duration, error),
	remove func(ctx context.Context, objectKey string) error,
) *StorageAdapter {
	return &StorageAdapter{
		bucketFn:      bucketFn,
		presignUpload: presignUpload,
		presignDown:   presignDown,
		remove:        remove,
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
