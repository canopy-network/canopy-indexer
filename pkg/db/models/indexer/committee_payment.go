package indexer

import "time"

const CommitteePaymentsProductionTableName = "committee_payments"
const CommitteePaymentsStagingTableName = CommitteePaymentsProductionTableName + "_staging"

// CommitteePaymentColumns defines the schema for the committee_payments table.
var CommitteePaymentColumns = []ColumnDef{
	{Name: "committee_id", Type: "UInt64", Codec: "ZSTD(1)"},
	{Name: "address", Type: "String", Codec: "ZSTD(1)"},
	{Name: "percent", Type: "UInt64", Codec: "ZSTD(1)"},
	{Name: "height", Type: "UInt64", Codec: "Delta, ZSTD(1)"},
	{Name: "height_time", Type: "DateTime64(3)", Codec: "DoubleDelta, ZSTD(1)"},
}

// CommitteePayment represents a payment recipient and their percentage share for a committee.
// Each committee can have multiple payment recipients (validators/delegates) who share rewards.
// This is tracked per height to show historical changes in payment distributions.
//
// Query patterns:
//   - Current payments for a committee: SELECT * FROM committee_payments FINAL WHERE committee_id = ? ORDER BY height DESC LIMIT 1
//   - All recipients for a committee: SELECT DISTINCT address FROM committee_payments FINAL WHERE committee_id = ?
//   - Payment history for an address: SELECT * FROM committee_payments FINAL WHERE address = ? ORDER BY height DESC
type CommitteePayment struct {
	CommitteeID uint64    `ch:"committee_id" json:"committee_id"` // Committee/chain ID
	Address     string    `ch:"address" json:"address"`           // Recipient address (validator/delegate)
	Percent     uint64    `ch:"percent" json:"percent"`           // Percentage share (0-100)
	Height      uint64    `ch:"height" json:"height"`             // Block height when this payment config was active
	HeightTime  time.Time `ch:"height_time" json:"height_time"`   // Block time
}
