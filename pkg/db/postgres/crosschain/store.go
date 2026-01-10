package crosschain

import (
	"context"
	"errors"
)

var ErrNotImplemented = errors.New("method not implemented for Postgres crosschain store")

// Ensure DB implements the required interface methods
var _ interface {
	DatabaseName() string
	GetConnection() interface{}
	InitializeDB(ctx context.Context) error
	Close() error
} = (*DB)(nil)

// Exec executes a query without returning rows
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) error {
	return db.Client.Exec(ctx, query, args...)
}

// QueryRow executes a query that returns a single row
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) interface{} {
	return db.Client.Pool.QueryRow(ctx, query, args...)
}

// Query executes a query that returns multiple rows
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (interface{}, error) {
	return db.Client.Pool.Query(ctx, query, args...)
}

// Select executes a query and scans results into dest
func (db *DB) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return db.Client.Select(ctx, dest, query, args...)
}

// =============================================================================
// Sync Methods (Implemented in sync.go)
// =============================================================================

// NOTE: All sync method implementations are in sync.go
// - SetupChainSync: Creates triggers for real-time sync
// - ResyncTable: Backfills existing data
// - ResyncChain: Resyncs all tables
// - GetSyncStatus: Monitors sync health
// - RemoveChainSync: Cleanup triggers and data

// =============================================================================
// Health Methods (Stubs - to be implemented)
// =============================================================================

// HealthStatus represents the health status of the database
type HealthStatus struct {
	Healthy bool
	Message string
}

func (db *DB) GetHealthStatus(ctx context.Context) (*HealthStatus, error) {
	return &HealthStatus{
		Healthy: true,
		Message: "Postgres crosschain database operational",
	}, nil
}

// =============================================================================
// Maintenance Methods (Stubs - to be implemented)
// =============================================================================

func (db *DB) OptimizeTables(ctx context.Context) error {
	// Postgres handles optimization automatically via VACUUM
	return nil
}

func (db *DB) OptimizeTable(ctx context.Context, database, table string, final bool) error {
	// Postgres handles optimization automatically via VACUUM
	return nil
}

// =============================================================================
// Query Options and Metadata
// =============================================================================

// QueryOptions represents options for querying entities
type QueryOptions struct {
	ChainID  *uint64
	Limit    int
	Offset   int
	OrderBy  string
	OrderDir string
	Filters  map[string]interface{}
}

// QueryMetadata represents metadata about query results
type QueryMetadata struct {
	TotalCount int64
	Limit      int
	Offset     int
	HasMore    bool
}

// =============================================================================
// CrossChain Entity Types
// =============================================================================

type AccountCrossChain struct {
	ChainID     uint64 `db:"chain_id"`
	Address     string `db:"address"`
	Amount      int64  `db:"amount"`
	Rewards     int64  `db:"rewards"`
	Slashes     int64  `db:"slashes"`
	Height      uint64 `db:"height"`
	HeightTime  string `db:"height_time"`
}

type ValidatorCrossChain struct {
	ChainID         uint64 `db:"chain_id"`
	Address         string `db:"address"`
	PublicKey       string `db:"public_key"`
	NetAddress      string `db:"net_address"`
	StakedAmount    int64  `db:"staked_amount"`
	MaxPausedHeight uint64 `db:"max_paused_height"`
	UnstakingHeight uint64 `db:"unstaking_height"`
	Output          string `db:"output"`
	Delegate        bool   `db:"delegate"`
	Compound        bool   `db:"compound"`
	Status          string `db:"status"`
	Height          uint64 `db:"height"`
	HeightTime      string `db:"height_time"`
}

type ValidatorNonSigningInfoCrossChain struct {
	ChainID           uint64 `db:"chain_id"`
	Address           string `db:"address"`
	MissedBlocksCount int64  `db:"missed_blocks_count"`
	LastSignedHeight  uint64 `db:"last_signed_height"`
	Height            uint64 `db:"height"`
	HeightTime        string `db:"height_time"`
}

type ValidatorDoubleSigningInfoCrossChain struct {
	ChainID             uint64 `db:"chain_id"`
	Address             string `db:"address"`
	EvidenceCount       int64  `db:"evidence_count"`
	FirstEvidenceHeight uint64 `db:"first_evidence_height"`
	LastEvidenceHeight  uint64 `db:"last_evidence_height"`
	Height              uint64 `db:"height"`
	HeightTime          string `db:"height_time"`
}

type PoolCrossChain struct {
	ChainID          uint64 `db:"chain_id"`
	PoolID           uint64 `db:"pool_id"`
	PoolChainID      uint64 `db:"pool_chain_id"`
	Amount           int64  `db:"amount"`
	TotalPoints      int64  `db:"total_points"`
	LpCount          int    `db:"lp_count"`
	Height           uint64 `db:"height"`
	HeightTime       string `db:"height_time"`
	LiquidityPoolID  uint64 `db:"liquidity_pool_id"`
	HoldingPoolID    uint64 `db:"holding_pool_id"`
	EscrowPoolID     uint64 `db:"escrow_pool_id"`
	RewardPoolID     uint64 `db:"reward_pool_id"`
	AmountDelta      int64  `db:"amount_delta"`
	TotalPointsDelta int64  `db:"total_points_delta"`
	LpCountDelta     int    `db:"lp_count_delta"`
}

type PoolPointsByHolderCrossChain struct {
	ChainID             uint64 `db:"chain_id"`
	Address             string `db:"address"`
	PoolID              uint64 `db:"pool_id"`
	Committee           uint64 `db:"committee"`
	Points              int64  `db:"points"`
	LiquidityPoolPoints int64  `db:"liquidity_pool_points"`
	LiquidityPoolID     uint64 `db:"liquidity_pool_id"`
	Height              uint64 `db:"height"`
	HeightTime          string `db:"height_time"`
}

type DexOrderCrossChain struct {
	ChainID         uint64 `db:"chain_id"`
	OrderID         string `db:"order_id"`
	Committee       uint64 `db:"committee"`
	Address         string `db:"address"`
	AmountForSale   int64  `db:"amount_for_sale"`
	RequestedAmount int64  `db:"requested_amount"`
	State           string `db:"state"`
	Success         bool   `db:"success"`
	SoldAmount      int64  `db:"sold_amount"`
	BoughtAmount    int64  `db:"bought_amount"`
	LocalOrigin     bool   `db:"local_origin"`
	LockedHeight    uint64 `db:"locked_height"`
	Height          uint64 `db:"height"`
	HeightTime      string `db:"height_time"`
}

type DexDepositCrossChain struct {
	ChainID        uint64 `db:"chain_id"`
	OrderID        string `db:"order_id"`
	Committee      uint64 `db:"committee"`
	Address        string `db:"address"`
	Amount         int64  `db:"amount"`
	State          string `db:"state"`
	LocalOrigin    bool   `db:"local_origin"`
	PointsReceived int64  `db:"points_received"`
	Height         uint64 `db:"height"`
	HeightTime     string `db:"height_time"`
}

type DexWithdrawalCrossChain struct {
	ChainID      uint64 `db:"chain_id"`
	OrderID      string `db:"order_id"`
	Committee    uint64 `db:"committee"`
	Address      string `db:"address"`
	Percent      int64  `db:"percent"`
	State        string `db:"state"`
	LocalAmount  int64  `db:"local_amount"`
	RemoteAmount int64  `db:"remote_amount"`
	PointsBurned int64  `db:"points_burned"`
	Height       uint64 `db:"height"`
	HeightTime   string `db:"height_time"`
}

type BlockSummaryCrossChain struct {
	ChainID            uint64 `db:"chain_id"`
	Height             uint64 `db:"height"`
	HeightTime         string `db:"height_time"`
	TotalTransactions  int64  `db:"total_transactions"`
	NumTxs             int    `db:"num_txs"`
	NumAccounts        int    `db:"num_accounts"`
	NumEvents          int    `db:"num_events"`
	NumValidators      int    `db:"num_validators"`
	NumCommittees      int    `db:"num_committees"`
	NumPools           int    `db:"num_pools"`
	NumOrders          int    `db:"num_orders"`
	NumDexOrders       int    `db:"num_dex_orders"`
	NumDexDeposits     int    `db:"num_dex_deposits"`
	NumDexWithdrawals  int    `db:"num_dex_withdrawals"`
	SupplyTotal        int64  `db:"supply_total"`
	SupplyStaked       int64  `db:"supply_staked"`
	SupplyDelegatedOnly int64 `db:"supply_delegated_only"`
}

type CommitteePaymentCrossChain struct {
	ChainID     uint64 `db:"chain_id"`
	CommitteeID uint64 `db:"committee_id"`
	Address     string `db:"address"`
	Percent     int64  `db:"percent"`
	Height      uint64 `db:"height"`
	HeightTime  string `db:"height_time"`
}

type EventCrossChain struct {
	ChainID              uint64  `db:"chain_id"`
	Height               uint64  `db:"height"`
	HeightTime           string  `db:"height_time"`
	EventChainID         uint64  `db:"event_chain_id"`
	Address              string  `db:"address"`
	Reference            string  `db:"reference"`
	EventType            string  `db:"event_type"`
	BlockHeight          uint64  `db:"block_height"`
	Amount               *int64  `db:"amount"`
	SoldAmount           *int64  `db:"sold_amount"`
	BoughtAmount         *int64  `db:"bought_amount"`
	LocalAmount          *int64  `db:"local_amount"`
	RemoteAmount         *int64  `db:"remote_amount"`
	Success              *bool   `db:"success"`
	LocalOrigin          *bool   `db:"local_origin"`
	OrderID              *string `db:"order_id"`
	PointsReceived       *int64  `db:"points_received"`
	PointsBurned         *int64  `db:"points_burned"`
	Data                 *string `db:"data"`
	SellerReceiveAddress *string `db:"seller_receive_address"`
	BuyerSendAddress     *string `db:"buyer_send_address"`
	SellersSendAddress   *string `db:"sellers_send_address"`
}

type TransactionCrossChain struct {
	ChainID             uint64   `db:"chain_id"`
	Height              uint64   `db:"height"`
	HeightTime          string   `db:"height_time"`
	TxHash              string   `db:"tx_hash"`
	TxIndex             int      `db:"tx_index"`
	TxTime              string   `db:"tx_time"`
	CreatedHeight       uint64   `db:"created_height"`
	NetworkID           uint64   `db:"network_id"`
	MessageType         string   `db:"message_type"`
	Signer              string   `db:"signer"`
	Amount              *int64   `db:"amount"`
	Fee                 int64    `db:"fee"`
	Memo                *string  `db:"memo"`
	ValidatorAddress    *string  `db:"validator_address"`
	Commission          *float64 `db:"commission"`
	TxChainID           *uint64  `db:"tx_chain_id"`
	SellAmount          *int64   `db:"sell_amount"`
	BuyAmount           *int64   `db:"buy_amount"`
	LiquidityAmount     *int64   `db:"liquidity_amount"`
	LiquidityPercent    *int64   `db:"liquidity_percent"`
	OrderID             *string  `db:"order_id"`
	Price               *float64 `db:"price"`
	ParamKey            *string  `db:"param_key"`
	ParamValue          *string  `db:"param_value"`
	CommitteeID         *uint64  `db:"committee_id"`
	Recipient           *string  `db:"recipient"`
	PollHash            *string  `db:"poll_hash"`
	BuyerReceiveAddress *string  `db:"buyer_receive_address"`
	BuyerSendAddress    *string  `db:"buyer_send_address"`
	BuyerChainDeadline  *uint64  `db:"buyer_chain_deadline"`
	PublicKey           *string  `db:"public_key"`
	Signature           *string  `db:"signature"`
}

// =============================================================================
// LP Position Snapshot Query Parameters
// =============================================================================

// LPPositionSnapshotQueryParams represents parameters for LP position snapshot queries
type LPPositionSnapshotQueryParams struct {
	SourceChainID *uint64
	Address       *string
	PoolID        *uint64
	Limit         int
	Offset        int
}

// NOTE: All query method implementations are in queries.go
