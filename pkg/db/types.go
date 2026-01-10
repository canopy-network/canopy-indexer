package db

import indexermodels "github.com/canopy-network/canopy-indexer/pkg/db/models/indexer"

// Column represents a database column with its name and type information.
type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ChainRPCEndpoints represents RPC endpoints for a chain.
type ChainRPCEndpoints struct {
	ChainID      uint64   `ch:"chain_id" db:"chain_id"`
	RPCEndpoints []string `ch:"rpc_endpoints" db:"rpc_endpoints"`
}

// TableConfig defines the schema configuration for a cross-chain global table.
type TableConfig struct {
	TableName        string   // Base table name (e.g., "accounts")
	PrimaryKey       []string // ORDER BY columns
	HasAddressColumn bool     // Whether to add bloom filter on address
	SchemaSQL        string   // Full CREATE TABLE schema (columns only, without ENGINE/ORDER BY)
	ColumnNames      []string // Explicit column names for INSERT/SELECT
}

// GetTableConfigs returns the configuration for the entities to sync across chains.
// Each config is derived from the ColumnDef definitions in pkg/db/models/indexer/*.go
func GetTableConfigs() []TableConfig {
	return []TableConfig{
		{
			TableName:        indexermodels.AccountsProductionTableName,
			PrimaryKey:       []string{"chain_id", "address"},
			HasAddressColumn: true,
			SchemaSQL:        indexermodels.ColumnsToSchemaSQL(indexermodels.FilterCrossChainColumns(indexermodels.AccountColumns)),
			ColumnNames:      indexermodels.GetCrossChainColumnNames(indexermodels.AccountColumns),
		},
		{
			TableName:        indexermodels.ValidatorsProductionTableName,
			PrimaryKey:       []string{"chain_id", "address"},
			HasAddressColumn: true,
			SchemaSQL:        indexermodels.ColumnsToSchemaSQL(indexermodels.FilterCrossChainColumns(indexermodels.ValidatorColumns)),
			ColumnNames:      indexermodels.GetCrossChainColumnNames(indexermodels.ValidatorColumns),
		},
		{
			TableName:        indexermodels.ValidatorNonSigningInfoProductionTableName,
			PrimaryKey:       []string{"chain_id", "address"},
			HasAddressColumn: true,
			SchemaSQL:        indexermodels.ColumnsToSchemaSQL(indexermodels.FilterCrossChainColumns(indexermodels.ValidatorNonSigningInfoColumns)),
			ColumnNames:      indexermodels.GetCrossChainColumnNames(indexermodels.ValidatorNonSigningInfoColumns),
		},
		{
			TableName:        indexermodels.ValidatorDoubleSigningInfoProductionTableName,
			PrimaryKey:       []string{"chain_id", "address"},
			HasAddressColumn: true,
			SchemaSQL:        indexermodels.ColumnsToSchemaSQL(indexermodels.FilterCrossChainColumns(indexermodels.ValidatorDoubleSigningInfoColumns)),
			ColumnNames:      indexermodels.GetCrossChainColumnNames(indexermodels.ValidatorDoubleSigningInfoColumns),
		},
		{
			TableName:        indexermodels.PoolsProductionTableName,
			PrimaryKey:       []string{"chain_id", "pool_id"},
			HasAddressColumn: false,
			SchemaSQL:        indexermodels.ColumnsToCrossChainSchemaSQL(indexermodels.PoolColumns),
			ColumnNames:      indexermodels.GetCrossChainColumnNames(indexermodels.PoolColumns),
		},
		{
			TableName:        indexermodels.PoolPointsByHolderProductionTableName,
			PrimaryKey:       []string{"chain_id", "pool_id", "address"},
			HasAddressColumn: true,
			SchemaSQL:        indexermodels.ColumnsToSchemaSQL(indexermodels.FilterCrossChainColumns(indexermodels.PoolPointsByHolderColumns)),
			ColumnNames:      indexermodels.GetCrossChainColumnNames(indexermodels.PoolPointsByHolderColumns),
		},
		{
			TableName:        indexermodels.OrdersProductionTableName,
			PrimaryKey:       []string{"chain_id", "order_id"},
			HasAddressColumn: false,
			SchemaSQL:        indexermodels.ColumnsToSchemaSQL(indexermodels.FilterCrossChainColumns(indexermodels.OrderColumns)),
			ColumnNames:      indexermodels.GetCrossChainColumnNames(indexermodels.OrderColumns),
		},
		{
			TableName:        indexermodels.DexOrdersProductionTableName,
			PrimaryKey:       []string{"chain_id", "order_id"},
			HasAddressColumn: false,
			SchemaSQL:        indexermodels.ColumnsToSchemaSQL(indexermodels.FilterCrossChainColumns(indexermodels.DexOrderColumns)),
			ColumnNames:      indexermodels.GetCrossChainColumnNames(indexermodels.DexOrderColumns),
		},
		{
			TableName:        indexermodels.DexDepositsProductionTableName,
			PrimaryKey:       []string{"chain_id", "order_id"},
			HasAddressColumn: false,
			SchemaSQL:        indexermodels.ColumnsToSchemaSQL(indexermodels.FilterCrossChainColumns(indexermodels.DexDepositColumns)),
			ColumnNames:      indexermodels.GetCrossChainColumnNames(indexermodels.DexDepositColumns),
		},
		{
			TableName:        indexermodels.DexWithdrawalsProductionTableName,
			PrimaryKey:       []string{"chain_id", "order_id"},
			HasAddressColumn: false,
			SchemaSQL:        indexermodels.ColumnsToSchemaSQL(indexermodels.FilterCrossChainColumns(indexermodels.DexWithdrawalColumns)),
			ColumnNames:      indexermodels.GetCrossChainColumnNames(indexermodels.DexWithdrawalColumns),
		},
		{
			TableName:        indexermodels.BlockSummariesProductionTableName,
			PrimaryKey:       []string{"chain_id", "height"},
			HasAddressColumn: false,
			SchemaSQL:        indexermodels.ColumnsToSchemaSQL(indexermodels.FilterCrossChainColumns(indexermodels.BlockSummaryColumns)),
			ColumnNames:      indexermodels.GetCrossChainColumnNames(indexermodels.BlockSummaryColumns),
		},
		{
			TableName:        indexermodels.CommitteePaymentsProductionTableName,
			PrimaryKey:       []string{"chain_id", "committee_id", "address", "height"},
			HasAddressColumn: true,
			SchemaSQL:        indexermodels.ColumnsToSchemaSQL(indexermodels.FilterCrossChainColumns(indexermodels.CommitteePaymentColumns)),
			ColumnNames:      indexermodels.GetCrossChainColumnNames(indexermodels.CommitteePaymentColumns),
		},
		{
			TableName:        indexermodels.EventsProductionTableName,
			PrimaryKey:       []string{"chain_id", "height", "tx_index", "event_index"},
			HasAddressColumn: false,
			SchemaSQL:        indexermodels.ColumnsToSchemaSQL(indexermodels.FilterCrossChainColumns(indexermodels.EventColumns)),
			ColumnNames:      indexermodels.GetCrossChainColumnNames(indexermodels.EventColumns),
		},
		{
			TableName:        indexermodels.TxsProductionTableName,
			PrimaryKey:       []string{"chain_id", "height", "tx_index"},
			HasAddressColumn: false,
			SchemaSQL:        indexermodels.ColumnsToSchemaSQL(indexermodels.FilterCrossChainColumns(indexermodels.TransactionColumns)),
			ColumnNames:      indexermodels.GetCrossChainColumnNames(indexermodels.TransactionColumns),
		},
	}
}
