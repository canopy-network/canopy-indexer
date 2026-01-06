package transform

import (
	"encoding/json"
	"time"

	"github.com/canopy-network/canopy/lib"
)

// Event holds the transformed event data for database insertion.
type Event struct {
	Height      uint64
	HeightTime  time.Time
	ChainID     uint64
	Address     string
	Reference   string
	EventType   string
	BlockHeight uint64
	Amount      uint64
	Msg         string
}

// EventFromLib converts a lib.Event to the database Event model.
// HeightTime must be set by the caller.
func EventFromLib(e *lib.Event) (*Event, error) {
	// Marshal full event to JSON
	msgJSON, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	evt := &Event{
		Height:      e.Height,
		ChainID:     e.ChainId,
		Address:     bytesToHex(e.Address),
		Reference:   e.Reference,
		EventType:   e.EventType,
		BlockHeight: e.BlockHeight,
		Msg:         string(msgJSON),
	}

	// Extract amount from type-specific fields
	extractEventAmount(e, evt)

	return evt, nil
}

// extractEventAmount extracts amount field from event-specific messages.
func extractEventAmount(e *lib.Event, evt *Event) {
	switch msg := e.Msg.(type) {
	case *lib.Event_Reward:
		evt.Amount = msg.Reward.Amount
	case *lib.Event_Slash:
		evt.Amount = msg.Slash.Amount
	case *lib.Event_DexLiquidityDeposit:
		evt.Amount = msg.DexLiquidityDeposit.Amount
	}
}
