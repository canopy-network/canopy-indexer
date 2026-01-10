package crosschain

import (
	"context"
	"fmt"
	"strings"

	indexermodels "github.com/canopy-network/canopy-indexer/pkg/db/models/indexer"
)

// buildQueryWithOptions builds a SELECT query with filtering, ordering, and pagination
func buildQueryWithOptions(baseQuery string, opts QueryOptions) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Apply ChainID filter if specified
	if opts.ChainID != nil {
		conditions = append(conditions, fmt.Sprintf("chain_id = $%d", argIndex))
		args = append(args, *opts.ChainID)
		argIndex++
	}

	// Apply additional filters
	for key, value := range opts.Filters {
		conditions = append(conditions, fmt.Sprintf("%s = $%d", key, argIndex))
		args = append(args, value)
		argIndex++
	}

	// Build WHERE clause
	query := baseQuery
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Apply ordering
	orderBy := opts.OrderBy
	if orderBy == "" {
		orderBy = "height"
	}
	orderDir := opts.OrderDir
	if orderDir == "" {
		orderDir = "DESC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)

	// Apply pagination
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	return query, args
}

// buildCountQueryWithOptions builds a COUNT query with the same filters as the main query
func buildCountQueryWithOptions(baseQuery string, opts QueryOptions) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Apply ChainID filter if specified
	if opts.ChainID != nil {
		conditions = append(conditions, fmt.Sprintf("chain_id = $%d", argIndex))
		args = append(args, *opts.ChainID)
		argIndex++
	}

	// Apply additional filters
	for key, value := range opts.Filters {
		conditions = append(conditions, fmt.Sprintf("%s = $%d", key, argIndex))
		args = append(args, value)
		argIndex++
	}

	// Build WHERE clause
	query := "SELECT COUNT(*) FROM (" + baseQuery + ") AS t"
	if len(conditions) > 0 {
		query = "SELECT COUNT(*) FROM (" + baseQuery + " WHERE " + strings.Join(conditions, " AND ") + ") AS t"
	}

	return query, args
}

// QueryLatestAccounts queries the latest account snapshots
func (db *DB) QueryLatestAccounts(ctx context.Context, opts QueryOptions) ([]AccountCrossChain, *QueryMetadata, error) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", db.SchemaTable("accounts_latest"))

	// Get total count
	countQuery, countArgs := buildCountQueryWithOptions(baseQuery, opts)
	var totalCount int64
	row := db.Client.Pool.QueryRow(ctx, countQuery, countArgs...)
	if err := row.Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get results
	query, args := buildQueryWithOptions(baseQuery, opts)
	var accounts []AccountCrossChain
	if err := db.Select(ctx, &accounts, query, args...); err != nil {
		return nil, nil, fmt.Errorf("failed to query accounts: %w", err)
	}

	metadata := &QueryMetadata{
		TotalCount: totalCount,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
		HasMore:    int64(opts.Offset+len(accounts)) < totalCount,
	}

	return accounts, metadata, nil
}

// QueryLatestValidators queries the latest validator snapshots
func (db *DB) QueryLatestValidators(ctx context.Context, opts QueryOptions) ([]ValidatorCrossChain, *QueryMetadata, error) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", db.SchemaTable("validators_latest"))

	// Get total count
	countQuery, countArgs := buildCountQueryWithOptions(baseQuery, opts)
	var totalCount int64
	row := db.Client.Pool.QueryRow(ctx, countQuery, countArgs...)
	if err := row.Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get results
	query, args := buildQueryWithOptions(baseQuery, opts)
	var validators []ValidatorCrossChain
	if err := db.Select(ctx, &validators, query, args...); err != nil {
		return nil, nil, fmt.Errorf("failed to query validators: %w", err)
	}

	metadata := &QueryMetadata{
		TotalCount: totalCount,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
		HasMore:    int64(opts.Offset+len(validators)) < totalCount,
	}

	return validators, metadata, nil
}

// QueryLatestValidatorNonSigningInfo queries the latest validator non-signing info
func (db *DB) QueryLatestValidatorNonSigningInfo(ctx context.Context, opts QueryOptions) ([]ValidatorNonSigningInfoCrossChain, *QueryMetadata, error) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", db.SchemaTable("validator_non_signing_info"))

	// Get total count
	countQuery, countArgs := buildCountQueryWithOptions(baseQuery, opts)
	var totalCount int64
	row := db.Client.Pool.QueryRow(ctx, countQuery, countArgs...)
	if err := row.Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get results
	query, args := buildQueryWithOptions(baseQuery, opts)
	var infos []ValidatorNonSigningInfoCrossChain
	if err := db.Select(ctx, &infos, query, args...); err != nil {
		return nil, nil, fmt.Errorf("failed to query validator non-signing info: %w", err)
	}

	metadata := &QueryMetadata{
		TotalCount: totalCount,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
		HasMore:    int64(opts.Offset+len(infos)) < totalCount,
	}

	return infos, metadata, nil
}

// QueryLatestValidatorDoubleSigningInfo queries the latest validator double-signing info
func (db *DB) QueryLatestValidatorDoubleSigningInfo(ctx context.Context, opts QueryOptions) ([]ValidatorDoubleSigningInfoCrossChain, *QueryMetadata, error) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", db.SchemaTable("validator_double_signing_info"))

	// Get total count
	countQuery, countArgs := buildCountQueryWithOptions(baseQuery, opts)
	var totalCount int64
	row := db.Client.Pool.QueryRow(ctx, countQuery, countArgs...)
	if err := row.Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get results
	query, args := buildQueryWithOptions(baseQuery, opts)
	var infos []ValidatorDoubleSigningInfoCrossChain
	if err := db.Select(ctx, &infos, query, args...); err != nil {
		return nil, nil, fmt.Errorf("failed to query validator double-signing info: %w", err)
	}

	metadata := &QueryMetadata{
		TotalCount: totalCount,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
		HasMore:    int64(opts.Offset+len(infos)) < totalCount,
	}

	return infos, metadata, nil
}

// QueryLatestPools queries the latest pool snapshots
func (db *DB) QueryLatestPools(ctx context.Context, opts QueryOptions) ([]PoolCrossChain, *QueryMetadata, error) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", db.SchemaTable("pools_latest"))

	// Get total count
	countQuery, countArgs := buildCountQueryWithOptions(baseQuery, opts)
	var totalCount int64
	row := db.Client.Pool.QueryRow(ctx, countQuery, countArgs...)
	if err := row.Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get results
	query, args := buildQueryWithOptions(baseQuery, opts)
	var pools []PoolCrossChain
	if err := db.Select(ctx, &pools, query, args...); err != nil {
		return nil, nil, fmt.Errorf("failed to query pools: %w", err)
	}

	metadata := &QueryMetadata{
		TotalCount: totalCount,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
		HasMore:    int64(opts.Offset+len(pools)) < totalCount,
	}

	return pools, metadata, nil
}

// QueryLatestPoolPointsByHolder queries the latest pool points by holder
func (db *DB) QueryLatestPoolPointsByHolder(ctx context.Context, opts QueryOptions) ([]PoolPointsByHolderCrossChain, *QueryMetadata, error) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", db.SchemaTable("pool_points_by_holder_latest"))

	// Get total count
	countQuery, countArgs := buildCountQueryWithOptions(baseQuery, opts)
	var totalCount int64
	row := db.Client.Pool.QueryRow(ctx, countQuery, countArgs...)
	if err := row.Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get results
	query, args := buildQueryWithOptions(baseQuery, opts)
	var points []PoolPointsByHolderCrossChain
	if err := db.Select(ctx, &points, query, args...); err != nil {
		return nil, nil, fmt.Errorf("failed to query pool points by holder: %w", err)
	}

	metadata := &QueryMetadata{
		TotalCount: totalCount,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
		HasMore:    int64(opts.Offset+len(points)) < totalCount,
	}

	return points, metadata, nil
}

// QueryLatestDexOrders queries the latest DEX orders
func (db *DB) QueryLatestDexOrders(ctx context.Context, opts QueryOptions) ([]DexOrderCrossChain, *QueryMetadata, error) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", db.SchemaTable("dex_orders_latest"))

	// Get total count
	countQuery, countArgs := buildCountQueryWithOptions(baseQuery, opts)
	var totalCount int64
	row := db.Client.Pool.QueryRow(ctx, countQuery, countArgs...)
	if err := row.Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get results
	query, args := buildQueryWithOptions(baseQuery, opts)
	var orders []DexOrderCrossChain
	if err := db.Select(ctx, &orders, query, args...); err != nil {
		return nil, nil, fmt.Errorf("failed to query dex orders: %w", err)
	}

	metadata := &QueryMetadata{
		TotalCount: totalCount,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
		HasMore:    int64(opts.Offset+len(orders)) < totalCount,
	}

	return orders, metadata, nil
}

// QueryLatestDexDeposits queries the latest DEX deposits
func (db *DB) QueryLatestDexDeposits(ctx context.Context, opts QueryOptions) ([]DexDepositCrossChain, *QueryMetadata, error) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", db.SchemaTable("dex_deposits_latest"))

	// Get total count
	countQuery, countArgs := buildCountQueryWithOptions(baseQuery, opts)
	var totalCount int64
	row := db.Client.Pool.QueryRow(ctx, countQuery, countArgs...)
	if err := row.Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get results
	query, args := buildQueryWithOptions(baseQuery, opts)
	var deposits []DexDepositCrossChain
	if err := db.Select(ctx, &deposits, query, args...); err != nil {
		return nil, nil, fmt.Errorf("failed to query dex deposits: %w", err)
	}

	metadata := &QueryMetadata{
		TotalCount: totalCount,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
		HasMore:    int64(opts.Offset+len(deposits)) < totalCount,
	}

	return deposits, metadata, nil
}

// QueryLatestDexWithdrawals queries the latest DEX withdrawals
func (db *DB) QueryLatestDexWithdrawals(ctx context.Context, opts QueryOptions) ([]DexWithdrawalCrossChain, *QueryMetadata, error) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", db.SchemaTable("dex_withdrawals_latest"))

	// Get total count
	countQuery, countArgs := buildCountQueryWithOptions(baseQuery, opts)
	var totalCount int64
	row := db.Client.Pool.QueryRow(ctx, countQuery, countArgs...)
	if err := row.Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get results
	query, args := buildQueryWithOptions(baseQuery, opts)
	var withdrawals []DexWithdrawalCrossChain
	if err := db.Select(ctx, &withdrawals, query, args...); err != nil {
		return nil, nil, fmt.Errorf("failed to query dex withdrawals: %w", err)
	}

	metadata := &QueryMetadata{
		TotalCount: totalCount,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
		HasMore:    int64(opts.Offset+len(withdrawals)) < totalCount,
	}

	return withdrawals, metadata, nil
}

// QueryLatestBlockSummaries queries the latest block summaries
func (db *DB) QueryLatestBlockSummaries(ctx context.Context, opts QueryOptions) ([]BlockSummaryCrossChain, *QueryMetadata, error) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", db.SchemaTable("block_summaries"))

	// Get total count
	countQuery, countArgs := buildCountQueryWithOptions(baseQuery, opts)
	var totalCount int64
	row := db.Client.Pool.QueryRow(ctx, countQuery, countArgs...)
	if err := row.Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get results
	query, args := buildQueryWithOptions(baseQuery, opts)
	var summaries []BlockSummaryCrossChain
	if err := db.Select(ctx, &summaries, query, args...); err != nil {
		return nil, nil, fmt.Errorf("failed to query block summaries: %w", err)
	}

	metadata := &QueryMetadata{
		TotalCount: totalCount,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
		HasMore:    int64(opts.Offset+len(summaries)) < totalCount,
	}

	return summaries, metadata, nil
}

// QueryLatestCommitteePayments queries the latest committee payments
func (db *DB) QueryLatestCommitteePayments(ctx context.Context, opts QueryOptions) ([]CommitteePaymentCrossChain, *QueryMetadata, error) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", db.SchemaTable("committee_payments"))

	// Get total count
	countQuery, countArgs := buildCountQueryWithOptions(baseQuery, opts)
	var totalCount int64
	row := db.Client.Pool.QueryRow(ctx, countQuery, countArgs...)
	if err := row.Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get results
	query, args := buildQueryWithOptions(baseQuery, opts)
	var payments []CommitteePaymentCrossChain
	if err := db.Select(ctx, &payments, query, args...); err != nil {
		return nil, nil, fmt.Errorf("failed to query committee payments: %w", err)
	}

	metadata := &QueryMetadata{
		TotalCount: totalCount,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
		HasMore:    int64(opts.Offset+len(payments)) < totalCount,
	}

	return payments, metadata, nil
}

// QueryLatestEvents queries the latest events
func (db *DB) QueryLatestEvents(ctx context.Context, opts QueryOptions) ([]EventCrossChain, *QueryMetadata, error) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", db.SchemaTable("events"))

	// Get total count
	countQuery, countArgs := buildCountQueryWithOptions(baseQuery, opts)
	var totalCount int64
	row := db.Client.Pool.QueryRow(ctx, countQuery, countArgs...)
	if err := row.Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get results
	query, args := buildQueryWithOptions(baseQuery, opts)
	var events []EventCrossChain
	if err := db.Select(ctx, &events, query, args...); err != nil {
		return nil, nil, fmt.Errorf("failed to query events: %w", err)
	}

	metadata := &QueryMetadata{
		TotalCount: totalCount,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
		HasMore:    int64(opts.Offset+len(events)) < totalCount,
	}

	return events, metadata, nil
}

// QueryLatestTransactions queries the latest transactions
func (db *DB) QueryLatestTransactions(ctx context.Context, opts QueryOptions) ([]TransactionCrossChain, *QueryMetadata, error) {
	baseQuery := fmt.Sprintf("SELECT * FROM %s", db.SchemaTable("txs"))

	// Get total count
	countQuery, countArgs := buildCountQueryWithOptions(baseQuery, opts)
	var totalCount int64
	row := db.Client.Pool.QueryRow(ctx, countQuery, countArgs...)
	if err := row.Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get results
	query, args := buildQueryWithOptions(baseQuery, opts)
	var txs []TransactionCrossChain
	if err := db.Select(ctx, &txs, query, args...); err != nil {
		return nil, nil, fmt.Errorf("failed to query transactions: %w", err)
	}

	metadata := &QueryMetadata{
		TotalCount: totalCount,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
		HasMore:    int64(opts.Offset+len(txs)) < totalCount,
	}

	return txs, metadata, nil
}

// QueryLatestEntity queries the latest entity data (generic method)
func (db *DB) QueryLatestEntity(ctx context.Context, entityName string, opts QueryOptions) ([]map[string]interface{}, *QueryMetadata, error) {
	// Validate entity name (prevent SQL injection)
	allowedEntities := map[string]bool{
		"accounts":                      true,
		"validators":                    true,
		"validator_non_signing_info":    true,
		"validator_double_signing_info": true,
		"pools":                         true,
		"pool_points_by_holder":         true,
		"orders":                        true,
		"dex_orders":                    true,
		"dex_deposits":                  true,
		"dex_withdrawals":               true,
		"block_summaries":               true,
		"committee_payments":            true,
		"events":                        true,
		"txs":                           true,
	}

	if !allowedEntities[entityName] {
		return nil, nil, fmt.Errorf("invalid entity name: %s", entityName)
	}

	// Determine if this entity has a _latest view
	hasLatestView := map[string]bool{
		"accounts":              true,
		"validators":            true,
		"pools":                 true,
		"pool_points_by_holder": true,
		"orders":                true,
		"dex_orders":            true,
		"dex_deposits":          true,
		"dex_withdrawals":       true,
	}

	tableName := entityName
	if hasLatestView[entityName] {
		tableName = entityName + "_latest"
	}

	baseQuery := fmt.Sprintf("SELECT * FROM %s", db.SchemaTable(tableName))

	// Get total count
	countQuery, countArgs := buildCountQueryWithOptions(baseQuery, opts)
	var totalCount int64
	row := db.Client.Pool.QueryRow(ctx, countQuery, countArgs...)
	if err := row.Scan(&totalCount); err != nil {
		return nil, nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get results
	query, args := buildQueryWithOptions(baseQuery, opts)
	rows, err := db.Client.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query %s: %w", entityName, err)
	}
	defer rows.Close()

	// Scan into maps
	var results []map[string]interface{}
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan row: %w", err)
		}

		fieldDescriptions := rows.FieldDescriptions()
		row := make(map[string]interface{})
		for i, field := range fieldDescriptions {
			row[string(field.Name)] = values[i]
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("error iterating rows: %w", err)
	}

	metadata := &QueryMetadata{
		TotalCount: totalCount,
		Limit:      opts.Limit,
		Offset:     opts.Offset,
		HasMore:    int64(opts.Offset+len(results)) < totalCount,
	}

	return results, metadata, nil
}

// QueryLPPositionSnapshots queries LP position snapshots
func (db *DB) QueryLPPositionSnapshots(ctx context.Context, params LPPositionSnapshotQueryParams) ([]*indexermodels.LPPositionSnapshot, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if params.SourceChainID != nil {
		conditions = append(conditions, fmt.Sprintf("source_chain_id = $%d", argIndex))
		args = append(args, *params.SourceChainID)
		argIndex++
	}

	if params.Address != nil {
		conditions = append(conditions, fmt.Sprintf("address = $%d", argIndex))
		args = append(args, *params.Address)
		argIndex++
	}

	if params.PoolID != nil {
		conditions = append(conditions, fmt.Sprintf("pool_id = $%d", argIndex))
		args = append(args, *params.PoolID)
		argIndex++
	}

	query := fmt.Sprintf("SELECT * FROM %s", db.SchemaTable("lp_position_snapshots"))
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY snapshot_time DESC"

	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	if params.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", params.Offset)
	}

	var snapshots []*indexermodels.LPPositionSnapshot
	if err := db.Select(ctx, &snapshots, query, args...); err != nil {
		return nil, fmt.Errorf("failed to query LP position snapshots: %w", err)
	}

	return snapshots, nil
}

// GetLatestLPPositionSnapshot gets the latest LP position snapshot for a specific account/pool
func (db *DB) GetLatestLPPositionSnapshot(ctx context.Context, sourceChainID uint64, address string, poolID uint64) (*indexermodels.LPPositionSnapshot, error) {
	query := fmt.Sprintf(`
		SELECT * FROM %s
		WHERE source_chain_id = $1 AND address = $2 AND pool_id = $3
		ORDER BY snapshot_time DESC
		LIMIT 1
	`, db.SchemaTable("lp_position_snapshots"))

	var snapshot indexermodels.LPPositionSnapshot
	rows, err := db.Client.Pool.Query(ctx, query, sourceChainID, address, poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest LP position snapshot: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil // Not found
	}

	if err := rows.Scan(
		&snapshot.SourceChainID,
		&snapshot.Address,
		&snapshot.PoolID,
		&snapshot.SnapshotDate,
		&snapshot.SnapshotHeight,
		&snapshot.SnapshotBalance,
		&snapshot.PoolSharePercentage,
		&snapshot.PositionCreatedDate,
		&snapshot.PositionClosedDate,
		&snapshot.IsPositionActive,
		&snapshot.ComputedAt,
		&snapshot.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("failed to scan LP position snapshot: %w", err)
	}

	return &snapshot, nil
}

// InsertLPPositionSnapshots inserts LP position snapshots
func (db *DB) InsertLPPositionSnapshots(ctx context.Context, snapshots []*indexermodels.LPPositionSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (
			source_chain_id, address, pool_id, snapshot_date,
			snapshot_height, snapshot_balance, pool_share_percentage,
			position_created_date, position_closed_date, is_position_active,
			computed_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (source_chain_id, address, pool_id, snapshot_date) DO UPDATE SET
			snapshot_height = EXCLUDED.snapshot_height,
			snapshot_balance = EXCLUDED.snapshot_balance,
			pool_share_percentage = EXCLUDED.pool_share_percentage,
			position_created_date = EXCLUDED.position_created_date,
			position_closed_date = EXCLUDED.position_closed_date,
			is_position_active = EXCLUDED.is_position_active,
			computed_at = EXCLUDED.computed_at,
			updated_at = EXCLUDED.updated_at
	`, db.SchemaTable("lp_position_snapshots"))

	for _, snap := range snapshots {
		if err := db.Exec(ctx, query,
			snap.SourceChainID,
			snap.Address,
			snap.PoolID,
			snap.SnapshotDate,
			snap.SnapshotHeight,
			snap.SnapshotBalance,
			snap.PoolSharePercentage,
			snap.PositionCreatedDate,
			snap.PositionClosedDate,
			snap.IsPositionActive,
			snap.ComputedAt,
			snap.UpdatedAt,
		); err != nil {
			return fmt.Errorf("failed to insert LP position snapshot: %w", err)
		}
	}

	return nil
}
