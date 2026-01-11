-- canopy-indexer schema migration
-- PostgreSQL schema for cross-chain indexing with schema-based organization
--
-- Architecture:
-- - admin schema: Shared administrative tables (chains, index_progress, etc.)
-- - chain_{chain_id} schemas: Created dynamically by the application for each chain
-- - crosschain schema: Optional global cross-chain tables (if enabled)
--
-- This migration focuses on the admin schema. Chain schemas are created
-- dynamically when chains are added via the application.

-- ============================================================================
-- SCHEMAS
-- ============================================================================

-- Admin schema for shared configuration and tracking
CREATE SCHEMA IF NOT EXISTS admin;

-- ============================================================================
-- ENUMS (created in admin schema for shared use)
-- ============================================================================

-- Validator status enum
CREATE TYPE admin.validator_status AS ENUM ('active', 'paused', 'unstaking');

-- Order status enum
CREATE TYPE admin.order_status AS ENUM ('open', 'complete', 'canceled');

-- DEX order state enum
CREATE TYPE admin.dex_order_state AS ENUM ('future', 'locked', 'complete');

-- DEX deposit state enum
CREATE TYPE admin.dex_deposit_state AS ENUM ('pending', 'locked', 'complete');

-- DEX withdrawal state enum
CREATE TYPE admin.dex_withdrawal_state AS ENUM ('pending', 'locked', 'complete');

-- Validator status enum for chain schemas
CREATE TYPE validator_status AS ENUM ('active', 'paused', 'unstaking');

-- ============================================================================
-- ADMIN TABLES
-- All admin tables are created in the admin schema
-- ============================================================================

-- Chains table: stores chain configuration and metadata
CREATE TABLE IF NOT EXISTS admin.chains (
    chain_id BIGINT PRIMARY KEY,
    chain_name TEXT NOT NULL,
    rpc_endpoints TEXT[] NOT NULL DEFAULT '{}',
    paused SMALLINT NOT NULL DEFAULT 0,
    deleted SMALLINT NOT NULL DEFAULT 0,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index progress table: tracks indexing progress per height per chain
CREATE TABLE IF NOT EXISTS admin.index_progress (
    chain_id BIGINT NOT NULL,
    height BIGINT NOT NULL,
    indexed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    indexing_time DOUBLE PRECISION NOT NULL DEFAULT 0,
    indexing_time_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    indexing_detail TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, height)
);

CREATE INDEX IF NOT EXISTS idx_admin_index_progress_chain_height ON admin.index_progress(chain_id, height);
CREATE INDEX IF NOT EXISTS idx_admin_index_progress_indexed_at ON admin.index_progress(indexed_at);

-- Reindex requests table: stores reindexing requests
CREATE TABLE IF NOT EXISTS admin.reindex_requests (
    chain_id BIGINT NOT NULL,
    from_height BIGINT NOT NULL,
    to_height BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    error TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (chain_id, from_height, to_height)
);

-- RPC endpoints table: stores RPC endpoint health status
CREATE TABLE IF NOT EXISTS admin.rpc_endpoints (
    chain_id BIGINT NOT NULL,
    endpoint TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'unknown',
    height BIGINT NOT NULL DEFAULT 0,
    latency_ms DOUBLE PRECISION NOT NULL DEFAULT 0,
    error TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, endpoint)
);

-- ============================================================================
-- NOTES
-- ============================================================================
--
-- Chain-specific tables (blocks, transactions, accounts, validators, etc.)
-- are NOT created here. They are created dynamically by the application
-- when chains are added.
--
-- Each chain gets its own schema named "chain_{chain_id}" containing:
--   - blocks
--   - block_summaries
--   - txs
--   - events
--   - accounts
--   - validators
--   - validator_non_signing_info
--   - validator_double_signing_info
--   - committees
--   - committee_validators
--   - committee_payments
--   - pools
--   - pool_points_by_holder
--   - orders
--   - dex_orders
--   - dex_deposits
--   - dex_withdrawals
--   - dex_prices
--   - params
--   - supply
--   - poll_snapshots
--   - proposal_snapshots
--
-- See pkg/db/postgres/chain/*.go for table creation SQL.
--
-- Optional crosschain schema can be created for cross-chain aggregation.
-- See pkg/db/postgres/crosschain/*.go for cross-chain table definitions.
