package backfill

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GapStats contains statistics about detected gaps.
type GapStats struct {
	TotalExpected uint64 // Total blocks expected in range
	TotalIndexed  uint64 // Blocks already indexed
	TotalMissing  uint64 // Blocks missing
	FirstMissing  uint64 // First missing height (0 if none)
	LastMissing   uint64 // Last missing height (0 if none)
}

// FindMissingHeights returns a slice of missing block heights between start and end.
// Uses generate_series with anti-join for efficient gap detection.
func FindMissingHeights(ctx context.Context, db *pgxpool.Pool, chainID, start, end uint64, limit int) ([]uint64, error) {
	query := `
		SELECT gs.height
		FROM generate_series($1::bigint, $2::bigint) AS gs(height)
		WHERE NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE b.chain_id = $3
			AND b.height = gs.height
		)
		ORDER BY gs.height
		LIMIT $4
	`

	rows, err := db.Query(ctx, query, start, end, chainID, limit)
	if err != nil {
		return nil, fmt.Errorf("query missing heights: %w", err)
	}
	defer rows.Close()

	var heights []uint64
	for rows.Next() {
		var h uint64
		if err := rows.Scan(&h); err != nil {
			return nil, fmt.Errorf("scan height: %w", err)
		}
		heights = append(heights, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return heights, nil
}

// GetGapStats returns statistics about gaps in the indexed blocks.
func GetGapStats(ctx context.Context, db *pgxpool.Pool, chainID, start, end uint64) (*GapStats, error) {
	query := `
		WITH expected AS (
			SELECT COUNT(*) as total FROM generate_series($1::bigint, $2::bigint)
		),
		indexed AS (
			SELECT COUNT(*) as total FROM blocks
			WHERE chain_id = $3 AND height BETWEEN $1 AND $2
		),
		missing AS (
			SELECT gs.height
			FROM generate_series($1::bigint, $2::bigint) AS gs(height)
			WHERE NOT EXISTS (
				SELECT 1 FROM blocks b
				WHERE b.chain_id = $3 AND b.height = gs.height
			)
		),
		missing_stats AS (
			SELECT
				COUNT(*) as total,
				MIN(height) as first_missing,
				MAX(height) as last_missing
			FROM missing
		)
		SELECT
			expected.total,
			indexed.total,
			missing_stats.total,
			COALESCE(missing_stats.first_missing, 0),
			COALESCE(missing_stats.last_missing, 0)
		FROM expected, indexed, missing_stats
	`

	stats := &GapStats{}
	err := db.QueryRow(ctx, query, start, end, chainID).Scan(
		&stats.TotalExpected,
		&stats.TotalIndexed,
		&stats.TotalMissing,
		&stats.FirstMissing,
		&stats.LastMissing,
	)
	if err != nil {
		return nil, fmt.Errorf("query gap stats: %w", err)
	}

	return stats, nil
}

// CountMissingBlocks returns the count of missing blocks without fetching them.
func CountMissingBlocks(ctx context.Context, db *pgxpool.Pool, chainID, start, end uint64) (uint64, error) {
	query := `
		SELECT COUNT(*)
		FROM generate_series($1::bigint, $2::bigint) AS gs(height)
		WHERE NOT EXISTS (
			SELECT 1 FROM blocks b
			WHERE b.chain_id = $3 AND b.height = gs.height
		)
	`

	var count uint64
	err := db.QueryRow(ctx, query, start, end, chainID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count missing blocks: %w", err)
	}

	return count, nil
}
