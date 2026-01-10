package rpc

import (
	"context"
	"fmt"
	"time"

	"github.com/canopy-network/canopy-indexer/pkg/rpc"
	"go.uber.org/zap"
)

// FetchChainID extracts the chain ID (network ID) from a blockchain RPC endpoint.
// It queries the genesis block (height 0) and extracts the network_id from the block header.
//
// This is used during chain creation to automatically detect and validate the chain ID
// from the blockchain's own data rather than accepting it from user input.
//
// The function uses a 5-second timeout context to prevent hanging on slow/unreachable endpoints.
// Returns the chain ID as uint64, or an error if the endpoint is unreachable or returns invalid data.
func FetchChainID(ctx context.Context, rpcURL string) (uint64, error) {
	// Create HTTP client with single endpoint
	opts := rpc.Opts{
		Endpoints: []string{rpcURL},
		Timeout:   5 * time.Second,
		RPS:       20,
		Burst:     40,
	}
	client := rpc.NewHTTPWithOpts(opts)

	// Query genesis block (height 1) to get chain ID from network_id
	// Note: height 0 returns latest block in Canopy RPC
	block, err := client.BlockByHeight(ctx, 1)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch genesis block from %s: %w", rpcURL, err)
	}

	// Validate that we got a valid block header
	if block.BlockHeader == nil {
		return 0, fmt.Errorf("invalid block header (nil) returned from %s", rpcURL)
	}

	// Extract network_id (chain_id)
	chainID := uint64(block.BlockHeader.NetworkId)
	if chainID == 0 {
		return 0, fmt.Errorf("invalid chain ID (0) returned from %s", rpcURL)
	}

	return chainID, nil
}

// ValidateAndExtractChainID validates RPC endpoints and extracts a consistent chain ID.
// It queries all provided endpoints in parallel, tolerates partial failures (at least one must succeed),
// and ensures all successful responses return the same chain ID.
//
// Edge cases handled:
// - Partial failures: OK if at least 1 endpoint succeeds
// - Timeouts: Each endpoint gets 5s timeout
// - Mismatches: FAIL if different chain IDs are returned
//
// Returns the validated chain ID or an error describing the failure.
func ValidateAndExtractChainID(ctx context.Context, endpoints []string, logger *zap.Logger) (uint64, error) {
	if len(endpoints) == 0 {
		return 0, fmt.Errorf("no RPC endpoints provided")
	}

	type result struct {
		endpoint string
		chainID  uint64
		err      error
	}

	results := make(chan result, len(endpoints))

	// Query all endpoints in parallel with 5s timeout each
	for _, endpoint := range endpoints {
		go func(endpoint string) {
			// Create timeout context for this specific endpoint
			timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			chainID, err := FetchChainID(timeoutCtx, endpoint)
			results <- result{
				endpoint: endpoint,
				chainID:  chainID,
				err:      err,
			}
		}(endpoint) // capture endpoint in the loop
	}

	// Collect results
	successfulResults := make(map[string]uint64)
	var errs []error

	for i := 0; i < len(endpoints); i++ {
		res := <-results
		if res.err != nil {
			logger.Warn("RPC endpoint failed during chain ID validation",
				zap.String("endpoint", res.endpoint),
				zap.Error(res.err),
			)
			errs = append(errs, fmt.Errorf("%s: %w", res.endpoint, res.err))
			continue
		}
		successfulResults[res.endpoint] = res.chainID
	}

	// Check if all endpoints failed
	if len(successfulResults) == 0 {
		return 0, fmt.Errorf("all RPC endpoints failed: %v", errs)
	}

	// Validate consistency: all successful endpoints must return the same chain ID
	var expectedChainID uint64
	first := true
	for endpoint, chainID := range successfulResults {
		if first {
			expectedChainID = chainID
			first = false
			logger.Info("Chain ID detected from RPC",
				zap.String("endpoint", endpoint),
				zap.Uint64("chain_id", chainID),
			)
			continue
		}

		if chainID != expectedChainID {
			return 0, fmt.Errorf("chain ID mismatch: endpoint %s returned %d, but another endpoint returned %d earlier",
				endpoint, chainID, expectedChainID)
		}
	}

	// Log summary
	logger.Info("Chain ID validation successful",
		zap.Uint64("chain_id", expectedChainID),
		zap.Int("successful_endpoints", len(successfulResults)),
		zap.Int("failed_endpoints", len(errs)),
	)

	return expectedChainID, nil
}
