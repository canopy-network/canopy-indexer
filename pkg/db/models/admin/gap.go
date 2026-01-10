package admin

// Gap represents a missing height range (gap) in the indexing progress for a specific blockchain.
type Gap struct {
	From uint64 `ch:"from_h"`
	To   uint64 `ch:"to_h"`
}
