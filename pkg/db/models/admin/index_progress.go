package admin

import (
	"time"
)

const IndexProgressTableName = "index_progress"
const IndexProgressAggTableName = "index_progress_agg"
const IndexProgressMvTableName = "index_progress_mv"

// IndexProgressColumns defines the schema for the index_progress table.
var IndexProgressColumns = []ColumnDef{
	{Name: "chain_id", Type: "UInt64"},
	{Name: "height", Type: "UInt64"},
	{Name: "indexed_at", Type: "DateTime64(6)"},
	{Name: "indexing_time", Type: "Float64"},
	{Name: "indexing_time_ms", Type: "Float64"},
	{Name: "indexing_detail", Type: "String"},
}

// IndexProgress Raw indexing progress (one row per indexed height).
type IndexProgress struct {
	ChainID      uint64    `ch:"chain_id"`
	Height       uint64    `ch:"height"`
	IndexedAt    time.Time `ch:"indexed_at"`
	IndexingTime float64   `ch:"indexing_time"` // Time in seconds from block creation to indexing completion

	// Workflow execution timing fields
	IndexingTimeMs float64 `ch:"indexing_time_ms"` // Total indexing time in milliseconds (actual processing time)
	IndexingDetail string  `ch:"indexing_detail"`  // JSON string with breakdown of individual activity timings
}

type ProgressPoint struct {
	TimeBucket        time.Time `ch:"time_bucket"`
	MaxHeight         uint64    `ch:"max_height"`
	AvgLatency        float64   `ch:"avg_latency"`
	AvgProcessingTime float64   `ch:"avg_processing_time"`
	BlocksIndexed     uint64    `ch:"blocks_indexed"`
}
