package worker

import (
	"context"
	"encoding/binary"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/canopy-network/pgindexer/internal/indexer"
	"github.com/redis/go-redis/v9"
)

// Config configures the worker.
type Config struct {
	RedisClient   redis.UniversalClient
	Indexer       *indexer.Indexer
	Topic         string
	ConsumerGroup string
	Concurrency   int
}

// QueueStats holds queue statistics.
type QueueStats struct {
	StreamLength int64
	Pending      int64
	Consumers    int64
}

// Worker consumes block heights from Redis Streams and indexes them.
type Worker struct {
	router        *message.Router
	indexer       *indexer.Indexer
	redisClient   redis.UniversalClient
	topic         string
	consumerGroup string
}

// New creates a new Worker.
func New(cfg Config) (*Worker, error) {
	logger := watermill.NewSlogLogger(nil)

	sub, err := redisstream.NewSubscriber(
		redisstream.SubscriberConfig{
			Client:        cfg.RedisClient,
			ConsumerGroup: cfg.ConsumerGroup,
		},
		logger,
	)
	if err != nil {
		return nil, err
	}

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, err
	}

	w := &Worker{
		router:        router,
		indexer:       cfg.Indexer,
		redisClient:   cfg.RedisClient,
		topic:         cfg.Topic,
		consumerGroup: cfg.ConsumerGroup,
	}

	router.AddNoPublisherHandler(
		"index-block",
		cfg.Topic,
		sub,
		w.handleBlock,
	)

	return w, nil
}

// handleBlock processes a single block message.
func (w *Worker) handleBlock(msg *message.Message) error {
	start := time.Now()
	msgUUID := msg.UUID

	if len(msg.Payload) < 16 {
		slog.Warn("worker invalid payload",
			"msg_uuid", msgUUID,
			"len", len(msg.Payload),
		)
		return nil // ack invalid messages to avoid infinite retry
	}

	chainID := binary.BigEndian.Uint64(msg.Payload[0:8])
	height := binary.BigEndian.Uint64(msg.Payload[8:16])

	slog.Info("worker indexing start",
		"chain_id", chainID,
		"height", height,
		"msg_uuid", msgUUID,
	)

	ctx := context.Background()
	if err := w.indexer.IndexBlock(ctx, chainID, height); err != nil {
		duration := time.Since(start)
		slog.Error("worker indexing failed",
			"chain_id", chainID,
			"height", height,
			"msg_uuid", msgUUID,
			"duration_ms", duration.Milliseconds(),
			"err", err,
		)
		// Delay before retry to avoid hammering on errors
		time.Sleep(5 * time.Second)
		return err // will be redelivered
	}

	duration := time.Since(start)
	slog.Info("worker indexing done",
		"chain_id", chainID,
		"height", height,
		"msg_uuid", msgUUID,
		"duration_ms", duration.Milliseconds(),
	)
	return nil
}

// Run starts the worker. It blocks until the context is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	return w.router.Run(ctx)
}

// Close closes the worker.
func (w *Worker) Close() error {
	return w.router.Close()
}

// QueueStats returns current queue statistics.
func (w *Worker) QueueStats(ctx context.Context) (QueueStats, error) {
	var stats QueueStats

	// Get stream length
	length, err := w.redisClient.XLen(ctx, w.topic).Result()
	if err != nil {
		return stats, err
	}
	stats.StreamLength = length

	// Get consumer group info
	groups, err := w.redisClient.XInfoGroups(ctx, w.topic).Result()
	if err != nil {
		// Stream might not exist yet
		return stats, nil
	}

	for _, g := range groups {
		if g.Name == w.consumerGroup {
			stats.Pending = g.Pending
			stats.Consumers = g.Consumers
			break
		}
	}

	return stats, nil
}

// LogQueueStats logs current queue statistics.
func (w *Worker) LogQueueStats(ctx context.Context) {
	stats, err := w.QueueStats(ctx)
	if err != nil {
		slog.Warn("worker queue stats error", "err", err)
		return
	}

	slog.Info("worker queue stats",
		"stream_length", stats.StreamLength,
		"pending", stats.Pending,
		"consumers", stats.Consumers,
	)
}
