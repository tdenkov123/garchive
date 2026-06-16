package service

import (
	"context"

	"github.com/tdenkov123/file-metadata-service/internal/domain"
	"github.com/tdenkov123/file-metadata-service/internal/events/kafka"
)

type KafkaEventPublisher struct {
	pub *kafka.Publisher
}

func NewKafkaEventPublisher(pub *kafka.Publisher) *KafkaEventPublisher {
	return &KafkaEventPublisher{pub: pub}
}

func (p *KafkaEventPublisher) PublishFileCreated(ctx context.Context, file domain.FileMetadata) error {
	return p.pub.Publish(ctx, kafka.FileEvent{
		Type:      kafka.EventFileCreated,
		FileID:    file.ID,
		OwnerID:   file.OwnerID,
		ObjectKey: file.ObjectKey,
	})
}

func (p *KafkaEventPublisher) PublishFileReady(ctx context.Context, file domain.FileMetadata) error {
	return p.pub.Publish(ctx, kafka.FileEvent{
		Type:      kafka.EventFileReady,
		FileID:    file.ID,
		OwnerID:   file.OwnerID,
		ObjectKey: file.ObjectKey,
	})
}

func (p *KafkaEventPublisher) PublishFileDeleted(ctx context.Context, file domain.FileMetadata) error {
	return p.pub.Publish(ctx, kafka.FileEvent{
		Type:      kafka.EventFileDeleted,
		FileID:    file.ID,
		OwnerID:   file.OwnerID,
		ObjectKey: file.ObjectKey,
	})
}
