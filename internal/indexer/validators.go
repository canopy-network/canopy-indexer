package indexer

import (
	"context"
	"time"

	"github.com/canopy-network/canopy/fsm"
	"github.com/canopy-network/canopy/lib"
	"github.com/canopy-network/pgindexer/pkg/transform"
	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"
)

func (idx *Indexer) indexValidators(ctx context.Context, chainID, height uint64, blockTime time.Time) error {
	rpc, err := idx.rpcForChain(chainID)
	if err != nil {
		return err
	}

	// Parallel fetch: current validators, previous validators, subsidized/retired committees
	var (
		currentValidators  []*fsm.Validator
		previousValidators []*fsm.Validator
		subsidized         []uint64
		retired            []uint64
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		currentValidators, err = rpc.ValidatorsByHeight(gCtx, height)
		return err
	})

	g.Go(func() error {
		if height <= 1 {
			previousValidators = make([]*fsm.Validator, 0)
			return nil
		}
		var err error
		previousValidators, err = rpc.ValidatorsByHeight(gCtx, height-1)
		return err
	})

	g.Go(func() error {
		var err error
		subsidized, err = rpc.SubsidizedCommitteesByHeight(gCtx, height)
		return err
	})

	g.Go(func() error {
		var err error
		retired, err = rpc.RetiredCommitteesByHeight(gCtx, height)
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	// Build lookup maps
	subsidizedMap := make(map[uint64]bool)
	for _, id := range subsidized {
		subsidizedMap[id] = true
	}
	retiredMap := make(map[uint64]bool)
	for _, id := range retired {
		retiredMap[id] = true
	}

	// Build previous validator map for O(1) lookup
	prevValidatorMap := make(map[string]*fsm.Validator, len(previousValidators))
	for _, val := range previousValidators {
		prevValidatorMap[transform.BytesToHex(val.Address)] = val
	}

	// Compare and collect changed validators
	type changedValidator struct {
		val    *fsm.Validator
		status string
	}
	var changedValidators []changedValidator

	for _, curr := range currentValidators {
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
	if len(changedValidators) > 0 {
		batch := &pgx.Batch{}
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
				chainID,
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
				height,
				blockTime,
			)
		}

		br := idx.db.SendBatch(ctx, batch)
		for range changedValidators {
			if _, err := br.Exec(); err != nil {
				br.Close()
				return err
			}
		}
		br.Close()
	}

	// Insert committee_validators for changed validators
	if len(changedValidators) > 0 {
		batch := &pgx.Batch{}
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
					chainID,
					committeeID,
					addrHex,
					val.StakedAmount,
					cv.status,
					val.Delegate,
					val.Compound,
					height,
					blockTime,
				)
			}
		}

		if batch.Len() > 0 {
			br := idx.db.SendBatch(ctx, batch)
			for i := 0; i < batch.Len(); i++ {
				if _, err := br.Exec(); err != nil {
					br.Close()
					return err
				}
			}
			br.Close()
		}
	}

	// Index non-signers and double-signers with change detection
	if err := idx.indexNonSigners(ctx, chainID, height, blockTime); err != nil {
		return err
	}

	return idx.indexDoubleSigners(ctx, chainID, height, blockTime)
}

func (idx *Indexer) indexNonSigners(ctx context.Context, chainID, height uint64, blockTime time.Time) error {
	rpc, err := idx.rpcForChain(chainID)
	if err != nil {
		return err
	}

	// Parallel fetch: current and previous non-signers
	var (
		currentNonSigners  []*fsm.NonSigner
		previousNonSigners []*fsm.NonSigner
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		currentNonSigners, err = rpc.NonSignersByHeight(gCtx, height)
		return err
	})

	g.Go(func() error {
		if height <= 1 {
			previousNonSigners = make([]*fsm.NonSigner, 0)
			return nil
		}
		var err error
		previousNonSigners, err = rpc.NonSignersByHeight(gCtx, height-1)
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	// Build previous non-signer map for O(1) lookup
	prevNonSignerMap := make(map[string]*fsm.NonSigner, len(previousNonSigners))
	for _, ns := range previousNonSigners {
		prevNonSignerMap[transform.BytesToHex(ns.Address)] = ns
	}

	// Build current non-signer map for reset detection
	currentNonSignerMap := make(map[string]*fsm.NonSigner, len(currentNonSigners))
	for _, ns := range currentNonSigners {
		currentNonSignerMap[transform.BytesToHex(ns.Address)] = ns
	}

	type nonSigningInfoRecord struct {
		address           string
		missedBlocksCount uint64
	}
	var changedRecords []nonSigningInfoRecord

	// Phase 1: Check current non-signers for new entries and updates
	for _, curr := range currentNonSigners {
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
	for _, prev := range previousNonSigners {
		addrHex := transform.BytesToHex(prev.Address)
		if _, exists := currentNonSignerMap[addrHex]; !exists {
			// Validator was a non-signer previously but not anymore (counter was reset)
			changedRecords = append(changedRecords, nonSigningInfoRecord{
				address:           addrHex,
				missedBlocksCount: 0, // Reset to zero
			})
		}
	}

	if len(changedRecords) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, rec := range changedRecords {
		batch.Queue(`
			INSERT INTO validator_non_signing_info (
				chain_id, address, missed_blocks_count, last_signed_height, height, height_time
			) VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (chain_id, address, height) DO UPDATE SET
				missed_blocks_count = EXCLUDED.missed_blocks_count,
				last_signed_height = EXCLUDED.last_signed_height
		`,
			chainID,
			rec.address,
			rec.missedBlocksCount,
			height, // last_signed_height
			height,
			blockTime,
		)
	}

	br := idx.db.SendBatch(ctx, batch)
	defer br.Close()

	for range changedRecords {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}

func (idx *Indexer) indexDoubleSigners(ctx context.Context, chainID, height uint64, blockTime time.Time) error {
	rpc, err := idx.rpcForChain(chainID)
	if err != nil {
		return err
	}

	// Parallel fetch: current and previous double-signers
	var (
		currentDoubleSigners  []*lib.DoubleSigner
		previousDoubleSigners []*lib.DoubleSigner
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		currentDoubleSigners, err = rpc.DoubleSignersByHeight(gCtx, height)
		return err
	})

	g.Go(func() error {
		if height <= 1 {
			previousDoubleSigners = make([]*lib.DoubleSigner, 0)
			return nil
		}
		var err error
		previousDoubleSigners, err = rpc.DoubleSignersByHeight(gCtx, height-1)
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	// Build previous double-signer map for O(1) lookup (stores evidence count)
	prevDoubleSignerMap := make(map[string]uint64, len(previousDoubleSigners))
	for _, ds := range previousDoubleSigners {
		prevDoubleSignerMap[transform.BytesToHex(ds.Id)] = uint64(len(ds.Heights))
	}

	// Build current double-signer map for reset detection
	currentDoubleSignerMap := make(map[string]*lib.DoubleSigner, len(currentDoubleSigners))
	for _, ds := range currentDoubleSigners {
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
	for _, curr := range currentDoubleSigners {
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
	for _, prev := range previousDoubleSigners {
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

	if len(changedRecords) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
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
			chainID,
			rec.address,
			rec.evidenceCount,
			rec.firstEvidenceHeight,
			rec.lastEvidenceHeight,
			height,
			blockTime,
		)
	}

	br := idx.db.SendBatch(ctx, batch)
	defer br.Close()

	for range changedRecords {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
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
