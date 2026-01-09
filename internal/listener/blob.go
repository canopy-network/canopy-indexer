package listener

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/gorilla/websocket"
)

// BlobConfig configures the IndexerBlob WebSocket listener.
type BlobConfig struct {
	URL            string        // Base WebSocket URL (e.g., "wss://node.example.com")
	ChainID        uint64        // Chain ID to subscribe to
	MaxRetries     int           // Max reconnection attempts (default: 25)
	ReconnectDelay time.Duration // Base delay between reconnects (default: 1s)
}

// BlobHandler is called when a new IndexerBlob is received.
type BlobHandler func(blob *fsm.IndexerBlob) error

// BlobListener subscribes to a Canopy node via WebSocket for IndexerBlob messages.
type BlobListener struct {
	config BlobConfig
	onBlob BlobHandler
	conn   *websocket.Conn
	mu     sync.RWMutex

	// Stats (protected by mu)
	connectedAt   time.Time
	messageCount  uint64
	lastMessageAt time.Time
}

// NewBlobListener creates a new IndexerBlob WebSocket listener.
func NewBlobListener(config BlobConfig, onBlob BlobHandler) *BlobListener {
	if config.MaxRetries <= 0 {
		config.MaxRetries = 25
	}
	if config.ReconnectDelay <= 0 {
		config.ReconnectDelay = time.Second
	}
	return &BlobListener{
		config: config,
		onBlob: onBlob,
	}
}

// Run starts the listener. It blocks until the context is cancelled.
func (l *BlobListener) Run(ctx context.Context) error {
	wsURL, err := l.buildURL()
	if err != nil {
		return fmt.Errorf("build websocket url: %w", err)
	}

	for attempt := 0; attempt < l.config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		slog.Info("connecting to canopy node for blobs",
			"attempt", attempt+1,
			"max_retries", l.config.MaxRetries,
			"url", wsURL,
		)

		conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
		if err == nil {
			l.mu.Lock()
			l.conn = conn
			l.connectedAt = time.Now()
			l.messageCount = 0
			l.mu.Unlock()

			slog.Info("blob websocket connected", "url", wsURL)

			err = l.listen(ctx)
			if err == context.Canceled {
				return err
			}

			l.mu.Lock()
			uptime := time.Since(l.connectedAt)
			msgCount := l.messageCount
			if l.conn != nil {
				_ = l.conn.Close()
				l.conn = nil
			}
			l.mu.Unlock()

			slog.Warn("blob websocket disconnected",
				"err", err,
				"uptime", uptime.Round(time.Second),
				"messages_received", msgCount,
			)

			// Reset attempt counter on successful connection
			attempt = 0
			continue
		}

		slog.Warn("failed to connect to canopy node for blobs",
			"attempt", attempt+1,
			"err", err,
		)

		// Linear backoff
		delay := time.Duration(attempt+1) * l.config.ReconnectDelay
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("max retries (%d) reached", l.config.MaxRetries)
}

// buildURL constructs the WebSocket subscription URL for IndexerBlob.
func (l *BlobListener) buildURL() (string, error) {
	parsed, err := url.Parse(l.config.URL)
	if err != nil {
		return "", err
	}

	host := parsed.Host
	if host == "" {
		host = parsed.Path
	}

	wsScheme := "ws"
	if parsed.Scheme == "https" || parsed.Scheme == "wss" {
		wsScheme = "wss"
	}

	// Canopy subscription path for IndexerBlob
	basePath := parsed.Path
	fullPath := basePath + "/v1/subscribe-indexer-blobs"

	wsURL := url.URL{
		Scheme:   wsScheme,
		Host:     host,
		Path:     fullPath,
		RawQuery: fmt.Sprintf("chainId=%d", l.config.ChainID),
	}

	return wsURL.String(), nil
}

// listen reads IndexerBlob messages from the WebSocket connection.
func (l *BlobListener) listen(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, data, err := l.conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read message: %w", err)
		}

		// Unmarshal protobuf IndexerBlob
		blob := new(fsm.IndexerBlob)
		if err := lib.Unmarshal(data, blob); err != nil {
			slog.Warn("blob websocket unmarshal failed",
				"err", err,
				"data_len", len(data),
			)
			continue
		}

		// Update stats
		l.mu.Lock()
		l.messageCount++
		l.lastMessageAt = time.Now()
		msgNum := l.messageCount
		l.mu.Unlock()

		// Extract height from block for logging
		var height uint64
		if len(blob.Block) > 0 {
			var block lib.BlockResult
			if err := lib.Unmarshal(blob.Block, &block); err == nil && block.BlockHeader != nil {
				height = block.BlockHeader.Height
			}
		}

		slog.Info("blob received",
			"chain_id", l.config.ChainID,
			"height", height,
			"msg_num", msgNum,
			"size_bytes", len(data),
			"accounts", len(blob.Accounts),
			"validators", len(blob.Validators),
			"pools", len(blob.Pools),
			"dex_batches", len(blob.DexBatches),
		)

		// Call the handler
		if err := l.onBlob(blob); err != nil {
			slog.Error("blob handler failed",
				"height", height,
				"err", err,
			)
			// Continue processing - don't disconnect on handler errors
		}
	}
}

// Close gracefully closes the WebSocket connection.
func (l *BlobListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.conn != nil {
		err := l.conn.Close()
		l.conn = nil
		return err
	}
	return nil
}

// IsConnected returns whether the listener is currently connected.
func (l *BlobListener) IsConnected() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.conn != nil
}

// Stats returns current connection statistics.
func (l *BlobListener) Stats() (connected bool, uptime time.Duration, messageCount uint64, lastMessage time.Time) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	connected = l.conn != nil
	if connected {
		uptime = time.Since(l.connectedAt)
	}
	messageCount = l.messageCount
	lastMessage = l.lastMessageAt
	return
}
