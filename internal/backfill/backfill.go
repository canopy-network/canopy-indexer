package backfill

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/canopy-network/canopy-indexer/internal/indexer"
	"github.com/canopy-network/canopy-indexer/pkg/blob"
	"github.com/canopy-network/canopy-indexer/pkg/rpc"
	"github.com/canopy-network/canopy-indexer/pkg/db/postgres"
	"golang.org/x/sync/errgroup"
)

// Result contains the results of a backfill operation.
type Result struct {
	TotalMissing   uint64
	TotalProcessed uint64
	TotalSucceeded uint64
	TotalFailed    uint64
	Duration       time.Duration
	Errors         []error
}

// Backfiller handles backfilling missing blocks.
type Backfiller struct {
	rpc     *rpc.HTTPClient
	db      *postgres.Client
	indexer *indexer.Indexer
	chainID uint64
	config  *Config
}

// New creates a new Backfiller.
func New(rpcClient *rpc.HTTPClient, db *postgres.Client, idx *indexer.Indexer, chainID uint64, cfg *Config) *Backfiller {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Backfiller{
		rpc:     rpcClient,
		db:      db,
		indexer: idx,
		chainID: chainID,
		config:  cfg,
	}
}

// Run executes the backfill operation.
func (b *Backfiller) Run(ctx context.Context) (*Result, error) {
	start := time.Now()
	result := &Result{}

	// Determine range
	startHeight := b.config.StartHeight
	if startHeight == 0 {
		startHeight = 1
	}

	endHeight := b.config.EndHeight
	if endHeight == 0 {
		// Fetch current chain height from RPC
		height, err := b.rpc.ChainHead(ctx)
		if err != nil {
			return nil, fmt.Errorf("get chain head: %w", err)
		}
		endHeight = height
		slog.Info("fetched chain head from RPC", "height", endHeight)
	}

	slog.Info("starting backfill",
		"chain_id", b.chainID,
		"start_height", startHeight,
		"end_height", endHeight,
		"batch_size", b.config.BatchSize,
		"concurrency", b.config.Concurrency,
		"dry_run", b.config.DryRun,
	)

	// Get initial gap stats
	stats, err := GetGapStats(ctx, b.db, b.chainID, startHeight, endHeight)
	if err != nil {
		return nil, fmt.Errorf("get gap stats: %w", err)
	}

	slog.Info("gap analysis complete",
		"total_expected", stats.TotalExpected,
		"total_indexed", stats.TotalIndexed,
		"total_missing", stats.TotalMissing,
		"first_missing", stats.FirstMissing,
		"last_missing", stats.LastMissing,
	)

	result.TotalMissing = stats.TotalMissing

	if stats.TotalMissing == 0 {
		slog.Info("no missing blocks found")
		result.Duration = time.Since(start)
		return result, nil
	}

	if b.config.DryRun {
		slog.Info("dry run complete, no blocks indexed")
		result.Duration = time.Since(start)
		return result, nil
	}

	// Process in batches
	var errorsMu sync.Mutex
	var processed, succeeded, failed atomic.Uint64

	// Start progress reporter
	progressCtx, cancelProgress := context.WithCancel(ctx)
	defer cancelProgress()
	go b.reportProgress(progressCtx, stats.TotalMissing, &processed, &succeeded, &failed)

	// Process batches
	currentStart := startHeight
	for currentStart <= endHeight {
		select {
		case <-ctx.Done():
			result.TotalProcessed = processed.Load()
			result.TotalSucceeded = succeeded.Load()
			result.TotalFailed = failed.Load()
			result.Duration = time.Since(start)
			return result, ctx.Err()
		default:
		}

		// Find missing heights for this batch
		heights, err := FindMissingHeights(ctx, b.db, b.chainID, currentStart, endHeight, b.config.BatchSize)
		if err != nil {
			return nil, fmt.Errorf("find missing heights: %w", err)
		}

		if len(heights) == 0 {
			break
		}

		slog.Debug("processing batch",
			"batch_start", heights[0],
			"batch_end", heights[len(heights)-1],
			"batch_size", len(heights),
		)

		// Process batch with limited concurrency
		g, gCtx := errgroup.WithContext(ctx)
		g.SetLimit(b.config.Concurrency)

		for _, height := range heights {
			g.Go(func() error {
				processed.Add(1)

				var err error
				if b.config.UseBlobs {
					err = b.indexBlockWithBlob(gCtx, height)
				} else {
					err = b.indexer.IndexBlock(gCtx, b.chainID, height)
				}

				if err != nil {
					failed.Add(1)
					errorsMu.Lock()
					result.Errors = append(result.Errors, fmt.Errorf("height %d: %w", height, err))
					errorsMu.Unlock()
					slog.Error("failed to index block",
						"chain_id", b.chainID,
						"height", height,
						"use_blobs", b.config.UseBlobs,
						"err", err,
					)
					// Continue with other blocks, don't fail entire backfill
					return nil
				}

				succeeded.Add(1)
				return nil
			})
		}

		if err := g.Wait(); err != nil {
			// Context cancelled
			break
		}

		// Move start past all found heights
		if len(heights) > 0 {
			currentStart = heights[len(heights)-1] + 1
		}
	}

	result.TotalProcessed = processed.Load()
	result.TotalSucceeded = succeeded.Load()
	result.TotalFailed = failed.Load()
	result.Duration = time.Since(start)

	slog.Info("backfill complete",
		"total_missing", result.TotalMissing,
		"total_processed", result.TotalProcessed,
		"total_succeeded", result.TotalSucceeded,
		"total_failed", result.TotalFailed,
		"duration", result.Duration,
	)

	return result, nil
}

// reportProgress logs progress at regular intervals.
func (b *Backfiller) reportProgress(ctx context.Context, total uint64, processed, succeeded, failed *atomic.Uint64) {
	ticker := time.NewTicker(b.config.ProgressInterval)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p := processed.Load()
			s := succeeded.Load()
			f := failed.Load()

			elapsed := time.Since(startTime)
			rate := float64(p) / elapsed.Seconds()

			var eta time.Duration
			if rate > 0 && p < total {
				remaining := total - p
				eta = time.Duration(float64(remaining)/rate) * time.Second
			}

			progress := float64(p) / float64(total) * 100

			slog.Info("backfill progress",
				"processed", p,
				"total", total,
				"progress_pct", fmt.Sprintf("%.1f%%", progress),
				"succeeded", s,
				"failed", f,
				"rate_per_sec", fmt.Sprintf("%.1f", rate),
				"eta", eta.Round(time.Second),
			)
		}
	}
}

// indexBlockWithBlob fetches a blob and indexes the block from it.
func (b *Backfiller) indexBlockWithBlob(ctx context.Context, height uint64) error {
	blobs, err := b.rpc.Blob(ctx, height)
	if err != nil {
		return fmt.Errorf("fetch blob: %w", err)
	}

	data, err := blob.Decode(blobs, b.chainID)
	if err != nil {
		return fmt.Errorf("decode blob: %w", err)
	}

	return b.indexer.IndexBlockWithData(ctx, data)
}

// CheckHealth performs a quick gap check and returns stats.
func (b *Backfiller) CheckHealth(ctx context.Context) (*GapStats, error) {
	endHeight, err := b.rpc.ChainHead(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain head: %w", err)
	}

	startHeight := b.config.StartHeight
	if startHeight == 0 {
		startHeight = 1
	}

	return GetGapStats(ctx, b.db, b.chainID, startHeight, endHeight)
}
