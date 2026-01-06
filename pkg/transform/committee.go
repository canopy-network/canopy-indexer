package transform

import (
	"time"

	"github.com/canopy-network/canopy/lib"
)

// Committee holds the transformed committee data for database insertion.
type Committee struct {
	ChainID                uint64
	LastRootHeightUpdated  uint64
	LastChainHeightUpdated uint64
	NumberOfSamples        uint64
	Subsidized             bool
	Retired                bool
	Height                 uint64
	HeightTime             time.Time
}

// CommitteeFromLib converts a lib.CommitteeData to the database Committee model.
// Subsidized and Retired status must be set by the caller based on lookup maps.
// Height and HeightTime must be set by the caller.
func CommitteeFromLib(c *lib.CommitteeData) *Committee {
	return &Committee{
		ChainID:                c.ChainId,
		LastRootHeightUpdated:  c.LastRootHeightUpdated,
		LastChainHeightUpdated: c.LastChainHeightUpdated,
		NumberOfSamples:        c.NumberOfSamples,
	}
}

// CommitteePayment holds the transformed committee payment data.
type CommitteePayment struct {
	CommitteeID uint64
	Address     string
	Percent     uint64
	Height      uint64
	HeightTime  time.Time
}

// CommitteePaymentsFromLib extracts payment percents from CommitteeData.
func CommitteePaymentsFromLib(c *lib.CommitteeData) []*CommitteePayment {
	payments := make([]*CommitteePayment, 0, len(c.PaymentPercents))
	for _, pp := range c.PaymentPercents {
		payments = append(payments, &CommitteePayment{
			CommitteeID: c.ChainId,
			Address:     bytesToHex(pp.Address),
			Percent:     pp.Percent,
		})
	}
	return payments
}
