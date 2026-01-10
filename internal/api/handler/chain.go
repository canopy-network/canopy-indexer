package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	rpcvalidation "github.com/canopy-network/canopy-indexer/internal/rpc"
	adminmodels "github.com/canopy-network/canopy-indexer/pkg/db/models/admin"
	"github.com/canopy-network/canopy-indexer/pkg/utils"
	"github.com/go-jose/go-jose/v4/json"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// HandleChainsList returns all registered chains
// Query param: ?deleted=true to include soft-deleted chains
func (h *Handler) HandleChainsList(w http.ResponseWriter, r *http.Request) {
	includeDeleted := r.URL.Query().Get("deleted") == "true"

	chains, err := h.AdminDB.ListChain(r.Context(), includeDeleted)
	if err != nil {
		h.Logger.Error("failed to list chains", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if chains == nil {
		chains = make([]adminmodels.Chain, 0)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(chains)
}

// HandleChainsUpsert creates or updates a chain
func (h *Handler) HandleChainsUpsert(w http.ResponseWriter, r *http.Request) {
	var chain adminmodels.Chain
	if err := json.NewDecoder(r.Body).Decode(&chain); err != nil {
		h.Logger.Warn("bad json in chain upsert request", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "bad json"})
		return
	}

	// RPC validation and chain ID detection
	if chain.ChainID == 0 {
		// Auto-detect chain ID from RPC endpoints
		if len(chain.RPCEndpoints) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "either chain_id or rpc_endpoints must be provided",
			})
			return
		}

		detectedChainID, err := rpcvalidation.ValidateAndExtractChainID(r.Context(), chain.RPCEndpoints, h.Logger)
		if err != nil {
			h.Logger.Warn("failed to validate RPC endpoints and detect chain ID",
				zap.Error(err),
				zap.Strings("endpoints", chain.RPCEndpoints),
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("failed to validate RPC endpoints and detect chain ID: %v", err),
			})
			return
		}
		chain.ChainID = detectedChainID
	} else {
		// Validate provided chain ID matches RPC endpoints
		if len(chain.RPCEndpoints) > 0 {
			detectedChainID, err := rpcvalidation.ValidateAndExtractChainID(r.Context(), chain.RPCEndpoints, h.Logger)
			if err != nil {
				h.Logger.Warn("failed to validate RPC endpoints",
					zap.Error(err),
					zap.Uint64("provided_chain_id", chain.ChainID),
					zap.Strings("endpoints", chain.RPCEndpoints),
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": fmt.Sprintf("failed to validate RPC endpoints: %v", err),
				})
				return
			}

			if detectedChainID != chain.ChainID {
				h.Logger.Warn("chain ID mismatch",
					zap.Uint64("provided", chain.ChainID),
					zap.Uint64("detected", detectedChainID),
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": fmt.Sprintf("chain ID mismatch: provided %d but RPC endpoints report %d", chain.ChainID, detectedChainID),
				})
				return
			}
		}
	}

	// Apply defaults
	if chain.ChainName == "" {
		chain.ChainName = fmt.Sprintf("Chain %d", chain.ChainID)
	}

	// Insert or update chain
	if err := h.AdminDB.InsertChain(r.Context(), &chain); err != nil {
		h.Logger.Error("failed to insert/update chain", zap.Error(err), zap.Uint64("chain_id", chain.ChainID))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	h.Logger.Info("chain upserted successfully", zap.Uint64("chain_id", chain.ChainID), zap.String("chain_name", chain.ChainName))

	// Return the created/updated chain
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(chain)
}

// HandleChainDetail returns a specific chain by ID
func (h *Handler) HandleChainDetail(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	chainID, parseErr := strconv.ParseUint(id, 10, 64)
	if parseErr != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid chain ID"})
		return
	}

	chain, err := h.AdminDB.GetChain(r.Context(), chainID)
	if err != nil {
		h.Logger.Warn("chain not found", zap.Uint64("chain_id", chainID), zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(chain)
}

// HandleChainPatch updates specific fields of a chain
func (h *Handler) HandleChainPatch(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	chainIDUint, parseErr := strconv.ParseUint(id, 10, 64)
	if parseErr != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid chain ID"})
		return
	}

	// Load current chain
	cur, err := h.AdminDB.GetChain(r.Context(), chainIDUint)
	if err != nil {
		h.Logger.Warn("chain not found for patch", zap.Uint64("chain_id", chainIDUint), zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		return
	}

	// Decode patch request
	var patch adminmodels.Chain
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		h.Logger.Warn("bad json in chain patch request", zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "bad json"})
		return
	}

	// Apply selective updates
	if patch.ChainName != "" {
		cur.ChainName = strings.TrimSpace(patch.ChainName)
	}
	if patch.Notes != "" {
		cur.Notes = strings.TrimSpace(patch.Notes)
	}

	// RPC Endpoints validation
	if patch.RPCEndpoints != nil {
		cleaned := make([]string, 0, len(patch.RPCEndpoints))
		for _, e := range patch.RPCEndpoints {
			e = strings.TrimSpace(e)
			if e != "" {
				cleaned = append(cleaned, e)
			}
		}
		cleaned = utils.Dedup(cleaned)

		// Validate endpoints match chain ID
		if len(cleaned) > 0 {
			detectedChainID, err := rpcvalidation.ValidateAndExtractChainID(r.Context(), cleaned, h.Logger)
			if err != nil {
				h.Logger.Warn("failed to validate RPC endpoints in patch",
					zap.Error(err),
					zap.Uint64("chain_id", chainIDUint),
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": fmt.Sprintf("failed to validate RPC endpoints: %v", err),
				})
				return
			}
			if detectedChainID != chainIDUint {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": fmt.Sprintf("chain ID mismatch: chain is %d but RPC endpoints report %d", chainIDUint, detectedChainID),
				})
				return
			}
		}
		cur.RPCEndpoints = cleaned
	}

	// Paused state
	if patch.Paused != cur.Paused {
		cur.Paused = patch.Paused
	}

	// Persist changes
	if upsertErr := h.AdminDB.InsertChain(r.Context(), cur); upsertErr != nil {
		h.Logger.Error("failed to update chain", zap.Error(upsertErr), zap.Uint64("chain_id", chainIDUint))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": upsertErr.Error()})
		return
	}

	h.Logger.Info("chain patched successfully", zap.Uint64("chain_id", chainIDUint))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"ok": "1"})
}

// HandleChainDelete performs a soft delete on a chain
func (h *Handler) HandleChainDelete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	chainIDUint, parseErr := strconv.ParseUint(id, 10, 64)
	if parseErr != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid chain ID"})
		return
	}

	// Verify chain exists
	chain, err := h.AdminDB.GetChain(r.Context(), chainIDUint)
	if err != nil {
		h.Logger.Warn("chain not found for delete", zap.Uint64("chain_id", chainIDUint), zap.Error(err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "chain not found"})
		return
	}

	// Soft delete: mark deleted=1
	chain.Deleted = 1

	if err := h.AdminDB.InsertChain(r.Context(), chain); err != nil {
		h.Logger.Error("failed to mark chain as deleted", zap.Error(err), zap.Uint64("chain_id", chainIDUint))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to delete chain"})
		return
	}

	h.Logger.Info("chain soft-deleted successfully", zap.Uint64("chain_id", chainIDUint))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"ok": "1"})
}
