package admin

import (
	"time"
)

const ReindexRequestsTableName = "reindex_requests"

const (
	ReindexRequestStatusQueued = "queued"
)

// ReindexRequestColumns defines the schema for the reindex_requests table.
var ReindexRequestColumns = []ColumnDef{
	{Name: "chain_id", Type: "UInt64"},
	{Name: "height", Type: "UInt64"},
	{Name: "requested_by", Type: "String"},
	{Name: "status", Type: "String"},
	{Name: "workflow_id", Type: "String"},
	{Name: "run_id", Type: "String"},
	{Name: "requested_at", Type: "DateTime"},
}

type ReindexRequest struct {
	ChainID     uint64    `ch:"chain_id"`
	Height      uint64    `ch:"height"`
	RequestedBy string    `ch:"requested_by"`
	Status      string    `ch:"status"`
	WorkflowID  string    `ch:"workflow_id"`
	RunID       string    `ch:"run_id"`
	RequestedAt time.Time `ch:"requested_at"`
}
