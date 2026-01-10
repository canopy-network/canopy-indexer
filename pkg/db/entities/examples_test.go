package entities_test

import (
	"context"
	"fmt"
	"log"

	"github.com/canopy-network/canopy-indexer/pkg/db/entities"
)

// Example_basicUsage demonstrates basic entity usage patterns.
func Example_basicUsage() {
	// Get entity constant
	entity := entities.Blocks

	// Get table names
	fmt.Println("Production table:", entity.TableName())
	fmt.Println("Staging table:", entity.StagingTableName())
	fmt.Println("Entity name:", entity.String())

	// Output:
	// Production table: blocks
	// Staging table: blocks_staging
	// Entity name: blocks
}

// Example_validation demonstrates entity validation from external input.
func Example_validation() {
	// Validate external input (e.g., from API, config, or user)
	input := "blocks"

	entity, err := entities.FromString(input)
	if err != nil {
		log.Fatalf("Invalid entity: %v", err)
	}

	fmt.Println("Valid entity:", entity)

	// Check if already have an entity value
	if entity.IsValid() {
		fmt.Println("Entity is valid")
	}

	// Output:
	// Valid entity: blocks
	// Entity is valid
}

// Example_invalidEntity demonstrates error handling for invalid entities.
func Example_invalidEntity() {
	// Try to create entity from invalid input
	_, err := entities.FromString("invalid_entity")
	if err != nil {
		fmt.Println("Error:", err)
		// Error message includes list of valid entities
	}

	// Output will include error message with valid entities list
}

// Example_iterateAllEntities shows how to process all entities.
func Example_iterateAllEntities() {
	// Get all entities for batch processing
	allEntities := entities.All()

	fmt.Printf("Processing %d entities:\n", len(allEntities))
	for _, entity := range allEntities {
		fmt.Printf("- %s (staging: %s)\n", entity.TableName(), entity.StagingTableName())
	}

	// Also available as strings only
	entityNames := entities.AllStrings()
	fmt.Printf("\nEntity names: %v\n", entityNames)
}

// Example_workflowUsage demonstrates usage in a Temporal workflow context.
func Example_workflowUsage() {
	// Simulate workflow context
	ctx := context.Background()
	height := uint64(1000)

	// Iterate through all entities to promote data
	for _, entity := range entities.All() {
		// Safe to use entity in activities
		promoteDataActivity(ctx, entity, height)
	}

	fmt.Println("All entities promoted")

	// Output:
	// Promoting blocks at height 1000
	// Promoting txs at height 1000
	// Promoting block_summaries at height 1000
	// Promoting accounts at height 1000
	// Promoting events at height 1000
	// Promoting orders at height 1000
	// Promoting pools at height 1000
	// Promoting dex_prices at height 1000
	// Promoting dex_orders at height 1000
	// Promoting dex_deposits at height 1000
	// Promoting dex_withdrawals at height 1000
	// Promoting pool_points_by_holder at height 1000
	// Promoting params at height 1000
	// Promoting validators at height 1000
	// Promoting validator_non_signing_info at height 1000
	// Promoting validator_double_signing_info at height 1000
	// Promoting committees at height 1000
	// Promoting committee_validators at height 1000
	// Promoting committee_payments at height 1000
	// Promoting poll_snapshots at height 1000
	// Promoting supply at height 1000
	// All entities promoted
}

// promoteDataActivity simulates a Temporal activity that promotes data.
func promoteDataActivity(ctx context.Context, entity entities.Entity, height uint64) {
	fmt.Printf("Promoting %s at height %d\n", entity, height)
}

// Example_activityValidation demonstrates input validation in activities.
func Example_activityValidation() {
	// Simulate activity input (entity name comes from workflow as string)
	type PromoteInput struct {
		EntityName string
		Height     uint64
	}

	input := PromoteInput{
		EntityName: "blocks",
		Height:     1000,
	}

	// Validate entity in activity
	entity, err := entities.FromString(input.EntityName)
	if err != nil {
		log.Fatalf("Invalid entity in activity input: %v", err)
	}

	// Now safely use entity
	fmt.Printf("Promoting %s to production table: %s\n",
		entity, entity.TableName())

	// Output:
	// Promoting blocks to production table: blocks
}

// Example_databaseOperations demonstrates database query construction.
func Example_databaseOperations() {
	entity := entities.Blocks
	height := uint64(1000)

	// Construct promote query
	promoteQuery := fmt.Sprintf(
		"INSERT INTO %s SELECT * FROM %s WHERE height = %d",
		entity.TableName(),
		entity.StagingTableName(),
		height,
	)
	fmt.Println("Promote:", promoteQuery)

	// Construct cleanup query
	cleanupQuery := fmt.Sprintf(
		"ALTER TABLE %s DELETE WHERE height = %d",
		entity.StagingTableName(),
		height,
	)
	fmt.Println("Cleanup:", cleanupQuery)

	// Output:
	// Promote: INSERT INTO blocks SELECT * FROM blocks_staging WHERE height = 1000
	// Cleanup: ALTER TABLE blocks_staging DELETE WHERE height = 1000
}

// Example_typeSafety demonstrates compile-time type safety.
func Example_typeSafety() {
	// These are compile-time safe - typos caught by compiler
	entity := entities.Blocks // ✓ Correct
	// entity := entities.Bloks  // ✗ Compile error - undefined

	// Type-safe function parameters
	processEntity(entity)

	// Can't pass arbitrary strings
	// processEntity("blocks") // ✗ Compile error - type mismatch

	// Must use FromString for runtime values
	runtimeEntity, _ := entities.FromString("blocks")
	processEntity(runtimeEntity)

	// Output:
	// Processing entity: blocks
	// Processing entity: blocks
}

// processEntity accepts only Entity type, preventing string typos.
func processEntity(entity entities.Entity) {
	fmt.Println("Processing entity:", entity)
}

// Example_switchStatement shows using entities in switch statements.
func Example_switchStatement() {
	entity := entities.Blocks

	var description string
	switch entity {
	case entities.Blocks:
		description = "Blockchain blocks"
	case entities.Transactions:
		description = "Transaction data"
	case entities.BlockSummaries:
		description = "Block aggregations"
	default:
		description = "Unknown entity"
	}

	fmt.Printf("%s: %s\n", entity, description)

	// Output:
	// blocks: Blockchain blocks
}

// Example_mapUsage demonstrates using entities as map keys.
func Example_mapUsage() {
	// Track processing status per entity
	processed := make(map[entities.Entity]bool)

	// Process each entity
	for _, entity := range entities.All() {
		// Do some work...
		processed[entity] = true
	}

	// Check status
	fmt.Println("Blocks processed:", processed[entities.Blocks])
	fmt.Println("Transactions processed:", processed[entities.Transactions])

	// Output:
	// Blocks processed: true
	// Transactions processed: true
}

// Example_loggingIntegration shows integration with structured logging.
func Example_loggingIntegration() {
	entity := entities.Blocks
	height := uint64(1000)

	// Entity implements Stringer, works great with structured logging
	// Example with zap (conceptual):
	// logger.Info("promoting data",
	//     zap.Stringer("entity", entity),
	//     zap.Uint64("height", height))

	// Or with standard formatting:
	fmt.Printf("Promoting data: entity=%s, height=%d\n", entity, height)

	// Output:
	// Promoting data: entity=blocks, height=1000
}

// Example_errorMessages demonstrates helpful error messages.
func Example_errorMessages() {
	// Try to use an invalid entity
	_, err := entities.FromString("acocunts") // Common typo

	if err != nil {
		// Error includes the invalid input and list of valid options
		fmt.Println(err)
		// This helps developers quickly identify and fix typos
	}
}

// Example_addingNewEntity demonstrates how to add a new entity.
func Example_addingNewEntity() {
	// To add a new entity (e.g., "accounts"):
	//
	// 1. Add constant in entities.go:
	//    const Accounts Entity = "accounts"
	//
	// 2. Add to allEntities slice in entities.go:
	//    var allEntities = []Entity{
	//        Blocks,
	//        Transactions,
	//        TransactionsRaw,
	//        BlockSummaries,
	//        Accounts,  // <-- Add here
	//    }
	//
	// 3. That's it! Everything else works automatically:
	//    - Accounts.TableName() returns "accounts"
	//    - Accounts.StagingTableName() returns "accounts_staging"
	//    - Accounts.IsValid() returns true
	//    - All() includes Accounts
	//    - FromString("accounts") works
	//
	// The init() function validates everything at startup.

	fmt.Println("See entities.go to add new entities")

	// Output:
	// See entities.go to add new entities
}

// Example_batchOperations demonstrates batch operations across entities.
func Example_batchOperations() {
	height := uint64(1000)

	// Promote all entities at a given height
	fmt.Printf("Promoting all entities at height %d:\n", height)
	for _, entity := range entities.All() {
		query := fmt.Sprintf(
			"INSERT INTO %s SELECT * FROM %s WHERE height = %d",
			entity.TableName(),
			entity.StagingTableName(),
			height,
		)
		fmt.Printf("  %s\n", query)
	}
}

// Example_performanceConsiderations demonstrates zero-overhead design.
func Example_performanceConsiderations() {
	// Entity operations have zero runtime overhead:
	//
	// 1. Constants compile away to their string values
	// 2. TableName() is a simple string conversion (no allocation)
	// 3. StagingTableName() is a string concatenation (single allocation)
	// 4. IsValid() is an O(1) map lookup
	//
	// No reflection, no runtime type checking, no interface overhead

	entity := entities.Blocks

	// This compiles to the same as: tableName := "blocks"
	tableName := entity.TableName()

	fmt.Println("Table:", tableName)

	// Output:
	// Table: blocks
}
