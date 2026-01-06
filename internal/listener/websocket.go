package listener

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/canopy-network/canopy/lib"
	"github.com/gorilla/websocket"
)

// Config configures the WebSocket listener.
type Config struct {
	URL            string        // Base WebSocket URL (e.g., "wss://node.example.com/rpc")
	ChainID        uint64        // Chain ID to subscribe to
	MaxRetries     int           // Max reconnection attempts (default: 25)
	ReconnectDelay time.Duration // Base delay between reconnects (default: 1s)
}

// BlockHandler is called when a new block is received.
type BlockHandler func(chainID, height uint64)

// Listener subscribes to a Canopy node via WebSocket for new block notifications.
type Listener struct {
	config     Config
	onNewBlock BlockHandler
	conn       *websocket.Conn
	mu         sync.RWMutex

	// Stats (protected by mu)
	connectedAt   time.Time
	messageCount  uint64
	lastMessageAt time.Time
}

// New creates a new WebSocket listener.
func New(config Config, onNewBlock BlockHandler) *Listener {
	if config.MaxRetries <= 0 {
		config.MaxRetries = 25
	}
	if config.ReconnectDelay <= 0 {
		config.ReconnectDelay = time.Second
	}
	return &Listener{
		config:     config,
		onNewBlock: onNewBlock,
	}
}

// Run starts the listener. It blocks until the context is cancelled.
func (l *Listener) Run(ctx context.Context) error {
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

		slog.Info("connecting to canopy node",
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

			slog.Info("websocket connected", "url", wsURL)

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

			slog.Warn("websocket disconnected",
				"err", err,
				"uptime", uptime.Round(time.Second),
				"messages_received", msgCount,
			)

			// Reset attempt counter on successful connection
			attempt = 0
			continue
		}

		slog.Warn("failed to connect to canopy node",
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

// buildURL constructs the WebSocket subscription URL.
func (l *Listener) buildURL() (string, error) {
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

	// Canopy subscription path (matches rpc.SubscribeRCInfoPath)
	basePath := parsed.Path
	fullPath := basePath + "/v1/subscribe-rc-info"

	wsURL := url.URL{
		Scheme:   wsScheme,
		Host:     host,
		Path:     fullPath,
		RawQuery: fmt.Sprintf("chainId=%d", l.config.ChainID),
	}

	return wsURL.String(), nil
}

// listen reads messages from the WebSocket connection.
func (l *Listener) listen(ctx context.Context) error {
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

		// Unmarshal protobuf RootChainInfo
		info := new(lib.RootChainInfo)
		if err := lib.Unmarshal(data, info); err != nil {
			slog.Warn("websocket unmarshal failed",
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

		slog.Info("websocket block received",
			"chain_id", info.RootChainId,
			"height", info.Height,
			"msg_num", msgNum,
		)

		l.onNewBlock(info.RootChainId, info.Height)
	}
}

// Close gracefully closes the WebSocket connection.
func (l *Listener) Close() error {
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
func (l *Listener) IsConnected() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.conn != nil
}

// Stats returns current connection statistics.
func (l *Listener) Stats() (connected bool, uptime time.Duration, messageCount uint64, lastMessage time.Time) {
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
