package transform

import (
	"time"

	"github.com/canopy-network/canopy/fsm"
)

// Pool type constants from Canopy (fsm/key.go).
const (
	RewardPoolAddend    = 0
	HoldingPoolAddend   = 16384
	LiquidityPoolAddend = 32768
	EscrowPoolAddend    = 65536
)

// Pool holds the transformed pool data for database insertion.
type Pool struct {
	PoolID      uint64
	ChainID     uint64
	Amount      uint64
	TotalPoints uint64
	LPCount     uint16
	Height      uint64
	HeightTime  time.Time
}

// PoolFromFSM converts a fsm.Pool to the database Pool model.
// Height and HeightTime must be set by the caller.
func PoolFromFSM(p *fsm.Pool) *Pool {
	return &Pool{
		PoolID:      p.Id,
		ChainID:     ExtractChainIDFromPoolID(p.Id),
		Amount:      p.Amount,
		TotalPoints: p.TotalPoolPoints,
		LPCount:     uint16(len(p.Points)),
	}
}

// ExtractChainIDFromPoolID extracts the chain ID from a pool ID.
// Pool IDs are encoded as: TypeAddend + ChainID
func ExtractChainIDFromPoolID(poolID uint64) uint64 {
	if poolID >= EscrowPoolAddend {
		return poolID - EscrowPoolAddend
	}
	if poolID >= LiquidityPoolAddend {
		return poolID - LiquidityPoolAddend
	}
	if poolID >= HoldingPoolAddend {
		return poolID - HoldingPoolAddend
	}
	return poolID // RewardPoolAddend is 0
}

// GetPoolType returns the pool type based on the pool ID.
func GetPoolType(poolID uint64) string {
	if poolID >= EscrowPoolAddend {
		return "escrow"
	}
	if poolID >= LiquidityPoolAddend {
		return "liquidity"
	}
	if poolID >= HoldingPoolAddend {
		return "holding"
	}
	return "reward"
}

// PoolPointsHolder holds a single holder's points in a pool for database insertion.
type PoolPointsHolder struct {
	Address             string
	PoolID              uint64
	Committee           uint64
	Points              uint64
	LiquidityPoolPoints uint64 // Pool's TotalPoolPoints
	LiquidityPoolID     uint64 // Computed: LiquidityPoolAddend + Committee
	Height              uint64
	HeightTime          time.Time
}

// PoolPointsHoldersFromFSM extracts all pool point holders from a pool.
// Only applicable for liquidity pools (pools with Points data).
func PoolPointsHoldersFromFSM(p *fsm.Pool) []*PoolPointsHolder {
	if len(p.Points) == 0 {
		return nil
	}

	committee := ExtractChainIDFromPoolID(p.Id)
	holders := make([]*PoolPointsHolder, 0, len(p.Points))

	for _, pp := range p.Points {
		holders = append(holders, &PoolPointsHolder{
			Address:             bytesToHex(pp.Address),
			PoolID:              p.Id,
			Committee:           committee,
			Points:              pp.Points,
			LiquidityPoolPoints: p.TotalPoolPoints,
			LiquidityPoolID:     LiquidityPoolAddend + committee,
		})
	}

	return holders
}
