package indexer

import (
	"context"
	"time"

	"github.com/canopy-network/pgindexer/pkg/transform"
	"github.com/jackc/pgx/v5"
)

func (idx *Indexer) indexOrders(ctx context.Context, chainID, height uint64, blockTime time.Time) error {
	rpc, err := idx.rpcForChain(chainID)
	if err != nil {
		return err
	}
	orders, err := rpc.OrdersByHeight(ctx, height)
	if err != nil {
		return err
	}

	if len(orders) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, order := range orders {
		o := transform.OrderFromLib(order)
		batch.Queue(`
			INSERT INTO orders (
				chain_id, order_id, committee, data, amount_for_sale, requested_amount,
				seller_receive_address, buyer_send_address, buyer_receive_address,
				buyer_chain_deadline, sellers_send_address, status, height, height_time
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			ON CONFLICT (chain_id, order_id, height) DO UPDATE SET
				amount_for_sale = EXCLUDED.amount_for_sale,
				requested_amount = EXCLUDED.requested_amount,
				status = EXCLUDED.status
		`,
			chainID,
			o.OrderID,
			o.Committee,
			"",                     // data
			o.AmountForSale,        // amount_for_sale
			o.RequestedAmount,      // requested_amount
			o.SellerReceiveAddress, // seller_receive_address
			o.BuyerSendAddress,     // buyer_send_address
			o.BuyerReceiveAddress,  // buyer_receive_address
			o.BuyerChainDeadline,   // buyer_chain_deadline
			o.SellersSendAddress,   // sellers_send_address
			o.Status,               // status
			height,
			blockTime,
		)
	}

	br := idx.db.SendBatch(ctx, batch)
	defer br.Close()

	for range orders {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}
