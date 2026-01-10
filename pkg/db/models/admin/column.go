package admin

import (
	"github.com/canopy-network/canopy-indexer/pkg/db/models/indexer"
)

// ColumnDef is re-exported from the indexer package for convenience.
// This allows admin models to use the same column definition system.
type ColumnDef = indexer.ColumnDef

// Re-export helper functions from the indexer package
var (
	ColumnsToSchemaSQL           = indexer.ColumnsToSchemaSQL
	ColumnsToCrossChainSchemaSQL = indexer.ColumnsToCrossChainSchemaSQL
	ColumnsToNameList            = indexer.ColumnsToNameList
	FilterCrossChainColumns      = indexer.FilterCrossChainColumns
	GetCrossChainColumnNames     = indexer.GetCrossChainColumnNames
	GetCrossChainSelectExprs     = indexer.GetCrossChainSelectExprs
	ValidateColumns              = indexer.ValidateColumns
)
