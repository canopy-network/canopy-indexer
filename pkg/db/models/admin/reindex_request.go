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
	{Name: "chain_id"},
	{Name: "height"},
	{Name: "requested_by"},
	{Name: "status"},
	{Name: "workflow_id"},
	{Name: "run_id"},
	{Name: "requested_at"},
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
