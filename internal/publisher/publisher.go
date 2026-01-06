package publisher

import (
	"context"
	"encoding/binary"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
)

// Publisher publishes block heights to Redis Streams.
type Publisher struct {
	pub         message.Publisher
	redisClient redis.UniversalClient
	topic       string
}

// New creates a new Publisher.
func New(redisClient redis.UniversalClient, topic string) (*Publisher, error) {
	logger := watermill.NewSlogLogger(nil)

	pub, err := redisstream.NewPublisher(
		redisstream.PublisherConfig{
			Client: redisClient,
		},
		logger,
	)
	if err != nil {
		return nil, err
	}

	return &Publisher{
		pub:         pub,
		redisClient: redisClient,
		topic:       topic,
	}, nil
}

// PublishBlock publishes a block height to the queue.
func (p *Publisher) PublishBlock(ctx context.Context, chainID, height uint64) error {
	start := time.Now()

	// Encode chain_id and height as 16 bytes
	payload := make([]byte, 16)
	binary.BigEndian.PutUint64(payload[0:8], chainID)
	binary.BigEndian.PutUint64(payload[8:16], height)

	msgUUID := watermill.NewUUID()
	msg := message.NewMessage(msgUUID, payload)

	err := p.pub.Publish(p.topic, msg)
	duration := time.Since(start)

	if err != nil {
		slog.Error("redis publish failed",
			"chain_id", chainID,
			"height", height,
			"msg_uuid", msgUUID,
			"duration_ms", duration.Milliseconds(),
			"err", err,
		)
		return err
	}

	slog.Info("redis publish ok",
		"chain_id", chainID,
		"height", height,
		"msg_uuid", msgUUID,
		"duration_ms", duration.Milliseconds(),
	)
	return nil
}

// Close closes the publisher.
func (p *Publisher) Close() error {
	return p.pub.Close()
}

// QueueLength returns the number of messages in the Redis stream.
func (p *Publisher) QueueLength(ctx context.Context) (int64, error) {
	return p.redisClient.XLen(ctx, p.topic).Result()
}

// Topic returns the Redis stream topic name.
func (p *Publisher) Topic() string {
	return p.topic
}
