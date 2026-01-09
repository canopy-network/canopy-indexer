package listener

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/gorilla/websocket"
)

// SnapshotConfig configures the IndexerSnapshot WebSocket listener.
type SnapshotConfig struct {
	URL            string        // Base WebSocket URL (e.g., "wss://node.example.com")
	ChainID        uint64        // Chain ID to subscribe to
	MaxRetries     int           // Max reconnection attempts (default: 25)
	ReconnectDelay time.Duration // Base delay between reconnects (default: 1s)
}

// SnapshotHandler is called when a new IndexerSnapshot is received.
type SnapshotHandler func(snapshot *lib.IndexerSnapshot) error

// SnapshotListener subscribes to a Canopy node via WebSocket for IndexerSnapshot messages.
type SnapshotListener struct {
	config     SnapshotConfig
	onSnapshot SnapshotHandler
	conn       *websocket.Conn
	mu         sync.RWMutex

	// Stats (protected by mu)
	connectedAt   time.Time
	messageCount  uint64
	lastMessageAt time.Time
}

// NewSnapshotListener creates a new IndexerSnapshot WebSocket listener.
func NewSnapshotListener(config SnapshotConfig, onSnapshot SnapshotHandler) *SnapshotListener {
	if config.MaxRetries <= 0 {
		config.MaxRetries = 25
	}
	if config.ReconnectDelay <= 0 {
		config.ReconnectDelay = time.Second
	}
	return &SnapshotListener{
		config:     config,
		onSnapshot: onSnapshot,
	}
}

// Run starts the listener. It blocks until the context is cancelled.
func (l *SnapshotListener) Run(ctx context.Context) error {
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

		slog.Info("connecting to canopy node for snapshots",
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

			slog.Info("snapshot websocket connected", "url", wsURL)

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

			slog.Warn("snapshot websocket disconnected",
				"err", err,
				"uptime", uptime.Round(time.Second),
				"messages_received", msgCount,
			)

			// Reset attempt counter on successful connection
			attempt = 0
			continue
		}

		slog.Warn("failed to connect to canopy node for snapshots",
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

// buildURL constructs the WebSocket subscription URL for IndexerSnapshot.
func (l *SnapshotListener) buildURL() (string, error) {
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

	// Canopy subscription path for IndexerSnapshot
	basePath := parsed.Path
	fullPath := basePath + "/v1/subscribe-block-data"

	wsURL := url.URL{
		Scheme:   wsScheme,
		Host:     host,
		Path:     fullPath,
		RawQuery: fmt.Sprintf("chainId=%d", l.config.ChainID),
	}

	return wsURL.String(), nil
}

// listen reads IndexerSnapshot messages from the WebSocket connection.
func (l *SnapshotListener) listen(ctx context.Context) error {
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

		// Unmarshal protobuf IndexerSnapshot
		snapshot := new(lib.IndexerSnapshot)
		if err := lib.Unmarshal(data, snapshot); err != nil {
			slog.Warn("snapshot websocket unmarshal failed",
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

		slog.Info("snapshot received",
			"chain_id", l.config.ChainID,
			"height", snapshot.Height,
			"msg_num", msgNum,
			"size_bytes", len(data),
			"txs", len(snapshot.Transactions),
			"events", len(snapshot.Events),
			"accounts", len(snapshot.Accounts),
			"validators", len(snapshot.ValidatorsCurrent),
			"pools", len(snapshot.PoolsCurrent),
			"dex_batches", len(snapshot.DexBatchesCurrent),
		)

		// Log all accounts and balances
		for i, accBytes := range snapshot.Accounts {
			acc := new(fsm.Account)
			if err := lib.Unmarshal(accBytes, acc); err != nil {
				slog.Warn("failed to unmarshal account", "index", i, "err", err)
				continue
			}
			slog.Info("account",
				"height", snapshot.Height,
				"address", hex.EncodeToString(acc.Address),
				"balance", acc.Amount,
			)
		}

		// Call the handler
		if err := l.onSnapshot(snapshot); err != nil {
			slog.Error("snapshot handler failed",
				"height", snapshot.Height,
				"err", err,
			)
			// Continue processing - don't disconnect on handler errors
		}
	}
}

// Close gracefully closes the WebSocket connection.
func (l *SnapshotListener) Close() error {
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
func (l *SnapshotListener) IsConnected() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.conn != nil
}

// Stats returns current connection statistics.
func (l *SnapshotListener) Stats() (connected bool, uptime time.Duration, messageCount uint64, lastMessage time.Time) {
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
