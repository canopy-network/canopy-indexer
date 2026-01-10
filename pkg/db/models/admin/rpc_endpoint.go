package admin

import (
	"time"
)

const RPCEndpointsTableName = "rpc_endpoints"

// RPCEndpointColumns defines the schema for the rpc_endpoints table.
var RPCEndpointColumns = []ColumnDef{
	{Name: "chain_id", Type: "UInt64"},
	{Name: "endpoint", Type: "String"},
	{Name: "status", Type: "String"},
	{Name: "height", Type: "UInt64"},
	{Name: "latency_ms", Type: "Float64"},
	{Name: "error", Type: "String"},
	{Name: "updated_at", Type: "DateTime"},
}

// RPCEndpoint represents the health status of a single RPC endpoint.
type RPCEndpoint struct {
	ChainID   uint64    `json:"chain_id" ch:"chain_id"`
	Endpoint  string    `json:"endpoint" ch:"endpoint"`
	Status    string    `json:"status" ch:"status"`         // healthy, unreachable, degraded
	Height    uint64    `json:"height" ch:"height"`         // last known block height
	LatencyMs float64   `json:"latency_ms" ch:"latency_ms"` // response time in milliseconds
	Error     string    `json:"error" ch:"error"`           // last error message (empty if healthy)
	UpdatedAt time.Time `json:"updated_at" ch:"updated_at"`
}
