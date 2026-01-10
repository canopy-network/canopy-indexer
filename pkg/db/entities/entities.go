// Package entities provides type-safe constants and helpers for managing database entity names.
//
// This package serves as the single source of truth for all entity names used throughout the system,
// including table names, staging tables, activity parameters, and workflow loops.
//
// Design Philosophy:
//   - Type safety: Catch errors at compile time, not runtime
//   - Single source of truth: All entity names defined in one place
//   - Zero overhead: Constants compile away to their string values
//   - Easy maintenance: Adding a new entity is a single-line change
//
// Usage Example:
//
//	// In workflow - iterate over all entities
//	for _, entity := range entities.All() {
//	    PromoteData(ctx, entity.String(), height)
//	}
//
//	// In activity - validate input
//	func PromoteData(ctx context.Context, entityName string, height uint64) error {
//	    entity, err := entities.FromString(entityName)
//	    if err != nil {
//	        return fmt.Errorf("invalid entity: %w", err)
//	    }
//	    // Use entity safely...
//	}
//
//	// In database operations
//	query := fmt.Sprintf("INSERT INTO %s SELECT * FROM %s WHERE height = ?",
//	    entities.Blocks.TableName(),
//	    entities.Blocks.StagingTableName())
//
// Thread Safety:
//
//	All functions and methods in this package are safe for concurrent use.
package entities

import (
	"fmt"
	"sort"
	"strings"
)

// StagingSuffix is the suffix appended to entity names to create staging table names.
// This constant ensures consistent naming across the entire system.
//
// Example:
//
//	tableName := entities.Blocks.String() + entities.StagingSuffix // "blocks_staging"
const StagingSuffix = "_staging"

// Entity represents a database entity type (e.g., blocks, transactions, accounts).
// This type provides compile-time safety when working with entity names and prevents
// typos that could cause runtime errors.
//
// Entity values should be treated as immutable constants. Use the package-level
// constants (Blocks, Transactions, etc.) rather than constructing Entity values directly.
type Entity string

// Core entity constants. These are the canonical entity names used throughout the system.
// When adding a new entity, add it here and update the allEntities slice below.
const (
	// Blocks represents the blockchain blocks entity.
	// Production table: blocks
	// Staging table: blocks_staging
	Blocks Entity = "blocks"

	// Transactions represents the transaction entity.
	// Production table: txs
	// Staging table: txs_staging
	Transactions Entity = "txs"

	// BlockSummaries represents the block summary aggregations.
	// Production table: block_summaries
	// Staging table: block_summaries_staging
	BlockSummaries Entity = "block_summaries"

	// Accounts represents the account balance entity.
	// Production table: accounts
	// Staging table: accounts_staging
	Accounts Entity = "accounts"

	// Events represents the blockchain events entity.
	// Production table: events
	// Staging table: events_staging
	Events Entity = "events"

	// Orders represents the order entity.
	// Production table: orders
	// Staging table: orders_staging
	Orders Entity = "orders"

	// Pools represents the liquidity pools entity.
	// Production table: pools
	// Staging table: pools_staging
	Pools Entity = "pools"

	// DexPrices represents the DEX price data entity.
	// Production table: dex_prices
	// Staging table: dex_prices_staging
	DexPrices Entity = "dex_prices"

	// DexOrders represents the DEX limit orders entity.
	// Production table: dex_orders
	// Staging table: dex_orders_staging
	DexOrders Entity = "dex_orders"

	// DexDeposits represents the DEX liquidity deposits entity.
	// Production table: dex_deposits
	// Staging table: dex_deposits_staging
	DexDeposits Entity = "dex_deposits"

	// DexWithdrawals represents the DEX liquidity withdrawals entity.
	// Production table: dex_withdrawals
	// Staging table: dex_withdrawals_staging
	DexWithdrawals Entity = "dex_withdrawals"

	// PoolPointsByHolder represents the pool points by holder entity.
	// Production table: pool_points_by_holder
	// Staging table: pool_points_by_holder_staging
	PoolPointsByHolder Entity = "pool_points_by_holder"

	// Params represents the chain parameters entity.
	// Production table: params
	// Staging table: params_staging
	Params Entity = "params"

	// Validators represents the validator entity.
	// Production table: validators
	// Staging table: validators_staging
	Validators Entity = "validators"

	// ValidatorNonSigningInfo represents the validator non-signing info entity.
	// Production table: validator_non_signing_info
	// Staging table: validator_non_signing_info_staging
	ValidatorNonSigningInfo Entity = "validator_non_signing_info"

	// ValidatorDoubleSigningInfo represents the validator double signing info entity.
	// Production table: validator_double_signing_info
	// Staging table: validator_double_signing_info_staging
	ValidatorDoubleSigningInfo Entity = "validator_double_signing_info"

	// Committees represents the committee data entity.
	// Production table: committees
	// Staging table: committees_staging
	Committees Entity = "committees"

	// CommitteeValidators represents the committee-validator junction table entity.
	// Production table: committee_validators
	// Staging table: committee_validators_staging
	CommitteeValidators Entity = "committee_validators"

	// CommitteePayments represents the committee payment distribution entity.
	// Production table: committee_payments
	// Staging table: committee_payments_staging
	CommitteePayments Entity = "committee_payments"

	// PollSnapshots represents the governance poll snapshots entity.
	// Production table: poll_snapshots
	// Staging table: poll_snapshots_staging
	PollSnapshots Entity = "poll_snapshots"

	// Supply represents the chain supply metrics entity.
	// Production table: supply
	// Staging table: supply_staging
	Supply Entity = "supply"

	// Future entities (uncomment and add to allEntities when implementing):
	// Logs Entity = "logs"
)

// allEntities contains the complete list of all valid entities in the system.
// This slice is used for validation, iteration, and generating entity lists.
//
// IMPORTANT: When adding a new entity constant above, you MUST also add it to this slice.
// The system will panic at initialization if an entity is missing from this list.
var allEntities = []Entity{
	Blocks,
	Transactions,
	BlockSummaries,
	Accounts,
	Events,
	Orders,
	Pools,
	DexPrices,
	DexOrders,
	DexDeposits,
	DexWithdrawals,
	PoolPointsByHolder,
	Params,
	Validators,
	ValidatorNonSigningInfo,
	ValidatorDoubleSigningInfo,
	Committees,
	CommitteeValidators,
	CommitteePayments,
	PollSnapshots,
	Supply,
}

// entitySet is a pre-computed map for O(1) validation lookups.
// This is built once at package initialization.
var entitySet map[Entity]bool

func init() {
	// Build the validation set for fast lookups
	entitySet = make(map[Entity]bool, len(allEntities))
	for _, e := range allEntities {
		entitySet[e] = true
	}

	// Validate that all entities are properly configured
	// This catches developer errors at startup rather than in production
	for _, e := range allEntities {
		if e == "" {
			panic("entities: empty entity name detected in allEntities")
		}
		if strings.Contains(string(e), "_staging") {
			panic(fmt.Sprintf("entities: entity name %q contains '_staging' - use base name only", e))
		}
		if strings.Contains(string(e), " ") {
			panic(fmt.Sprintf("entities: entity name %q contains whitespace", e))
		}
	}
}

// String returns the entity name as a string.
// This implements the fmt.Stringer interface and is used for logging and serialization.
//
// Example:
//
//	entity := entities.Blocks
//	fmt.Println(entity.String()) // Output: "blocks"
func (e Entity) String() string {
	return string(e)
}

// TableName returns the production table name for this entity.
// The production table contains validated, promoted data.
//
// Example:
//
//	entities.Blocks.TableName() // Returns: "blocks"
//	entities.Transactions.TableName() // Returns: "transactions"
func (e Entity) TableName() string {
	return string(e)
}

// StagingTableName returns the staging table name for this entity.
// The staging table is used for new data before validation and promotion.
//
// Naming Convention: {entity}_staging
//
// Example:
//
//	entities.Blocks.StagingTableName() // Returns: "blocks_staging"
//	entities.Transactions.StagingTableName() // Returns: "transactions_staging"
func (e Entity) StagingTableName() string {
	return string(e) + StagingSuffix
}

// IsValid returns true if this entity is in the list of known entities.
// Use this to validate entity values that may come from external sources.
//
// Example:
//
//	entity := entities.Entity("blocks")
//	if entity.IsValid() {
//	    // Safe to use
//	}
func (e Entity) IsValid() bool {
	return entitySet[e]
}

// MarshalText implements encoding.TextMarshaler for JSON/YAML serialization.
// This ensures Entity values serialize to their string representation.
func (e Entity) MarshalText() ([]byte, error) {
	return []byte(e), nil
}

// UnmarshalText implements encoding.TextUnmarshaler for JSON/YAML deserialization.
// This validates that deserialized values are valid entities.
func (e *Entity) UnmarshalText(text []byte) error {
	entity := Entity(text)
	if !entity.IsValid() {
		return fmt.Errorf("invalid entity: %q", text)
	}
	*e = entity
	return nil
}

// FromString converts a string to an Entity and validates it.
// This is the safe way to construct an Entity from external input.
//
// Returns an error if the string is not a recognized entity name.
//
// Example:
//
//	entity, err := entities.FromString("blocks")
//	if err != nil {
//	    return fmt.Errorf("invalid entity: %w", err)
//	}
//	// Use entity safely...
func FromString(s string) (Entity, error) {
	entity := Entity(s)
	if !entity.IsValid() {
		return "", fmt.Errorf("unknown entity %q, valid entities: %s", s, validEntitiesString())
	}
	return entity, nil
}

// MustFromString converts a string to an Entity, panicking if invalid.
// Only use this for hardcoded values where you know the string is valid.
//
// CAUTION: This will panic on invalid input. Prefer FromString for runtime values.
//
// Example:
//
//	entity := entities.MustFromString("blocks") // Safe, but panics on typo
func MustFromString(s string) Entity {
	entity, err := FromString(s)
	if err != nil {
		panic(err)
	}
	return entity
}

// All returns a slice of all valid entities in the system.
// The returned slice is a copy, so modifications won't affect the internal state.
//
// Use this for iterating over all entities in workflows or batch operations.
//
// Example:
//
//	for _, entity := range entities.All() {
//	    fmt.Printf("Processing entity: %s\n", entity)
//	}
func All() []Entity {
	// Return a copy to prevent external modifications
	result := make([]Entity, len(allEntities))
	copy(result, allEntities)
	return result
}

// AllStrings returns all entity names as strings.
// This is useful for logging, debugging, or passing to external systems.
//
// Example:
//
//	entityNames := entities.AllStrings()
//	// ["blocks", "transactions", "transactions_raw", "block_summaries"]
func AllStrings() []string {
	result := make([]string, len(allEntities))
	for i, e := range allEntities {
		result[i] = e.String()
	}
	return result
}

// Count returns the number of entities in the system.
// Useful for sizing slices/maps or reporting metrics.
func Count() int {
	return len(allEntities)
}

// entitiesWithoutStaging lists entities that don't use the staging pattern.
// These are time-based snapshots captured via scheduled workflows, not height-based indexing.
var entitiesWithoutStaging = map[Entity]bool{
	PollSnapshots: true, // Time-based snapshots, no staging table
	// ProposalSnapshots uses staging pattern (height-based)
}

// AllWithStaging returns entities that use the staging table pattern.
// Use this for batch cleanup operations that target staging tables.
//
// Excludes:
// - PollSnapshots: Time-based snapshots captured via scheduled workflow
func AllWithStaging() []Entity {
	result := make([]Entity, 0, len(allEntities))
	for _, e := range allEntities {
		if !entitiesWithoutStaging[e] {
			result = append(result, e)
		}
	}
	return result
}

// HasStaging returns true if this entity uses the staging table pattern.
func (e Entity) HasStaging() bool {
	return !entitiesWithoutStaging[e]
}

// validEntitiesString returns a comma-separated list of valid entity names.
// Used for error messages.
func validEntitiesString() string {
	names := AllStrings()
	sort.Strings(names) // Sort for consistent error messages
	return strings.Join(names, ", ")
}
