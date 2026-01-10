package admin

import (
	"context"
	"fmt"

	adminmodels "github.com/canopy-network/canopy-indexer/pkg/db/models/admin"
)

// initIndexProgress creates the index_progress table
// This table tracks indexing progress per height per chain (similar to ClickHouse)
func (db *DB) initIndexProgress(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS index_progress (
			chain_id BIGINT NOT NULL,
			height BIGINT NOT NULL,
			indexed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			indexing_time DOUBLE PRECISION NOT NULL DEFAULT 0,
			indexing_time_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
			indexing_detail TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			PRIMARY KEY (chain_id, height)
		);

		CREATE INDEX IF NOT EXISTS idx_index_progress_chain_height ON index_progress(chain_id, height);
		CREATE INDEX IF NOT EXISTS idx_index_progress_indexed_at ON index_progress(indexed_at);
	`

	return db.Exec(ctx, query)
}

// RecordIndexed records that a height was successfully indexed
// Inserts a record for each height indexed (similar to ClickHouse)
func (db *DB) RecordIndexed(ctx context.Context, chainID uint64, height uint64, indexingTimeMs float64, indexingDetail string) error {
	query := `
		INSERT INTO index_progress (chain_id, height, indexed_at, indexing_time, indexing_time_ms, indexing_detail)
		VALUES ($1, $2, NOW(), $3 / 1000.0, $4, $5)
		ON CONFLICT (chain_id, height) DO UPDATE SET
			indexed_at = NOW(),
			indexing_time = EXCLUDED.indexing_time,
			indexing_time_ms = EXCLUDED.indexing_time_ms,
			indexing_detail = EXCLUDED.indexing_detail
	`

	return db.Exec(ctx, query, chainID, height, indexingTimeMs, indexingTimeMs, indexingDetail)
}

// LastIndexed returns the highest indexed height for a chain
func (db *DB) LastIndexed(ctx context.Context, chainID uint64) (uint64, error) {
	query := `
		SELECT COALESCE(MAX(height), 0)
		FROM index_progress
		WHERE chain_id = $1
	`

	var height uint64
	err := db.QueryRow(ctx, query, chainID).Scan(&height)
	if err != nil {
		return 0, fmt.Errorf("failed to query last indexed height: %w", err)
	}

	return height, nil
}

// FindGaps returns missing [From, To] heights strictly inside observed heights,
// and does NOT include the trailing gap to 'up to'. The caller should add a tail gap separately.
func (db *DB) FindGaps(ctx context.Context, chainID uint64) ([]adminmodels.Gap, error) {
	query := `
		SELECT (prev_h + 1)::BIGINT AS from_h, (h - 1)::BIGINT AS to_h
		FROM (
			SELECT
				height AS h,
				LAG(height) OVER (ORDER BY height) AS prev_h
			FROM index_progress
			WHERE chain_id = $1
			ORDER BY height
		) t
		WHERE prev_h IS NOT NULL AND h > prev_h + 1
		ORDER BY from_h
	`

	rows, err := db.Query(ctx, query, chainID)
	if err != nil {
		return nil, fmt.Errorf("query gaps: %w", err)
	}
	defer rows.Close()

	var gaps []adminmodels.Gap
	for rows.Next() {
		var gap adminmodels.Gap
		if err := rows.Scan(&gap.From, &gap.To); err != nil {
			return nil, fmt.Errorf("scan gap: %w", err)
		}
		gaps = append(gaps, gap)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return gaps, nil
}

// GetCleanableHeights returns heights that have been indexed and are safe to clean from staging.
// Returns heights indexed within the last lookbackHours - these have been promoted to production.
func (db *DB) GetCleanableHeights(ctx context.Context, chainID uint64, lookbackHours int) ([]uint64, error) {
	query := `
		SELECT DISTINCT height
		FROM index_progress
		WHERE chain_id = $1
		  AND indexed_at >= NOW() - INTERVAL '1 hour' * $2
		ORDER BY height
	`

	rows, err := db.Query(ctx, query, chainID, lookbackHours)
	if err != nil {
		return nil, fmt.Errorf("query cleanable heights: %w", err)
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

// IndexProgressHistory returns index progress metrics grouped by time intervals
func (db *DB) IndexProgressHistory(ctx context.Context, chainID uint64, hours, intervalMinutes int) ([]adminmodels.ProgressPoint, error) {
	query := fmt.Sprintf(`
		SELECT
			date_trunc('minute', indexed_at - INTERVAL '%d second' * EXTRACT(second FROM indexed_at)) AS time_bucket,
			MAX(height) AS max_height,
			AVG(indexing_time) AS avg_latency,
			AVG(indexing_time_ms) AS avg_processing_time,
			COUNT(*) AS blocks_indexed
		FROM index_progress
		WHERE chain_id = $1
		  AND indexed_at >= NOW() - INTERVAL '1 hour' * $2
		GROUP BY time_bucket
		ORDER BY time_bucket ASC
	`, intervalMinutes*60) // Convert minutes to seconds for modulo

	rows, err := db.Query(ctx, query, chainID, hours)
	if err != nil {
		return nil, fmt.Errorf("query progress history: %w", err)
	}
	defer rows.Close()

	var points []adminmodels.ProgressPoint
	for rows.Next() {
		var p adminmodels.ProgressPoint
		if err := rows.Scan(&p.TimeBucket, &p.MaxHeight, &p.AvgLatency, &p.AvgProcessingTime, &p.BlocksIndexed); err != nil {
			return nil, fmt.Errorf("scan progress point: %w", err)
		}
		points = append(points, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return points, nil
}

// DeleteIndexProgressForChain deletes all index progress records for a chain
func (db *DB) DeleteIndexProgressForChain(ctx context.Context, chainID uint64) error {
	query := `DELETE FROM index_progress WHERE chain_id = $1`
	return db.Exec(ctx, query, chainID)
}
