package minio

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	gominio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/tdenkov123/file-metadata-service/internal/config"
	"github.com/tdenkov123/file-metadata-service/internal/domain"
)

type Client struct {
	client     *gominio.Client
	core       *gominio.Core
	bucket     string
	presignTTL time.Duration
}

func NewClient(cfg *config.Config) (*Client, error) {
	client, err := gominio.New(cfg.MinioEndpoint, &gominio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}
	return &Client{
		client:     client,
		core:       &gominio.Core{Client: client},
		bucket:     cfg.MinioBucket,
		presignTTL: cfg.PresignTTL,
	}, nil
}

func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := c.client.MakeBucket(ctx, c.bucket, gominio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
	}
	return nil
}

func (c *Client) PresignUpload(ctx context.Context, objectKey, contentType string) (*url.URL, time.Duration, error) {
	reqParams := make(url.Values)
	reqParams.Set("Content-Type", contentType)
	u, err := c.client.PresignedPutObject(ctx, c.bucket, objectKey, c.presignTTL)
	if err != nil {
		return nil, 0, fmt.Errorf("presign upload: %w", err)
	}
	return u, c.presignTTL, nil
}

func (c *Client) PresignDownload(ctx context.Context, objectKey string) (*url.URL, time.Duration, error) {
	u, err := c.client.PresignedGetObject(ctx, c.bucket, objectKey, c.presignTTL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("presign download: %w", err)
	}
	return u, c.presignTTL, nil
}

func (c *Client) RemoveObject(ctx context.Context, objectKey string) error {
	return c.client.RemoveObject(ctx, c.bucket, objectKey, gominio.RemoveObjectOptions{})
}

func (c *Client) CreateMultipartUpload(ctx context.Context, objectKey, contentType string) (string, error) {
	uploadID, err := c.core.NewMultipartUpload(ctx, c.bucket, objectKey, gominio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("create multipart upload: %w", err)
	}
	return uploadID, nil
}

func (c *Client) PresignUploadPart(ctx context.Context, objectKey, uploadID string, partNumber int32) (*url.URL, time.Duration, error) {
	reqParams := make(url.Values)
	reqParams.Set("uploadId", uploadID)
	reqParams.Set("partNumber", strconv.Itoa(int(partNumber)))
	u, err := c.client.Presign(ctx, http.MethodPut, c.bucket, objectKey, c.presignTTL, reqParams)
	if err != nil {
		return nil, 0, fmt.Errorf("presign upload part: %w", err)
	}
	return u, c.presignTTL, nil
}

func (c *Client) CompleteMultipartUpload(ctx context.Context, objectKey, uploadID string, parts []domain.CompletedPart) error {
	completeParts := make([]gominio.CompletePart, len(parts))
	for i, p := range parts {
		completeParts[i] = gominio.CompletePart{
			PartNumber: int(p.PartNumber),
			ETag:       p.ETag,
		}
	}
	_, err := c.core.CompleteMultipartUpload(ctx, c.bucket, objectKey, uploadID, completeParts, gominio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("complete multipart upload: %w", err)
	}
	return nil
}

func (c *Client) AbortMultipartUpload(ctx context.Context, objectKey, uploadID string) error {
	err := c.core.AbortMultipartUpload(ctx, c.bucket, objectKey, uploadID)
	if err != nil {
		return fmt.Errorf("abort multipart upload: %w", err)
	}
	return nil
}

func (c *Client) Bucket() string {
	return c.bucket
}
