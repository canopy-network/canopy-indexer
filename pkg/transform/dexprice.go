package transform

import (
	"fmt"
	"time"

	"github.com/canopy-network/canopy/lib"
)

// DexPrice holds the transformed DEX price data for database insertion.
type DexPrice struct {
	LocalChainID  uint64
	RemoteChainID uint64
	Height        uint64
	HeightTime    time.Time
	LocalPool     uint64
	RemotePool    uint64
	PriceE6       uint64
	PriceDelta    int64
	LocalDelta    int64
	RemoteDelta   int64
}

// DexPriceFromLib converts a lib.DexPrice to the database model.
// Height and HeightTime must be set by the caller.
func DexPriceFromLib(r *lib.DexPrice) *DexPrice {
	return &DexPrice{
		LocalChainID:  r.LocalChainId,
		RemoteChainID: r.RemoteChainId,
		LocalPool:     r.LocalPool,
		RemotePool:    r.RemotePool,
		PriceE6:       r.E6ScaledPrice,
	}
}

// DexPriceKey creates a unique key for DexPrice lookups based on chain pair.
func DexPriceKey(localChainID, remoteChainID uint64) string {
	return fmt.Sprintf("%d:%d", localChainID, remoteChainID)
}
