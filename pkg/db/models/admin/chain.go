package admin

import (
	"time"
)

const ChainsTableName = "chains"

// ChainColumns defines the schema for the chains table.
var ChainColumns = []ColumnDef{
	{Name: "chain_id", Type: "UInt64"},
	{Name: "chain_name", Type: "String"},
	{Name: "rpc_endpoints", Type: "Array(String)"},
	{Name: "paused", Type: "UInt8"},
	{Name: "deleted", Type: "UInt8"},
	{Name: "notes", Type: "String"},
	{Name: "created_at", Type: "DateTime"},
	{Name: "updated_at", Type: "DateTime"},
}

type Chain struct {
	ChainID      uint64    `json:"chain_id" db:"chain_id"`
	ChainName    string    `json:"chain_name" db:"chain_name"`
	RPCEndpoints []string  `json:"rpc_endpoints" db:"rpc_endpoints"`
	Paused       uint8     `json:"paused" db:"paused"`
	Deleted      uint8     `json:"deleted" db:"deleted"`
	Notes        string    `json:"notes,omitempty" db:"notes"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
