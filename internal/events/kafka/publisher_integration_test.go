//go:build integration

package kafka_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
	tckafka "github.com/testcontainers/testcontainers-go/modules/kafka"

	"github.com/tdenkov123/file-metadata-service/internal/config"
	kafkapub "github.com/tdenkov123/file-metadata-service/internal/events/kafka"
)

func ensureTopic(t *testing.T, ctx context.Context, broker, topic string) {
	t.Helper()

	conn, err := kafkago.DialContext(ctx, "tcp", broker)
	require.NoError(t, err)
	defer conn.Close()

	err = conn.CreateTopics(kafkago.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil {
		require.Contains(t, err.Error(), "Topic already exists")
	}

	require.Eventually(t, func() bool {
		metaConn, dialErr := kafkago.DialContext(ctx, "tcp", broker)
		if dialErr != nil {
			return false
		}
		defer metaConn.Close()

		partitions, readErr := metaConn.ReadPartitions(topic)
		return readErr == nil && len(partitions) > 0
	}, 30*time.Second, 500*time.Millisecond, "topic %q was not visible in cluster metadata", topic)
}

func TestPublisher_PublishFileReady(t *testing.T) {
	ctx := context.Background()

	container, err := tckafka.Run(ctx, "confluentinc/confluent-local:7.5.0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	brokers, err := container.Brokers(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, brokers)

	topic := "file.events-" + uuid.NewString()
	ensureTopic(t, ctx, brokers[0], topic)

	pub := kafkapub.NewPublisher(&config.Config{
		KafkaBrokers: brokers,
		KafkaTopic:   topic,
	})
	t.Cleanup(func() { _ = pub.Close() })

	fileID := uuid.NewString()
	err = pub.Publish(ctx, kafkapub.FileEvent{
		Type:      kafkapub.EventFileReady,
		FileID:    fileID,
		OwnerID:   "user-1",
		ObjectKey: "user-1/" + fileID + "/doc.pdf",
		Timestamp: time.Now().UTC(),
	})
	require.NoError(t, err)

	reader := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: "integration-test-" + uuid.NewString(),
	})
	t.Cleanup(func() { _ = reader.Close() })

	msgCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	msg, err := reader.ReadMessage(msgCtx)
	require.NoError(t, err)
	require.Contains(t, string(msg.Value), fileID)
}
