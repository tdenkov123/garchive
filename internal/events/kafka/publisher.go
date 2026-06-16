package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"github.com/tdenkov123/file-metadata-service/internal/config"
)

type EventType string

const (
	EventFileCreated  EventType = "file.created"
	EventFileReady    EventType = "file.ready"
	EventFileDeleted  EventType = "file.deleted"
)

type FileEvent struct {
	Type      EventType `json:"type"`
	FileID    string    `json:"file_id"`
	OwnerID   string    `json:"owner_id"`
	ObjectKey string    `json:"object_key"`
	Timestamp time.Time `json:"timestamp"`
}

type Publisher struct {
	writer *kafkago.Writer
}

func NewPublisher(cfg *config.Config) *Publisher {
	return &Publisher{
		writer: &kafkago.Writer{
			Addr:     kafkago.TCP(cfg.KafkaBrokers...),
			Topic:    cfg.KafkaTopic,
			Balancer: &kafkago.LeastBytes{},
		},
	}
}

func (p *Publisher) Publish(ctx context.Context, event FileEvent) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	msg := kafkago.Message{
		Key:   []byte(event.FileID),
		Value: payload,
		Time:  event.Timestamp,
	}
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("write kafka message: %w", err)
	}
	return nil
}

func (p *Publisher) Close() error {
	return p.writer.Close()
}
