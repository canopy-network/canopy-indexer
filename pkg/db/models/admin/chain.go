package admin

import (
	"time"
)

const ChainsTableName = "chains"

// ChainColumns defines the schema for the chains table.
var ChainColumns = []ColumnDef{
	{Name: "chain_id"},
	{Name: "chain_name"},
	{Name: "rpc_endpoints"},
	{Name: "paused"},
	{Name: "deleted"},
	{Name: "notes"},
	{Name: "created_at"},
	{Name: "updated_at"},
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
