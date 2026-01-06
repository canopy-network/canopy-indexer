package transform

import (
	"time"

	"github.com/canopy-network/canopy/lib"
)

// Order holds the transformed order data for database insertion.
type Order struct {
	OrderID              string
	Committee            uint64
	Data                 string
	AmountForSale        uint64
	RequestedAmount      uint64
	SellerReceiveAddress string
	BuyerSendAddress     string
	BuyerReceiveAddress  string
	BuyerChainDeadline   uint64
	SellersSendAddress   string
	Status               string
	Height               uint64
	HeightTime           time.Time
}

// OrderFromLib converts a lib.SellOrder to the database Order model.
// Height, HeightTime, and Status must be set by the caller.
func OrderFromLib(o *lib.SellOrder) *Order {
	return &Order{
		OrderID:              bytesToHex(o.Id),
		Committee:            o.Committee,
		Data:                 bytesToHex(o.Data),
		AmountForSale:        o.AmountForSale,
		RequestedAmount:      o.RequestedAmount,
		SellerReceiveAddress: bytesToHex(o.SellerReceiveAddress),
		BuyerSendAddress:     bytesToHex(o.BuyerSendAddress),
		BuyerReceiveAddress:  bytesToHex(o.BuyerReceiveAddress),
		BuyerChainDeadline:   o.BuyerChainDeadline,
		SellersSendAddress:   bytesToHex(o.SellersSendAddress),
		Status:               "open", // Default, can be overridden
	}
}
