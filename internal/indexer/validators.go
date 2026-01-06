package indexer

import (
	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/pgindexer/pkg/transform"
	"github.com/jackc/pgx/v5"
)

func (idx *Indexer) writeValidators(batch *pgx.Batch, data *BlockData) {
	// Build previous validator map for O(1) lookup
	prevValidatorMap := make(map[string]*fsm.Validator, len(data.ValidatorsPrevious))
	for _, val := range data.ValidatorsPrevious {
		prevValidatorMap[transform.BytesToHex(val.Address)] = val
	}

	// Compare and collect changed validators
	type changedValidator struct {
		val    *fsm.Validator
		status string
	}
	var changedValidators []changedValidator

	for _, curr := range data.ValidatorsCurrent {
		addrHex := transform.BytesToHex(curr.Address)
		prev := prevValidatorMap[addrHex]

		// Derive status
		status := "active"
		if curr.UnstakingHeight > 0 {
			status = "unstaking"
		} else if curr.MaxPausedHeight > 0 {
			status = "paused"
		}

		// Check if changed
		changed := false
		if prev == nil {
			// New validator
			changed = true
		} else {
			// Compare fields
			if curr.StakedAmount != prev.StakedAmount ||
				transform.BytesToHex(curr.PublicKey) != transform.BytesToHex(prev.PublicKey) ||
				curr.NetAddress != prev.NetAddress ||
				curr.MaxPausedHeight != prev.MaxPausedHeight ||
				curr.UnstakingHeight != prev.UnstakingHeight ||
				transform.BytesToHex(curr.Output) != transform.BytesToHex(prev.Output) ||
				curr.Delegate != prev.Delegate ||
				curr.Compound != prev.Compound ||
				!equalCommittees(curr.Committees, prev.Committees) {
				changed = true
			}
		}

		if changed {
			changedValidators = append(changedValidators, changedValidator{val: curr, status: status})
		}
	}

	// Insert changed validators
	for _, cv := range changedValidators {
		val := cv.val
		batch.Queue(`
			INSERT INTO validators (
				chain_id, address, public_key, net_address, staked_amount,
				max_paused_height, unstaking_height, output, delegate, compound,
				status, height, height_time
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (chain_id, address, height) DO UPDATE SET
				staked_amount = EXCLUDED.staked_amount,
				max_paused_height = EXCLUDED.max_paused_height,
				unstaking_height = EXCLUDED.unstaking_height,
				status = EXCLUDED.status
		`,
			data.ChainID,
			transform.BytesToHex(val.Address),
			transform.BytesToHex(val.PublicKey),
			val.NetAddress,
			val.StakedAmount,
			val.MaxPausedHeight,
			val.UnstakingHeight,
			transform.BytesToHex(val.Output),
			val.Delegate,
			val.Compound,
			cv.status,
			data.Height,
			data.BlockTime,
		)
	}

	// Insert committee_validators for changed validators
	for _, cv := range changedValidators {
		val := cv.val
		addrHex := transform.BytesToHex(val.Address)
		for _, committeeID := range val.Committees {
			batch.Queue(`
				INSERT INTO committee_validators (
					chain_id, committee_id, address, staked_amount, status,
					delegate, compound, height, height_time
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				ON CONFLICT (chain_id, committee_id, address, height) DO UPDATE SET
					staked_amount = EXCLUDED.staked_amount,
					status = EXCLUDED.status,
					delegate = EXCLUDED.delegate,
					compound = EXCLUDED.compound
			`,
				data.ChainID,
				committeeID,
				addrHex,
				val.StakedAmount,
				cv.status,
				val.Delegate,
				val.Compound,
				data.Height,
				data.BlockTime,
			)
		}
	}

	// Write non-signers
	idx.writeNonSigners(batch, data)

	// Write double-signers
	idx.writeDoubleSigners(batch, data)
}

func (idx *Indexer) writeNonSigners(batch *pgx.Batch, data *BlockData) {
	// Build previous non-signer map for O(1) lookup
	prevNonSignerMap := make(map[string]*fsm.NonSigner, len(data.NonSignersPrevious))
	for _, ns := range data.NonSignersPrevious {
		prevNonSignerMap[transform.BytesToHex(ns.Address)] = ns
	}

	// Build current non-signer map for reset detection
	currentNonSignerMap := make(map[string]*fsm.NonSigner, len(data.NonSignersCurrent))
	for _, ns := range data.NonSignersCurrent {
		currentNonSignerMap[transform.BytesToHex(ns.Address)] = ns
	}

	type nonSigningInfoRecord struct {
		address           string
		missedBlocksCount uint64
	}
	var changedRecords []nonSigningInfoRecord

	// Phase 1: Check current non-signers for new entries and updates
	for _, curr := range data.NonSignersCurrent {
		addrHex := transform.BytesToHex(curr.Address)
		prev := prevNonSignerMap[addrHex]

		changed := false
		if prev == nil {
			// New non-signer entry
			changed = true
		} else if curr.Counter != prev.Counter {
			// Counter changed
			changed = true
		}

		if changed {
			changedRecords = append(changedRecords, nonSigningInfoRecord{
				address:           addrHex,
				missedBlocksCount: curr.Counter,
			})
		}
	}

	// Phase 2: Detect resets - existed at H-1 but not at H
	for _, prev := range data.NonSignersPrevious {
		addrHex := transform.BytesToHex(prev.Address)
		if _, exists := currentNonSignerMap[addrHex]; !exists {
			// Validator was a non-signer previously but not anymore (counter was reset)
			changedRecords = append(changedRecords, nonSigningInfoRecord{
				address:           addrHex,
				missedBlocksCount: 0, // Reset to zero
			})
		}
	}

	for _, rec := range changedRecords {
		batch.Queue(`
			INSERT INTO validator_non_signing_info (
				chain_id, address, missed_blocks_count, last_signed_height, height, height_time
			) VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (chain_id, address, height) DO UPDATE SET
				missed_blocks_count = EXCLUDED.missed_blocks_count,
				last_signed_height = EXCLUDED.last_signed_height
		`,
			data.ChainID,
			rec.address,
			rec.missedBlocksCount,
			data.Height, // last_signed_height
			data.Height,
			data.BlockTime,
		)
	}
}

func (idx *Indexer) writeDoubleSigners(batch *pgx.Batch, data *BlockData) {
	// Build previous double-signer map for O(1) lookup (stores evidence count)
	prevDoubleSignerMap := make(map[string]uint64, len(data.DoubleSignersPrevious))
	for _, ds := range data.DoubleSignersPrevious {
		prevDoubleSignerMap[transform.BytesToHex(ds.Id)] = uint64(len(ds.Heights))
	}

	// Build current double-signer map for reset detection
	currentDoubleSignerMap := make(map[string]*lib.DoubleSigner, len(data.DoubleSignersCurrent))
	for _, ds := range data.DoubleSignersCurrent {
		currentDoubleSignerMap[transform.BytesToHex(ds.Id)] = ds
	}

	type doubleSigningInfoRecord struct {
		address             string
		evidenceCount       uint64
		firstEvidenceHeight uint64
		lastEvidenceHeight  uint64
	}
	var changedRecords []doubleSigningInfoRecord

	// Phase 1: Check current double-signers for new entries and updates
	for _, curr := range data.DoubleSignersCurrent {
		addrHex := transform.BytesToHex(curr.Id)
		currentCount := uint64(len(curr.Heights))
		prevCount := prevDoubleSignerMap[addrHex]

		// Snapshot-on-change: only insert if evidence count changed
		if currentCount != prevCount {
			var firstHeight, lastHeight uint64
			if len(curr.Heights) > 0 {
				firstHeight = curr.Heights[0]
				lastHeight = curr.Heights[len(curr.Heights)-1]
			}

			changedRecords = append(changedRecords, doubleSigningInfoRecord{
				address:             addrHex,
				evidenceCount:       currentCount,
				firstEvidenceHeight: firstHeight,
				lastEvidenceHeight:  lastHeight,
			})
		}
	}

	// Phase 2: Detect resets - existed at H-1 but not at H
	for _, prev := range data.DoubleSignersPrevious {
		addrHex := transform.BytesToHex(prev.Id)
		if _, exists := currentDoubleSignerMap[addrHex]; !exists {
			// Evidence was cleared/reset
			changedRecords = append(changedRecords, doubleSigningInfoRecord{
				address:             addrHex,
				evidenceCount:       0,
				firstEvidenceHeight: 0,
				lastEvidenceHeight:  0,
			})
		}
	}

	for _, rec := range changedRecords {
		batch.Queue(`
			INSERT INTO validator_double_signing_info (
				chain_id, address, evidence_count, first_evidence_height, last_evidence_height, height, height_time
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (chain_id, address, height) DO UPDATE SET
				evidence_count = EXCLUDED.evidence_count,
				first_evidence_height = EXCLUDED.first_evidence_height,
				last_evidence_height = EXCLUDED.last_evidence_height
		`,
			data.ChainID,
			rec.address,
			rec.evidenceCount,
			rec.firstEvidenceHeight,
			rec.lastEvidenceHeight,
			data.Height,
			data.BlockTime,
		)
	}
}

// equalCommittees compares two committee slices for equality.
func equalCommittees(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[uint64]bool, len(a))
	for _, c := range a {
		aMap[c] = true
	}
	for _, c := range b {
		if !aMap[c] {
			return false
		}
	}
	return true
}
