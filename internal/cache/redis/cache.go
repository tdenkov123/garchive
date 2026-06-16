package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/tdenkov123/file-metadata-service/internal/config"
	"github.com/tdenkov123/file-metadata-service/internal/domain"
)

type Cache struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewCache(cfg *config.Config) *Cache {
	return &Cache{
		client: goredis.NewClient(&goredis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		}),
		ttl: cfg.CacheTTL,
	}
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Cache) GetFile(ctx context.Context, id string) (domain.FileMetadata, bool, error) {
	key := fileKey(id)
	data, err := c.client.Get(ctx, key).Bytes()
	if err == goredis.Nil {
		return domain.FileMetadata{}, false, nil
	}
	if err != nil {
		return domain.FileMetadata{}, false, fmt.Errorf("redis get: %w", err)
	}
	var file domain.FileMetadata
	if err := json.Unmarshal(data, &file); err != nil {
		return domain.FileMetadata{}, false, fmt.Errorf("unmarshal cache: %w", err)
	}
	return file, true, nil
}

func (c *Cache) SetFile(ctx context.Context, file domain.FileMetadata) error {
	data, err := json.Marshal(file)
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}
	return c.client.Set(ctx, fileKey(file.ID), data, c.ttl).Err()
}

func (c *Cache) InvalidateFile(ctx context.Context, id string) error {
	return c.client.Del(ctx, fileKey(id)).Err()
}

func fileKey(id string) string {
	return "file:" + id
}
