# Entity Package

Type-safe constants for database entity names.

## What It Is

Single source of truth for entity names (blocks, transactions, accounts, etc.) used in:
- Table names (production + staging)
- Workflow loops
- Activity parameters
- Database operations

## Why Use It

**Without entities:**
```go
db.PromoteData("acocunts", height)  // Typo - runtime error
db.PromoteData("account", height)   // Wrong name - runtime error
```

**With entities:**
```go
db.PromoteData(entities.Accounts, height)  // Compile-time safe
```

## Usage

**Import:**
```go
import "github.com/canopy-network/canopy-indexer/pkg/db/entities"
```

**Use constants:**
```go
entity := entities.Blocks
table := entity.TableName()         // "blocks"
staging := entity.StagingTableName() // "blocks_staging"
```

**Iterate all entities:**
```go
for _, entity := range entities.All() {
    db.PromoteData(entity, height)
}
```

**Validate external input:**
```go
entity, err := entities.FromString("blocks")
if err != nil {
    return fmt.Errorf("invalid entity: %w", err)
}
```

## Available Entities

See `entities.go` for the complete list:
- `Blocks` - Blockchain blocks
- `Transactions` - Transaction data
- `BlockSummaries` - Block aggregations
- `Accounts` - Account balances
- `Events` - Blockchain events
- `Validators` - Validator state
- `Committees` - Committee data
- `Pools` - Pool balances
- `Orders` - DEX orders
- `Params` - Chain parameters
- And more...

## Adding a New Entity

1. Add constant to `entities.go`:
```go
const MyEntity Entity = "my_entity"
```

2. Add to `allEntities` slice:
```go
var allEntities = []Entity{
    // ...
    MyEntity,
}
```

3. Create Insert method in `pkg/db/chain/my_entity.go`

Done. Type safety enforced everywhere.
