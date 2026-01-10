package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/canopy-network/canopy-indexer/internal/api/handler"
	adminstore "github.com/canopy-network/canopy-indexer/pkg/db/postgres/admin"
	"go.uber.org/zap"
)

// Server wraps the HTTP server for the chain management API
type Server struct {
	httpServer *http.Server
	logger     *zap.Logger
}

// NewServer creates a new API server instance
func NewServer(adminDB *adminstore.DB, logger *zap.Logger, addr string, adminToken string) (*Server, error) {
	h := handler.NewHandler(adminDB, logger, adminToken)
	router := h.NewRouter()

	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &Server{
		httpServer: server,
		logger:     logger,
	}, nil
}

// Run starts the HTTP server and blocks until the context is canceled
func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("starting HTTP API server", zap.String("addr", s.httpServer.Addr))

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		s.logger.Info("shutting down HTTP API server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	case err := <-errChan:
		return err
	}
}
