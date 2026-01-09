package rpc

// RPC endpoint paths for Canopy blockchain queries.
const (
	// Block and chain queries
	headPath          = "/v1/query/height"
	blockByHeightPath = "/v1/query/block-by-height"
	certByHeightPath  = "/v1/query/cert-by-height"

	// Transaction and event queries
	txsByHeightPath    = "/v1/query/txs-by-height"
	eventsByHeightPath = "/v1/query/events-by-height"

	// Account queries
	accountsByHeightPath = "/v1/query/accounts"

	// Order queries
	ordersByHeightPath = "/v1/query/orders"

	// Pool queries
	poolsPath = "/v1/query/pools"

	// DEX queries
	dexPricePath             = "/v1/query/dex-price"
	dexBatchByHeightPath     = "/v1/query/dex-batch"
	nextDexBatchByHeightPath = "/v1/query/next-dex-batch"

	// Parameter queries
	allParamsPath = "/v1/query/params"
	feeParamsPath = "/v1/query/fee-params"
	conParamsPath = "/v1/query/con-params"
	valParamsPath = "/v1/query/val-params"
	govParamsPath = "/v1/query/gov-params"

	// Supply queries
	supplyPath = "/v1/query/supply"

	// Validator queries
	validatorsPath    = "/v1/query/validators"
	nonSignersPath    = "/v1/query/non-signers"
	doubleSignersPath = "/v1/query/double-signers"

	// Committee queries
	committeesDataPath       = "/v1/query/committees-data"
	indexerBlobsPath         = "/v1/query/indexer-blobs"
	subsidizedCommitteesPath = "/v1/query/subsidized-committees"
	retiredCommitteesPath    = "/v1/query/retired-committees"

	// Governance queries
	pollPath     = "/v1/gov/poll"
	proposalPath = "/v1/gov/proposals"
)
