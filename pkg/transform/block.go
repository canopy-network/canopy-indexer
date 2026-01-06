package transform

import (
	"time"

	"github.com/canopy-network/canopy/lib"
)

// Block holds the transformed block data for database insertion.
type Block struct {
	Height          uint64
	HeightTime      time.Time
	Hash            string
	ProposerAddress string
	TotalTxs        uint64
	NumTxs          uint64
}

// BlockFromResult converts a Canopy lib.BlockResult to the database Block model.
func BlockFromResult(result *lib.BlockResult) *Block {
	h := result.BlockHeader

	return &Block{
		Height:          h.Height,
		HeightTime:      time.UnixMicro(int64(h.Time)),
		Hash:            bytesToHex(h.Hash),
		ProposerAddress: bytesToHex(h.ProposerAddress),
		TotalTxs:        h.TotalTxs,
		NumTxs:          h.NumTxs,
	}
}
