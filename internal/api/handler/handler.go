package handler

import (
	"net/http"

	adminstore "github.com/canopy-network/canopy-indexer/pkg/db/postgres/admin"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Handler holds the dependencies for API handlers
type Handler struct {
	AdminDB    *adminstore.DB
	Logger     *zap.Logger
	AdminToken string
}

// NewHandler creates a new Handler instance
func NewHandler(adminDB *adminstore.DB, logger *zap.Logger, adminToken string) *Handler {
	return &Handler{
		AdminDB:    adminDB,
		Logger:     logger,
		AdminToken: adminToken,
	}
}

// NewRouter creates and configures the HTTP router with all API routes
func (h *Handler) NewRouter() *mux.Router {
	r := mux.NewRouter()

	// Public health check endpoint
	r.HandleFunc("/api/health", h.HandleHealth).Methods(http.MethodGet)

	// Protected chain management endpoints
	r.HandleFunc("/api/chains", h.RequireAuth(h.HandleChainsList)).Methods(http.MethodGet)
	r.HandleFunc("/api/chains", h.RequireAuth(h.HandleChainsUpsert)).Methods(http.MethodPost)
	r.HandleFunc("/api/chains/{id}", h.RequireAuth(h.HandleChainDetail)).Methods(http.MethodGet)
	r.HandleFunc("/api/chains/{id}", h.RequireAuth(h.HandleChainPatch)).Methods(http.MethodPatch)
	r.HandleFunc("/api/chains/{id}", h.RequireAuth(h.HandleChainDelete)).Methods(http.MethodDelete)

	return r
}

// RequireAuth is a middleware that validates the bearer token
func (h *Handler) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		expected := "Bearer " + h.AdminToken

		if auth != expected {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}

		next(w, r)
	}
}

// HandleHealth returns a simple health check response
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
