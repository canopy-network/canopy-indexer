package transform

import (
	"time"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
)

// Validator holds the transformed validator data for database insertion.
type Validator struct {
	Address         string
	PublicKey       string
	NetAddress      string
	StakedAmount    uint64
	MaxPausedHeight uint64
	UnstakingHeight uint64
	Output          string
	Delegate        bool
	Compound        bool
	Status          string
	Height          uint64
	HeightTime      time.Time
}

// ValidatorFromFSM converts a fsm.Validator to the database Validator model.
// Height and HeightTime must be set by the caller.
func ValidatorFromFSM(v *fsm.Validator) *Validator {
	// Derive status from heights
	status := "active"
	if v.UnstakingHeight > 0 {
		status = "unstaking"
	} else if v.MaxPausedHeight > 0 {
		status = "paused"
	}

	return &Validator{
		Address:         bytesToHex(v.Address),
		PublicKey:       bytesToHex(v.PublicKey),
		NetAddress:      v.NetAddress,
		StakedAmount:    v.StakedAmount,
		MaxPausedHeight: v.MaxPausedHeight,
		UnstakingHeight: v.UnstakingHeight,
		Output:          bytesToHex(v.Output),
		Delegate:        v.Delegate,
		Compound:        v.Compound,
		Status:          status,
	}
}

// NonSigner holds the transformed non-signer data for database insertion.
type NonSigner struct {
	Address           string
	MissedBlocksCount uint64
	Height            uint64
	HeightTime        time.Time
}

// NonSignerFromFSM converts a fsm.NonSigner to the database NonSigner model.
func NonSignerFromFSM(ns *fsm.NonSigner) *NonSigner {
	return &NonSigner{
		Address:           bytesToHex(ns.Address),
		MissedBlocksCount: ns.Counter,
	}
}

// DoubleSigner holds the transformed double-signer data for database insertion.
type DoubleSigner struct {
	Address    string
	Heights    []uint64
	Height     uint64
	HeightTime time.Time
}

// DoubleSignerFromLib converts a lib.DoubleSigner to the database DoubleSigner model.
func DoubleSignerFromLib(ds *lib.DoubleSigner) *DoubleSigner {
	return &DoubleSigner{
		Address: bytesToHex(ds.Id),
		Heights: ds.Heights,
	}
}
