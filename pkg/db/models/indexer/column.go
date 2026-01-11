package indexer

import (
	"fmt"
)

// ColumnDef defines a single column for a table.
// This is the single source of truth for column definitions, used by:
// - Cross-chain global tables sync (pkg/db/crosschain/store.go)
type ColumnDef struct {
	// Name is the column name in the source table
	Name string

	// CrossChainSkip excludes this column from cross-chain global tables.
	// Use this for columns that are too granular or not needed for cross-chain queries.
	// Example: Detailed event breakdowns in block_summaries
	CrossChainSkip bool

	// CrossChainRename renames this column in cross-chain global tables.
	// Use this to avoid naming conflicts (e.g., pools.chain_id → pool_chain_id)
	// Mutually exclusive with CrossChainSkip
	CrossChainRename string
}

// GetCrossChainName returns the column name to use in cross-chain global tables.
// Returns empty string if column should be skipped.
func (c ColumnDef) GetCrossChainName() string {
	if c.CrossChainSkip {
		return ""
	}
	if c.CrossChainRename != "" {
		return c.CrossChainRename
	}
	return c.Name
}

// GetSelectExpr returns the SELECT expression for materialized views.
// Returns empty string if the column should be skipped.
// Examples:
//   - Normal column: "address"
//   - Renamed column: "chain_id AS pool_chain_id"
//   - Skipped column: ""
func (c ColumnDef) GetSelectExpr() string {
	if c.CrossChainSkip {
		return ""
	}
	if c.CrossChainRename != "" {
		return fmt.Sprintf("%s AS %s", c.Name, c.CrossChainRename)
	}
	return c.Name
}

// ShouldSyncToCrossChain returns whether this column should be included in cross-chain tables.
func (c ColumnDef) ShouldSyncToCrossChain() bool {
	return !c.CrossChainSkip
}

// Validate checks if the column definition is valid.
// Returns error if CrossChainSkip and CrossChainRename are both set (mutually exclusive).
func (c ColumnDef) Validate() error {
	if c.CrossChainSkip && c.CrossChainRename != "" {
		return fmt.Errorf("column %s: CrossChainSkip and CrossChainRename are mutually exclusive", c.Name)
	}
	if c.Name == "" {
		return fmt.Errorf("column name cannot be empty")
	}
	return nil
}

// ColumnsToNameList extracts just the column names from a list of ColumnDef.
// Useful for INSERT statements.
func ColumnsToNameList(columns []ColumnDef) []string {
	var names []string
	for _, col := range columns {
		names = append(names, col.Name)
	}
	return names
}

// FilterCrossChainColumns returns only columns that should be synced to cross-chain.
// Excludes columns with CrossChainSkip=true.
func FilterCrossChainColumns(columns []ColumnDef) []ColumnDef {
	var result []ColumnDef
	for _, col := range columns {
		if col.ShouldSyncToCrossChain() {
			result = append(result, col)
		}
	}
	return result
}

// GetCrossChainColumnNames returns the column names as they appear in cross-chain tables.
// Handles CrossChainRename and excludes CrossChainSkip columns.
func GetCrossChainColumnNames(columns []ColumnDef) []string {
	var names []string
	for _, col := range columns {
		if name := col.GetCrossChainName(); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// GetCrossChainSelectExprs returns SELECT expressions for materialized views.
// Handles CrossChainRename (with AS) and excludes CrossChainSkip columns.
func GetCrossChainSelectExprs(columns []ColumnDef) []string {
	var exprs []string
	for _, col := range columns {
		if expr := col.GetSelectExpr(); expr != "" {
			exprs = append(exprs, expr)
		}
	}
	return exprs
}

// ValidateColumns validates all columns in a list.
// Returns the first validation error encountered.
func ValidateColumns(columns []ColumnDef) error {
	for _, col := range columns {
		if err := col.Validate(); err != nil {
			return err
		}
	}
	return nil
}
