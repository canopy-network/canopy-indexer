# Schema Update Example: Transactions Table

This document shows the before/after of updating a Postgres table schema to match the actual ClickHouse model.

## Before: Simplified Schema (INCORRECT)

**File**: `transaction.go` (original)
**Fields**: 13 fields
**Problem**: Missing 20+ fields needed by the application

```sql
CREATE TABLE IF NOT EXISTS txs (
    height BIGINT NOT NULL,
    tx_hash TEXT NOT NULL,
    tx_index INTEGER NOT NULL,
    message_type TEXT NOT NULL,
    signer TEXT NOT NULL,
    recipient TEXT,
    amount BIGINT,
    fee BIGINT NOT NULL DEFAULT 0,
    memo TEXT,
    msg TEXT NOT NULL,
    public_key TEXT,
    signature TEXT,
    height_time TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (height, tx_hash)
);
```

### Issues with Simplified Schema

1. **Missing validator fields**: `validator_address`, `commission` (needed for stake transactions)
2. **Missing DEX fields**: `chain_id`, `sell_amount`, `buy_amount`, `liquidity_amount`, `liquidity_percent`
3. **Missing order fields**: `order_id`, `price`
4. **Missing governance fields**: `param_key`, `param_value`
5. **Missing other fields**: `committee_id`, `recipient`, `poll_hash`, `buyer_receive_address`, etc.
6. **Missing time fields**: `time`, `created_height`, `network_id`
7. **Wrong types**: `tx_index` should be SMALLINT (UInt16), not INTEGER

## After: Complete Schema (CORRECT)

**File**: `transaction_updated.go`
**Fields**: 33 fields
**Matches**: `pkg/db/models/indexer/tx.go:52-113`

```sql
CREATE TABLE IF NOT EXISTS txs (
    -- Primary key fields
    height BIGINT NOT NULL,
    tx_hash TEXT NOT NULL,
    tx_index SMALLINT NOT NULL,  -- UInt16 -> SMALLINT

    -- Time fields for queries
    time TIMESTAMP WITH TIME ZONE NOT NULL,       -- Transaction timestamp
    height_time TIMESTAMP WITH TIME ZONE NOT NULL, -- Block timestamp
    created_height BIGINT NOT NULL,
    network_id INTEGER NOT NULL,

    -- Common filterable fields
    message_type TEXT NOT NULL,
    signer TEXT NOT NULL,
    amount BIGINT DEFAULT 0,
    fee BIGINT NOT NULL,
    memo TEXT DEFAULT '',

    -- Validator-related (stake, unstake, editStake)
    validator_address TEXT DEFAULT '',
    commission DOUBLE PRECISION DEFAULT 0,

    -- DEX-related (dexLimitOrder, dexLiquidityDeposit, dexLiquidityWithdraw)
    chain_id SMALLINT DEFAULT 0,
    sell_amount BIGINT DEFAULT 0,
    buy_amount BIGINT DEFAULT 0,
    liquidity_amount BIGINT DEFAULT 0,
    liquidity_percent BIGINT DEFAULT 0,

    -- Order-related (createOrder, editOrder, deleteOrder)
    order_id TEXT DEFAULT '',
    price DOUBLE PRECISION DEFAULT 0,

    -- Governance-related (changeParameter, startPoll, votePoll)
    param_key TEXT DEFAULT '',
    param_value TEXT DEFAULT '',

    -- Other
    committee_id SMALLINT DEFAULT 0,
    recipient TEXT DEFAULT '',
    poll_hash TEXT DEFAULT '',

    -- LockOrder-related
    buyer_receive_address TEXT DEFAULT '',
    buyer_send_address TEXT DEFAULT '',
    buyer_chain_deadline BIGINT DEFAULT 0,

    -- Full message as JSON
    msg TEXT NOT NULL,

    -- Signature fields
    public_key TEXT DEFAULT '',
    signature TEXT DEFAULT '',

    PRIMARY KEY (height, tx_hash)
);
```

## Key Changes Explained

### 1. Type Corrections
```diff
- tx_index INTEGER NOT NULL
+ tx_index SMALLINT NOT NULL  -- Matches ClickHouse UInt16
```

### 2. Added Time Fields
```diff
+ time TIMESTAMP WITH TIME ZONE NOT NULL,       -- Tx timestamp
+ height_time TIMESTAMP WITH TIME ZONE NOT NULL, -- Block timestamp
+ created_height BIGINT NOT NULL,                -- Creation height
+ network_id INTEGER NOT NULL,                   -- Network ID
```

### 3. Added Validator Fields
```diff
+ validator_address TEXT DEFAULT '',  -- For stake/unstake/editStake
+ commission DOUBLE PRECISION DEFAULT 0,  -- Validator commission
```

### 4. Added DEX Fields
```diff
+ chain_id SMALLINT DEFAULT 0,       -- Target chain for DEX operations
+ sell_amount BIGINT DEFAULT 0,      -- Amount to sell
+ buy_amount BIGINT DEFAULT 0,       -- Amount to buy
+ liquidity_amount BIGINT DEFAULT 0,   -- Liquidity deposit amount
+ liquidity_percent BIGINT DEFAULT 0,  -- Liquidity withdrawal percent
```

### 5. Added Order Fields
```diff
+ order_id TEXT DEFAULT '',          -- Order book order ID
+ price DOUBLE PRECISION DEFAULT 0,  -- Order price
```

### 6. Added Governance Fields
```diff
+ param_key TEXT DEFAULT '',    -- Parameter being changed
+ param_value TEXT DEFAULT '',  -- New parameter value
```

### 7. Added Other Transaction-Type Fields
```diff
+ committee_id SMALLINT DEFAULT 0,      -- For subsidy transactions
+ recipient TEXT DEFAULT '',            -- For daoTransfer
+ poll_hash TEXT DEFAULT '',            -- For poll transactions
+ buyer_receive_address TEXT DEFAULT '', -- For lockOrder
+ buyer_send_address TEXT DEFAULT '',    -- For lockOrder
+ buyer_chain_deadline BIGINT DEFAULT 0, -- For lockOrder
```

### 8. Improved Indexes
```diff
  CREATE INDEX IF NOT EXISTS idx_txs_signer ON txs(signer);
- CREATE INDEX IF NOT EXISTS idx_txs_recipient ON txs(recipient);
+ CREATE INDEX IF NOT EXISTS idx_txs_recipient ON txs(recipient) WHERE recipient != '';
  CREATE INDEX IF NOT EXISTS idx_txs_message_type ON txs(message_type);
  CREATE INDEX IF NOT EXISTS idx_txs_time ON txs(height_time);
+ CREATE INDEX IF NOT EXISTS idx_txs_validator ON txs(validator_address) WHERE validator_address != '';
+ CREATE INDEX IF NOT EXISTS idx_txs_order ON txs(order_id) WHERE order_id != '';
+ CREATE INDEX IF NOT EXISTS idx_txs_poll ON txs(poll_hash) WHERE poll_hash != '';
```

**Explanation**: Partial indexes (with `WHERE`) are more efficient because they only index rows where the field is actually used, reducing index size.

## How to Apply This Pattern to Other Tables

### Step 1: Find the ClickHouse Model

```bash
# Look in pkg/db/models/indexer/ for the model definition
ls pkg/db/models/indexer/
```

For example:
- `block.go` → `Block` struct
- `event.go` → `Event` struct
- `account.go` → `Account` struct
- `validator.go` → `Validator` struct

### Step 2: Read the Model Struct

```go
// Example from pkg/db/models/indexer/tx.go
type Transaction struct {
    Height  uint64    `ch:"height" json:"height"`
    TxHash  string    `ch:"tx_hash" json:"tx_hash"`
    TxIndex uint16    `ch:"tx_index" json:"tx_index"`
    // ... all other fields
}
```

### Step 3: Map Each Field to Postgres Type

| Go Type | ClickHouse Type | Postgres Type | Notes |
|---------|----------------|---------------|-------|
| `uint64` | `UInt64` | `BIGINT` | |
| `uint32` | `UInt32` | `INTEGER` | |
| `uint16` | `UInt16` | `SMALLINT` | |
| `uint8` | `UInt8` | `SMALLINT` | Postgres has no TINYINT |
| `string` | `String` | `TEXT` | |
| `string` | `LowCardinality(String)` | `TEXT` | Consider ENUM if limited values |
| `time.Time` | `DateTime64(6)` | `TIMESTAMP WITH TIME ZONE` | Microsecond precision |
| `bool` | `Bool` | `BOOLEAN` | |
| `float64` | `Float64` | `DOUBLE PRECISION` | |
| `int32` | `Int32` | `INTEGER` | |

### Step 4: Add DEFAULT Values

For optional fields, add `DEFAULT` values to match ClickHouse behavior:

```sql
-- Instead of NULL for missing values, use defaults
validator_address TEXT DEFAULT '',       -- Empty string default
amount BIGINT DEFAULT 0,                 -- Zero default
success BOOLEAN DEFAULT false,           -- False default
```

This matches ClickHouse's approach: "Uses non-Nullable types with defaults to avoid UInt8 null-mask overhead."

### Step 5: Create Appropriate Indexes

Index fields that are commonly used in WHERE clauses:

```sql
-- Index filterable fields
CREATE INDEX idx_txs_message_type ON txs(message_type);
CREATE INDEX idx_txs_signer ON txs(signer);

-- Partial index for optional fields (more efficient)
CREATE INDEX idx_txs_validator ON txs(validator_address) WHERE validator_address != '';
```

## Example: Updating Events Table

Let's see how this applies to the events table:

### Current Simplified Schema
```sql
CREATE TABLE events (
    height BIGINT,
    event_type TEXT,
    event_index INTEGER,
    address TEXT,
    amount BIGINT,
    data TEXT,
    time TIMESTAMP WITH TIME ZONE,
    msg JSONB
);
```

### Model to Match: `pkg/db/models/indexer/event.go`
```go
type Event struct {
    Height       uint64 `ch:"height"`
    ChainID      uint16 `ch:"chain_id"`
    Address      string `ch:"address"`
    Reference    string `ch:"reference"`
    EventType    string `ch:"event_type"`
    BlockHeight  uint64 `ch:"block_height"`
    Amount       uint64 `ch:"amount"`
    SoldAmount   uint64 `ch:"sold_amount"`
    BoughtAmount uint64 `ch:"bought_amount"`
    LocalAmount  uint64 `ch:"local_amount"`
    RemoteAmount uint64 `ch:"remote_amount"`
    Success      bool   `ch:"success"`
    LocalOrigin  bool   `ch:"local_origin"`
    OrderID      string `ch:"order_id"`
    PointsReceived uint64 `ch:"points_received"`
    PointsBurned   uint64 `ch:"points_burned"`
    Data         string `ch:"data"`
    SellerReceiveAddress string `ch:"seller_receive_address"`
    BuyerSendAddress     string `ch:"buyer_send_address"`
    SellersSendAddress   string `ch:"sellers_send_address"`
    Msg          string `ch:"msg"`
    HeightTime   time.Time `ch:"height_time"`
}
```

### Updated Complete Schema
```sql
CREATE TABLE events (
    -- Primary key components
    height BIGINT NOT NULL,
    chain_id SMALLINT NOT NULL,              -- NEW
    address TEXT NOT NULL,
    reference TEXT NOT NULL,                  -- NEW: "begin_block", tx hash, or "end_block"
    event_type TEXT NOT NULL,
    block_height BIGINT NOT NULL,             -- NEW

    -- Extracted queryable fields with defaults
    amount BIGINT DEFAULT 0,
    sold_amount BIGINT DEFAULT 0,             -- NEW
    bought_amount BIGINT DEFAULT 0,           -- NEW
    local_amount BIGINT DEFAULT 0,            -- NEW
    remote_amount BIGINT DEFAULT 0,           -- NEW
    success BOOLEAN DEFAULT false,            -- NEW
    local_origin BOOLEAN DEFAULT false,       -- NEW
    order_id TEXT DEFAULT '',                 -- NEW
    points_received BIGINT DEFAULT 0,         -- NEW
    points_burned BIGINT DEFAULT 0,           -- NEW
    data TEXT DEFAULT '',
    seller_receive_address TEXT DEFAULT '',   -- NEW
    buyer_send_address TEXT DEFAULT '',       -- NEW
    sellers_send_address TEXT DEFAULT '',     -- NEW

    -- Full message and timestamp
    msg TEXT NOT NULL,
    height_time TIMESTAMP WITH TIME ZONE NOT NULL,

    PRIMARY KEY (height, chain_id, address, reference, event_type)
);

CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_address ON events(address);
CREATE INDEX idx_events_time ON events(height_time);
CREATE INDEX idx_events_order ON events(order_id) WHERE order_id != '';
```

## Summary

**Before**: 8 fields, missing critical data
**After**: 21 fields, matches ClickHouse model completely

The updated schema now supports all transaction types without losing data in the generic `msg` field.
