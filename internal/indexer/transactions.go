package indexer

import (
	"encoding/json"

	"github.com/canopy-network/pgindexer/pkg/transform"
	"github.com/jackc/pgx/v5"
)

func (idx *Indexer) writeTransactions(batch *pgx.Batch, data *BlockData) {
	for i, tx := range data.Transactions {
		msgJSON, _ := json.Marshal(tx)

		batch.Queue(`
			INSERT INTO txs (
				chain_id, height, height_time, tx_hash, tx_index, tx_time,
				created_height, network_id, message_type, signer, fee, msg
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (chain_id, height, tx_hash) DO NOTHING
		`,
			data.ChainID,
			data.Height,
			data.BlockTime,
			tx.TxHash,
			i,
			data.BlockTime, // tx_time
			data.Height,    // created_height
			0,              // network_id
			tx.MessageType,
			transform.BytesToHex(tx.Sender),
			0, // fee (extract from tx if available)
			msgJSON,
		)
	}
}
