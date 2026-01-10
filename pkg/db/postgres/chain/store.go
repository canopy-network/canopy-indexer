package chain

import (
	"context"
	"errors"

	dbtypes "github.com/canopy-network/canopy-indexer/pkg/db"
	"github.com/canopy-network/canopy-indexer/pkg/db/entities"
	indexermodels "github.com/canopy-network/canopy-indexer/pkg/db/models/indexer"
)

var ErrNotImplemented = errors.New("method not implemented for Postgres chain store")

// NOTE: Postgres chain store provides equivalent functionality to chain.Store but does not
// formally implement the interface due to GetConnection() returning *pgxpool.Pool instead of
// driver.Conn (ClickHouse-specific type). All other methods match the chain.Store interface.
//
// Method coverage:
// - ✅ All Insert*Staging methods (wire to direct inserts, no staging needed)
// - ✅ All query methods (GetBlock, GetEventsByTypeAndHeight, etc.)
// - ✅ Delete methods (DeleteBlock, DeleteTransactions)
// - ✅ Staging cleanup methods (implemented as no-ops)
// - ⚠️ GetConnection() returns *pgxpool.Pool (not driver.Conn)
// - ⚠️ DescribeTable, GetTableSchema, GetTableDataPaginated (not implemented)

// Exec executes a query without returning rows
func (db *DB) Exec(ctx context.Context, query string, args ...any) error {
	return db.Client.Exec(ctx, query, args...)
}

// Select executes a query and scans results into dest
func (db *DB) Select(ctx context.Context, dest interface{}, query string, args ...any) error {
	return db.Client.Select(ctx, dest, query, args...)
}

// DescribeTable returns column information for a table
func (db *DB) DescribeTable(ctx context.Context, tableName string) ([]dbtypes.Column, error) {
	return nil, ErrNotImplemented
}

// GetTableSchema returns the schema for a table
func (db *DB) GetTableSchema(ctx context.Context, tableName string) ([]dbtypes.Column, error) {
	return nil, ErrNotImplemented
}

// GetTableDataPaginated returns paginated table data
func (db *DB) GetTableDataPaginated(ctx context.Context, tableName string, limit, offset int, fromHeight, toHeight *uint64) ([]map[string]interface{}, int64, bool, error) {
	return nil, 0, false, ErrNotImplemented
}

// =============================================================================
// Staging Methods (No-ops for Postgres - ACID transactions replace staging)
// =============================================================================

func (db *DB) InsertAccounts(ctx context.Context, accounts []*indexermodels.Account) error {
	// Postgres has ACID transactions - no staging needed
	// Direct insert with ON CONFLICT DO UPDATE for idempotency
	return db.insertAccounts(ctx, db.Client.GetExecutor(ctx), accounts)
}

func (db *DB) InsertBlocks(ctx context.Context, block *indexermodels.Block) error {
	return db.insertBlock(ctx, db.Client.GetExecutor(ctx), block)
}

func (db *DB) InsertTransactions(ctx context.Context, txs []*indexermodels.Transaction) error {
	return db.insertTransactions(ctx, db.Client.GetExecutor(ctx), txs)
}

func (db *DB) InsertBlockSummaries(ctx context.Context, summary *indexermodels.BlockSummary) error {
	return db.insertBlockSummary(ctx, db.Client.GetExecutor(ctx), summary)
}

func (db *DB) InsertEvents(ctx context.Context, events []*indexermodels.Event) error {
	return db.insertEvents(ctx, db.Client.GetExecutor(ctx), events)
}

func (db *DB) InsertDexPrices(ctx context.Context, prices []*indexermodels.DexPrice) error {
	return db.insertDexPrices(ctx, db.Client.GetExecutor(ctx), prices)
}

func (db *DB) InsertPools(ctx context.Context, pools []*indexermodels.Pool) error {
	return db.insertPools(ctx, db.Client.GetExecutor(ctx), pools)
}

func (db *DB) InsertOrders(ctx context.Context, orders []*indexermodels.Order) error {
	return db.insertOrders(ctx, db.Client.GetExecutor(ctx), orders)
}

func (db *DB) InsertDexOrders(ctx context.Context, orders []*indexermodels.DexOrder) error {
	return db.insertDexOrders(ctx, db.Client.GetExecutor(ctx), orders)
}

func (db *DB) InsertDexDeposits(ctx context.Context, deposits []*indexermodels.DexDeposit) error {
	return db.insertDexDeposits(ctx, db.Client.GetExecutor(ctx), deposits)
}

func (db *DB) InsertDexWithdrawals(ctx context.Context, withdrawals []*indexermodels.DexWithdrawal) error {
	return db.insertDexWithdrawals(ctx, db.Client.GetExecutor(ctx), withdrawals)
}

func (db *DB) InsertPoolPointsByHolder(ctx context.Context, holders []*indexermodels.PoolPointsByHolder) error {
	return db.insertPoolPointsByHolder(ctx, db.Client.GetExecutor(ctx), holders)
}

func (db *DB) InsertParams(ctx context.Context, params *indexermodels.Params) error {
	return db.insertParams(ctx, db.Client.GetExecutor(ctx), params)
}

func (db *DB) InsertValidators(ctx context.Context, validators []*indexermodels.Validator) error {
	return db.insertValidators(ctx, db.Client.GetExecutor(ctx), validators)
}

func (db *DB) InsertValidatorNonSigningInfo(ctx context.Context, nonSigningInfos []*indexermodels.ValidatorNonSigningInfo) error {
	return db.insertValidatorNonSigningInfo(ctx, db.Client.GetExecutor(ctx), nonSigningInfos)
}

func (db *DB) InsertValidatorDoubleSigningInfo(ctx context.Context, doubleSigningInfos []*indexermodels.ValidatorDoubleSigningInfo) error {
	return db.insertValidatorDoubleSigningInfo(ctx, db.Client.GetExecutor(ctx), doubleSigningInfos)
}

func (db *DB) InsertCommittees(ctx context.Context, committees []*indexermodels.Committee) error {
	return db.insertCommittees(ctx, db.Client.GetExecutor(ctx), committees)
}

func (db *DB) InsertCommitteeValidators(ctx context.Context, cvs []*indexermodels.CommitteeValidator) error {
	return db.insertCommitteeValidators(ctx, db.Client.GetExecutor(ctx), cvs)
}

func (db *DB) InsertCommitteePayments(ctx context.Context, payments []*indexermodels.CommitteePayment) error {
	return db.insertCommitteePayments(ctx, db.Client.GetExecutor(ctx), payments)
}

func (db *DB) InsertPollSnapshots(ctx context.Context, snapshots []*indexermodels.PollSnapshot) error {
	return db.insertPollSnapshots(ctx, db.Client.GetExecutor(ctx), snapshots)
}

func (db *DB) InsertProposalSnapshots(ctx context.Context, snapshots []*indexermodels.ProposalSnapshot) error {
	return db.insertProposalSnapshots(ctx, db.Client.GetExecutor(ctx), snapshots)
}

func (db *DB) InsertSupply(ctx context.Context, supplies []*indexermodels.Supply) error {
	if len(supplies) == 0 {
		return nil
	}
	// InsertSupplyStaging receives a slice but insertSupply expects a single item
	// Insert each supply individually
	exec := db.Client.GetExecutor(ctx)
	for _, supply := range supplies {
		if err := db.insertSupply(ctx, exec, supply); err != nil {
			return err
		}
	}
	return nil
}

// PromoteEntity is a no-op for Postgres (staging pattern not used)
func (db *DB) PromoteEntity(ctx context.Context, entity entities.Entity, height uint64) error {
	// No staging in Postgres - writes go directly to main tables
	return nil
}

// CleanEntityStaging is a no-op for Postgres (staging pattern not used)
func (db *DB) CleanEntityStaging(ctx context.Context, entity entities.Entity, height uint64) error {
	// No staging in Postgres
	return nil
}

// CleanAllEntitiesStaging is a no-op for Postgres (staging pattern not used)
func (db *DB) CleanAllEntitiesStaging(ctx context.Context, entitiesToClean []entities.Entity, height uint64) error {
	// No staging in Postgres
	return nil
}

// CleanStagingBatch is a no-op for Postgres (staging pattern not used)
func (db *DB) CleanStagingBatch(ctx context.Context, heights []uint64) (int, error) {
	// No staging in Postgres
	return 0, nil
}

// EventRewardSlash represents reward/slash event data
type EventRewardSlash struct {
	Address string
	Amount  int64
	Type    string
}

// EventLifecycle represents validator lifecycle event data
type EventLifecycle struct {
	Address   string
	EventType string
}

// EventWithOrderID represents an event with an order ID
type EventWithOrderID struct {
	OrderID   string
	EventType string
}

// EventDexBatch represents DEX batch event data
type EventDexBatch struct {
	OrderID   string
	EventType string
	Amount    int64
}

// =============================================================================
// Delete Methods (Wire to private delete methods in deletes.go)
// =============================================================================

// DeleteBlock deletes a block and all associated data at the given height
func (db *DB) DeleteBlock(ctx context.Context, height uint64) error {
	return db.DeleteAllAtHeight(ctx, height)
}

// DeleteTransactions deletes transactions at the given height
func (db *DB) DeleteTransactions(ctx context.Context, height uint64) error {
	return db.deleteTransactionsAtHeight(ctx, db.Client.GetExecutor(ctx), height)
}

// Note: Query methods are implemented in queries.go
// Note: Insert methods (private) are implemented in inserts.go
// Note: Delete methods (private) are implemented in deletes.go
