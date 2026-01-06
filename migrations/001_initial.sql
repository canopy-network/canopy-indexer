-- canopy-indexer schema
-- Mirrors ClickHouse canopyx_cross_chain database for PostgreSQL

-- ============================================================================
-- ENUMS
-- ============================================================================

CREATE TYPE validator_status AS ENUM ('active', 'paused', 'unstaking');
CREATE TYPE order_status AS ENUM ('open', 'complete', 'canceled');
CREATE TYPE dex_order_state AS ENUM ('future', 'locked', 'complete');
CREATE TYPE dex_deposit_state AS ENUM ('pending', 'locked', 'complete');
CREATE TYPE dex_withdrawal_state AS ENUM ('pending', 'locked', 'complete');

-- ============================================================================
-- INDEXING PROGRESS
-- ============================================================================

CREATE TABLE index_progress (
    chain_id        BIGINT PRIMARY KEY,
    last_height     BIGINT NOT NULL DEFAULT 0,
    last_indexed_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================================
-- BLOCKS
-- ============================================================================

CREATE TABLE blocks (
    chain_id            BIGINT NOT NULL,
    height              BIGINT NOT NULL,
    height_time         TIMESTAMPTZ NOT NULL,
    block_hash          TEXT NOT NULL,
    proposer_address    TEXT,
    total_txs           BIGINT NOT NULL DEFAULT 0,
    num_txs             INTEGER NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, height)
);

CREATE INDEX idx_blocks_time ON blocks (height_time);

-- ============================================================================
-- BLOCK SUMMARIES
-- ============================================================================

CREATE TABLE block_summaries (
    chain_id                                BIGINT NOT NULL,
    height                                  BIGINT NOT NULL,
    height_time                             TIMESTAMPTZ NOT NULL,
    total_transactions                      BIGINT NOT NULL DEFAULT 0,

    -- Transaction counts by type
    num_txs                                 INTEGER NOT NULL DEFAULT 0,
    num_txs_send                            INTEGER NOT NULL DEFAULT 0,
    num_txs_stake                           INTEGER NOT NULL DEFAULT 0,
    num_txs_unstake                         INTEGER NOT NULL DEFAULT 0,
    num_txs_edit_stake                      INTEGER NOT NULL DEFAULT 0,
    num_txs_start_poll                      INTEGER NOT NULL DEFAULT 0,
    num_txs_vote_poll                       INTEGER NOT NULL DEFAULT 0,
    num_txs_lock_order                      INTEGER NOT NULL DEFAULT 0,
    num_txs_close_order                     INTEGER NOT NULL DEFAULT 0,
    num_txs_unknown                         INTEGER NOT NULL DEFAULT 0,
    num_txs_pause                           INTEGER NOT NULL DEFAULT 0,
    num_txs_unpause                         INTEGER NOT NULL DEFAULT 0,
    num_txs_change_parameter                INTEGER NOT NULL DEFAULT 0,
    num_txs_dao_transfer                    INTEGER NOT NULL DEFAULT 0,
    num_txs_certificate_results             INTEGER NOT NULL DEFAULT 0,
    num_txs_subsidy                         INTEGER NOT NULL DEFAULT 0,
    num_txs_create_order                    INTEGER NOT NULL DEFAULT 0,
    num_txs_edit_order                      INTEGER NOT NULL DEFAULT 0,
    num_txs_delete_order                    INTEGER NOT NULL DEFAULT 0,
    num_txs_dex_limit_order                 INTEGER NOT NULL DEFAULT 0,
    num_txs_dex_liquidity_deposit           INTEGER NOT NULL DEFAULT 0,
    num_txs_dex_liquidity_withdraw          INTEGER NOT NULL DEFAULT 0,

    -- Account counts
    num_accounts                            INTEGER NOT NULL DEFAULT 0,
    num_accounts_new                        INTEGER NOT NULL DEFAULT 0,

    -- Event counts by type
    num_events                              INTEGER NOT NULL DEFAULT 0,
    num_events_reward                       INTEGER NOT NULL DEFAULT 0,
    num_events_slash                        INTEGER NOT NULL DEFAULT 0,
    num_events_dex_liquidity_deposit        INTEGER NOT NULL DEFAULT 0,
    num_events_dex_liquidity_withdraw       INTEGER NOT NULL DEFAULT 0,
    num_events_dex_swap                     INTEGER NOT NULL DEFAULT 0,
    num_events_order_book_swap              INTEGER NOT NULL DEFAULT 0,
    num_events_automatic_pause              INTEGER NOT NULL DEFAULT 0,
    num_events_automatic_begin_unstaking    INTEGER NOT NULL DEFAULT 0,
    num_events_automatic_finish_unstaking   INTEGER NOT NULL DEFAULT 0,

    -- Order counts
    num_orders                              INTEGER NOT NULL DEFAULT 0,
    num_orders_new                          INTEGER NOT NULL DEFAULT 0,
    num_orders_open                         INTEGER NOT NULL DEFAULT 0,
    num_orders_filled                       INTEGER NOT NULL DEFAULT 0,
    num_orders_cancelled                    INTEGER NOT NULL DEFAULT 0,

    -- Pool counts
    num_pools                               INTEGER NOT NULL DEFAULT 0,
    num_pools_new                           INTEGER NOT NULL DEFAULT 0,

    -- DEX price counts
    num_dex_prices                          INTEGER NOT NULL DEFAULT 0,

    -- DEX order counts
    num_dex_orders                          INTEGER NOT NULL DEFAULT 0,
    num_dex_orders_future                   INTEGER NOT NULL DEFAULT 0,
    num_dex_orders_locked                   INTEGER NOT NULL DEFAULT 0,
    num_dex_orders_complete                 INTEGER NOT NULL DEFAULT 0,
    num_dex_orders_success                  INTEGER NOT NULL DEFAULT 0,
    num_dex_orders_failed                   INTEGER NOT NULL DEFAULT 0,

    -- DEX deposit counts
    num_dex_deposits                        INTEGER NOT NULL DEFAULT 0,
    num_dex_deposits_pending                INTEGER NOT NULL DEFAULT 0,
    num_dex_deposits_locked                 INTEGER NOT NULL DEFAULT 0,
    num_dex_deposits_complete               INTEGER NOT NULL DEFAULT 0,

    -- DEX withdrawal counts
    num_dex_withdrawals                     INTEGER NOT NULL DEFAULT 0,
    num_dex_withdrawals_pending             INTEGER NOT NULL DEFAULT 0,
    num_dex_withdrawals_locked              INTEGER NOT NULL DEFAULT 0,
    num_dex_withdrawals_complete            INTEGER NOT NULL DEFAULT 0,

    -- Pool points holders
    num_pool_points_holders                 INTEGER NOT NULL DEFAULT 0,
    num_pool_points_holders_new             INTEGER NOT NULL DEFAULT 0,

    -- Params
    params_changed                          BOOLEAN NOT NULL DEFAULT FALSE,

    -- Validator counts
    num_validators                          INTEGER NOT NULL DEFAULT 0,
    num_validators_new                      INTEGER NOT NULL DEFAULT 0,
    num_validators_active                   INTEGER NOT NULL DEFAULT 0,
    num_validators_paused                   INTEGER NOT NULL DEFAULT 0,
    num_validators_unstaking                INTEGER NOT NULL DEFAULT 0,
    num_validator_non_signing_info          INTEGER NOT NULL DEFAULT 0,
    num_validator_non_signing_info_new      INTEGER NOT NULL DEFAULT 0,
    num_validator_double_signing_info       INTEGER NOT NULL DEFAULT 0,

    -- Committee counts
    num_committees                          INTEGER NOT NULL DEFAULT 0,
    num_committees_new                      INTEGER NOT NULL DEFAULT 0,
    num_committees_subsidized               INTEGER NOT NULL DEFAULT 0,
    num_committees_retired                  INTEGER NOT NULL DEFAULT 0,
    num_committee_validators                INTEGER NOT NULL DEFAULT 0,
    num_committee_payments                  INTEGER NOT NULL DEFAULT 0,

    -- Supply metrics
    supply_changed                          BOOLEAN NOT NULL DEFAULT FALSE,
    supply_total                            BIGINT NOT NULL DEFAULT 0,
    supply_staked                           BIGINT NOT NULL DEFAULT 0,
    supply_delegated_only                   BIGINT NOT NULL DEFAULT 0,

    created_at                              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, height)
);

-- ============================================================================
-- TRANSACTIONS
-- ============================================================================

CREATE TABLE txs (
    chain_id                BIGINT NOT NULL,
    height                  BIGINT NOT NULL,
    height_time             TIMESTAMPTZ NOT NULL,
    tx_hash                 TEXT NOT NULL,
    tx_index                INTEGER NOT NULL,
    tx_time                 TIMESTAMPTZ NOT NULL,
    created_height          BIGINT NOT NULL,
    network_id              BIGINT NOT NULL,
    message_type            TEXT NOT NULL,
    signer                  TEXT NOT NULL,
    amount                  BIGINT,
    fee                     BIGINT NOT NULL DEFAULT 0,
    memo                    TEXT,
    validator_address       TEXT,
    commission              DOUBLE PRECISION,
    tx_chain_id             BIGINT,
    sell_amount             BIGINT,
    buy_amount              BIGINT,
    liquidity_amount        BIGINT,
    liquidity_percent       BIGINT,
    order_id                TEXT,
    price                   DOUBLE PRECISION,
    param_key               TEXT,
    param_value             TEXT,
    committee_id            BIGINT,
    recipient               TEXT,
    poll_hash               TEXT,
    buyer_receive_address   TEXT,
    buyer_send_address      TEXT,
    buyer_chain_deadline    BIGINT,
    msg                     JSONB NOT NULL,
    public_key              TEXT,
    signature               TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, height, tx_hash)
);

CREATE INDEX idx_txs_signer ON txs (chain_id, signer);
CREATE INDEX idx_txs_message_type ON txs (chain_id, message_type);
CREATE INDEX idx_txs_time ON txs (height_time);

-- ============================================================================
-- EVENTS
-- ============================================================================

CREATE TABLE events (
    chain_id                BIGINT NOT NULL,
    height                  BIGINT NOT NULL,
    height_time             TIMESTAMPTZ NOT NULL,
    event_chain_id          BIGINT NOT NULL,
    address                 TEXT NOT NULL,
    reference               TEXT NOT NULL,
    event_type              TEXT NOT NULL,
    block_height            BIGINT NOT NULL,
    amount                  BIGINT,
    sold_amount             BIGINT,
    bought_amount           BIGINT,
    local_amount            BIGINT,
    remote_amount           BIGINT,
    success                 BOOLEAN,
    local_origin            BOOLEAN,
    order_id                TEXT,
    points_received         BIGINT,
    points_burned           BIGINT,
    data                    TEXT,
    seller_receive_address  TEXT,
    buyer_send_address      TEXT,
    sellers_send_address    TEXT,
    msg                     JSONB NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_pk ON events (chain_id, height, event_type, address);
CREATE INDEX idx_events_address ON events (chain_id, address);
CREATE INDEX idx_events_type ON events (chain_id, event_type);
CREATE INDEX idx_events_time ON events (height_time);

-- ============================================================================
-- ACCOUNTS
-- Snapshot-on-change pattern: only insert when balance changes
-- ============================================================================

CREATE TABLE accounts (
    chain_id        BIGINT NOT NULL,
    address         TEXT NOT NULL,
    amount          BIGINT NOT NULL DEFAULT 0,
    rewards         BIGINT NOT NULL DEFAULT 0,
    slashes         BIGINT NOT NULL DEFAULT 0,
    height          BIGINT NOT NULL,
    height_time     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, address, height)
);

CREATE INDEX idx_accounts_address ON accounts (chain_id, address);

-- View for latest account state
CREATE VIEW accounts_latest AS
SELECT DISTINCT ON (chain_id, address) *
FROM accounts
ORDER BY chain_id, address, height DESC;

-- ============================================================================
-- VALIDATORS
-- ============================================================================

CREATE TABLE validators (
    chain_id            BIGINT NOT NULL,
    address             TEXT NOT NULL,
    public_key          TEXT NOT NULL,
    net_address         TEXT NOT NULL,
    staked_amount       BIGINT NOT NULL DEFAULT 0,
    max_paused_height   BIGINT NOT NULL DEFAULT 0,
    unstaking_height    BIGINT NOT NULL DEFAULT 0,
    output              TEXT NOT NULL,
    delegate            BOOLEAN NOT NULL DEFAULT FALSE,
    compound            BOOLEAN NOT NULL DEFAULT FALSE,
    status              validator_status NOT NULL DEFAULT 'active',
    height              BIGINT NOT NULL,
    height_time         TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, address, height)
);

CREATE INDEX idx_validators_address ON validators (chain_id, address);
CREATE INDEX idx_validators_status ON validators (chain_id, status);

CREATE VIEW validators_latest AS
SELECT DISTINCT ON (chain_id, address) *
FROM validators
ORDER BY chain_id, address, height DESC;

-- ============================================================================
-- VALIDATOR NON-SIGNING INFO
-- ============================================================================

CREATE TABLE validator_non_signing_info (
    chain_id            BIGINT NOT NULL,
    address             TEXT NOT NULL,
    missed_blocks_count BIGINT NOT NULL DEFAULT 0,
    last_signed_height  BIGINT NOT NULL DEFAULT 0,
    height              BIGINT NOT NULL,
    height_time         TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, address, height)
);

-- ============================================================================
-- VALIDATOR DOUBLE-SIGNING INFO
-- ============================================================================

CREATE TABLE validator_double_signing_info (
    chain_id                BIGINT NOT NULL,
    address                 TEXT NOT NULL,
    evidence_count          BIGINT NOT NULL DEFAULT 0,
    first_evidence_height   BIGINT NOT NULL DEFAULT 0,
    last_evidence_height    BIGINT NOT NULL DEFAULT 0,
    height                  BIGINT NOT NULL,
    height_time             TIMESTAMPTZ NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, address, height)
);

-- ============================================================================
-- COMMITTEES
-- ============================================================================

CREATE TABLE committees (
    chain_id                    BIGINT NOT NULL,
    committee_chain_id          BIGINT NOT NULL,
    last_root_height_updated    BIGINT NOT NULL DEFAULT 0,
    last_chain_height_updated   BIGINT NOT NULL DEFAULT 0,
    number_of_samples           BIGINT NOT NULL DEFAULT 0,
    subsidized                  BOOLEAN NOT NULL DEFAULT FALSE,
    retired                     BOOLEAN NOT NULL DEFAULT FALSE,
    height                      BIGINT NOT NULL,
    height_time                 TIMESTAMPTZ NOT NULL,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, committee_chain_id, height)
);

CREATE VIEW committees_latest AS
SELECT DISTINCT ON (chain_id, committee_chain_id) *
FROM committees
ORDER BY chain_id, committee_chain_id, height DESC;

-- ============================================================================
-- COMMITTEE VALIDATORS
-- ============================================================================

CREATE TABLE committee_validators (
    chain_id        BIGINT NOT NULL,
    committee_id    BIGINT NOT NULL,
    address         TEXT NOT NULL,
    staked_amount   BIGINT NOT NULL DEFAULT 0,
    status          validator_status NOT NULL DEFAULT 'active',
    delegate        BOOLEAN NOT NULL DEFAULT FALSE,
    compound        BOOLEAN NOT NULL DEFAULT FALSE,
    height          BIGINT NOT NULL,
    height_time     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, committee_id, address, height)
);

-- ============================================================================
-- COMMITTEE PAYMENTS
-- ============================================================================

CREATE TABLE committee_payments (
    chain_id        BIGINT NOT NULL,
    committee_id    BIGINT NOT NULL,
    address         TEXT NOT NULL,
    percent         BIGINT NOT NULL DEFAULT 0,
    height          BIGINT NOT NULL,
    height_time     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, committee_id, address, height)
);

-- ============================================================================
-- POOLS
-- ============================================================================

CREATE TABLE pools (
    chain_id            BIGINT NOT NULL,
    pool_id             BIGINT NOT NULL,
    pool_chain_id       BIGINT NOT NULL,
    amount              BIGINT NOT NULL DEFAULT 0,
    total_points        BIGINT NOT NULL DEFAULT 0,
    lp_count            INTEGER NOT NULL DEFAULT 0,
    height              BIGINT NOT NULL,
    height_time         TIMESTAMPTZ NOT NULL,
    liquidity_pool_id   BIGINT NOT NULL DEFAULT 0,
    holding_pool_id     BIGINT NOT NULL DEFAULT 0,
    escrow_pool_id      BIGINT NOT NULL DEFAULT 0,
    reward_pool_id      BIGINT NOT NULL DEFAULT 0,
    amount_delta        BIGINT NOT NULL DEFAULT 0,
    total_points_delta  BIGINT NOT NULL DEFAULT 0,
    lp_count_delta      INTEGER NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, pool_id, height)
);

CREATE VIEW pools_latest AS
SELECT DISTINCT ON (chain_id, pool_id) *
FROM pools
ORDER BY chain_id, pool_id, height DESC;

-- ============================================================================
-- POOL POINTS BY HOLDER
-- ============================================================================

CREATE TABLE pool_points_by_holder (
    chain_id                BIGINT NOT NULL,
    address                 TEXT NOT NULL,
    pool_id                 BIGINT NOT NULL,
    committee               BIGINT NOT NULL,
    points                  BIGINT NOT NULL DEFAULT 0,
    liquidity_pool_points   BIGINT NOT NULL DEFAULT 0,
    liquidity_pool_id       BIGINT NOT NULL DEFAULT 0,
    height                  BIGINT NOT NULL,
    height_time             TIMESTAMPTZ NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, address, pool_id, height)
);

CREATE VIEW pool_points_by_holder_latest AS
SELECT DISTINCT ON (chain_id, address, pool_id) *
FROM pool_points_by_holder
ORDER BY chain_id, address, pool_id, height DESC;

-- ============================================================================
-- ORDERS (Order Book - P2P Bridge)
-- ============================================================================

CREATE TABLE orders (
    chain_id                BIGINT NOT NULL,
    order_id                TEXT NOT NULL,
    committee               BIGINT NOT NULL,
    data                    TEXT,
    amount_for_sale         BIGINT NOT NULL DEFAULT 0,
    requested_amount        BIGINT NOT NULL DEFAULT 0,
    seller_receive_address  TEXT NOT NULL,
    buyer_send_address      TEXT,
    buyer_receive_address   TEXT,
    buyer_chain_deadline    BIGINT NOT NULL DEFAULT 0,
    sellers_send_address    TEXT,
    status                  order_status NOT NULL DEFAULT 'open',
    height                  BIGINT NOT NULL,
    height_time             TIMESTAMPTZ NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, order_id, height)
);

CREATE VIEW orders_latest AS
SELECT DISTINCT ON (chain_id, order_id) *
FROM orders
ORDER BY chain_id, order_id, height DESC;

-- ============================================================================
-- DEX ORDERS (AMM Limit Orders)
-- ============================================================================

CREATE TABLE dex_orders (
    chain_id            BIGINT NOT NULL,
    order_id            TEXT NOT NULL,
    committee           BIGINT NOT NULL,
    address             TEXT NOT NULL,
    amount_for_sale     BIGINT NOT NULL DEFAULT 0,
    requested_amount    BIGINT NOT NULL DEFAULT 0,
    state               dex_order_state NOT NULL DEFAULT 'future',
    success             BOOLEAN NOT NULL DEFAULT FALSE,
    sold_amount         BIGINT NOT NULL DEFAULT 0,
    bought_amount       BIGINT NOT NULL DEFAULT 0,
    local_origin        BOOLEAN NOT NULL DEFAULT FALSE,
    locked_height       BIGINT NOT NULL DEFAULT 0,
    height              BIGINT NOT NULL,
    height_time         TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, order_id, height)
);

CREATE INDEX idx_dex_orders_address ON dex_orders (chain_id, address);
CREATE INDEX idx_dex_orders_state ON dex_orders (chain_id, state);

CREATE VIEW dex_orders_latest AS
SELECT DISTINCT ON (chain_id, order_id) *
FROM dex_orders
ORDER BY chain_id, order_id, height DESC;

-- ============================================================================
-- DEX DEPOSITS
-- ============================================================================

CREATE TABLE dex_deposits (
    chain_id            BIGINT NOT NULL,
    order_id            TEXT NOT NULL,
    committee           BIGINT NOT NULL,
    address             TEXT NOT NULL,
    amount              BIGINT NOT NULL DEFAULT 0,
    state               dex_deposit_state NOT NULL DEFAULT 'pending',
    local_origin        BOOLEAN NOT NULL DEFAULT FALSE,
    points_received     BIGINT NOT NULL DEFAULT 0,
    height              BIGINT NOT NULL,
    height_time         TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, order_id, height)
);

CREATE INDEX idx_dex_deposits_address ON dex_deposits (chain_id, address);

CREATE VIEW dex_deposits_latest AS
SELECT DISTINCT ON (chain_id, order_id) *
FROM dex_deposits
ORDER BY chain_id, order_id, height DESC;

-- ============================================================================
-- DEX WITHDRAWALS
-- ============================================================================

CREATE TABLE dex_withdrawals (
    chain_id            BIGINT NOT NULL,
    order_id            TEXT NOT NULL,
    committee           BIGINT NOT NULL,
    address             TEXT NOT NULL,
    percent             BIGINT NOT NULL DEFAULT 0,
    state               dex_withdrawal_state NOT NULL DEFAULT 'pending',
    local_amount        BIGINT NOT NULL DEFAULT 0,
    remote_amount       BIGINT NOT NULL DEFAULT 0,
    points_burned       BIGINT NOT NULL DEFAULT 0,
    height              BIGINT NOT NULL,
    height_time         TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, order_id, height)
);

CREATE INDEX idx_dex_withdrawals_address ON dex_withdrawals (chain_id, address);

CREATE VIEW dex_withdrawals_latest AS
SELECT DISTINCT ON (chain_id, order_id) *
FROM dex_withdrawals
ORDER BY chain_id, order_id, height DESC;

-- ============================================================================
-- DEX PRICES
-- ============================================================================

CREATE TABLE dex_prices (
    chain_id            BIGINT NOT NULL,
    local_chain_id      BIGINT NOT NULL,
    remote_chain_id     BIGINT NOT NULL,
    height              BIGINT NOT NULL,
    height_time         TIMESTAMPTZ NOT NULL,
    local_pool          BIGINT NOT NULL DEFAULT 0,
    remote_pool         BIGINT NOT NULL DEFAULT 0,
    price_e6            BIGINT NOT NULL DEFAULT 0,
    price_delta         BIGINT NOT NULL DEFAULT 0,
    local_pool_delta    BIGINT NOT NULL DEFAULT 0,
    remote_pool_delta   BIGINT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, local_chain_id, remote_chain_id, height)
);

CREATE INDEX idx_dex_prices_time ON dex_prices (height_time);

-- ============================================================================
-- PARAMS
-- Only inserted when parameters change
-- ============================================================================

CREATE TABLE params (
    chain_id                        BIGINT NOT NULL,
    height                          BIGINT NOT NULL,
    height_time                     TIMESTAMPTZ NOT NULL,
    block_size                      BIGINT NOT NULL DEFAULT 0,
    protocol_version                TEXT,
    root_chain_id                   BIGINT NOT NULL DEFAULT 0,
    retired                         BIGINT NOT NULL DEFAULT 0,
    unstaking_blocks                BIGINT NOT NULL DEFAULT 0,
    max_pause_blocks                BIGINT NOT NULL DEFAULT 0,
    double_sign_slash_percentage    BIGINT NOT NULL DEFAULT 0,
    non_sign_slash_percentage       BIGINT NOT NULL DEFAULT 0,
    max_non_sign                    BIGINT NOT NULL DEFAULT 0,
    non_sign_window                 BIGINT NOT NULL DEFAULT 0,
    max_committees                  BIGINT NOT NULL DEFAULT 0,
    max_committee_size              BIGINT NOT NULL DEFAULT 0,
    early_withdrawal_penalty        BIGINT NOT NULL DEFAULT 0,
    delegate_unstaking_blocks       BIGINT NOT NULL DEFAULT 0,
    minimum_order_size              BIGINT NOT NULL DEFAULT 0,
    stake_percent_for_subsidized    BIGINT NOT NULL DEFAULT 0,
    max_slash_per_committee         BIGINT NOT NULL DEFAULT 0,
    delegate_reward_percentage      BIGINT NOT NULL DEFAULT 0,
    buy_deadline_blocks             BIGINT NOT NULL DEFAULT 0,
    lock_order_fee_multiplier       BIGINT NOT NULL DEFAULT 0,
    minimum_stake_for_validators    BIGINT NOT NULL DEFAULT 0,
    minimum_stake_for_delegates     BIGINT NOT NULL DEFAULT 0,
    maximum_delegates_per_committee BIGINT NOT NULL DEFAULT 0,
    send_fee                        BIGINT NOT NULL DEFAULT 0,
    stake_fee                       BIGINT NOT NULL DEFAULT 0,
    edit_stake_fee                  BIGINT NOT NULL DEFAULT 0,
    unstake_fee                     BIGINT NOT NULL DEFAULT 0,
    pause_fee                       BIGINT NOT NULL DEFAULT 0,
    unpause_fee                     BIGINT NOT NULL DEFAULT 0,
    change_parameter_fee            BIGINT NOT NULL DEFAULT 0,
    dao_transfer_fee                BIGINT NOT NULL DEFAULT 0,
    certificate_results_fee         BIGINT NOT NULL DEFAULT 0,
    subsidy_fee                     BIGINT NOT NULL DEFAULT 0,
    create_order_fee                BIGINT NOT NULL DEFAULT 0,
    edit_order_fee                  BIGINT NOT NULL DEFAULT 0,
    delete_order_fee                BIGINT NOT NULL DEFAULT 0,
    dex_limit_order_fee             BIGINT NOT NULL DEFAULT 0,
    dex_liquidity_deposit_fee       BIGINT NOT NULL DEFAULT 0,
    dex_liquidity_withdraw_fee      BIGINT NOT NULL DEFAULT 0,
    dao_reward_percentage           BIGINT NOT NULL DEFAULT 0,
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, height)
);

CREATE VIEW params_latest AS
SELECT DISTINCT ON (chain_id) *
FROM params
ORDER BY chain_id, height DESC;

-- ============================================================================
-- SUPPLY
-- Only inserted when supply changes
-- ============================================================================

CREATE TABLE supply (
    chain_id        BIGINT NOT NULL,
    total           BIGINT NOT NULL DEFAULT 0,
    staked          BIGINT NOT NULL DEFAULT 0,
    delegated_only  BIGINT NOT NULL DEFAULT 0,
    height          BIGINT NOT NULL,
    height_time     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, height)
);

CREATE VIEW supply_latest AS
SELECT DISTINCT ON (chain_id) *
FROM supply
ORDER BY chain_id, height DESC;

-- ============================================================================
-- LP POSITION SNAPSHOTS (Real-time view derived from pool_points_by_holder)
-- ============================================================================

-- Index for efficient "latest position" lookups
CREATE INDEX idx_pph_latest
ON pool_points_by_holder (chain_id, address, pool_id, height DESC);

-- Index for first-seen aggregation
CREATE INDEX idx_pph_first_seen
ON pool_points_by_holder (chain_id, address, pool_id, height_time);

-- View: lp_position_snapshots
-- Derives LP position data from pool_points_by_holder in real-time
CREATE VIEW lp_position_snapshots AS
WITH latest_positions AS (
    -- Most recent record per holder/pool
    SELECT DISTINCT ON (chain_id, address, pool_id)
        chain_id, address, pool_id, committee,
        points, liquidity_pool_points, liquidity_pool_id,
        height, height_time
    FROM pool_points_by_holder
    ORDER BY chain_id, address, pool_id, height DESC
),
first_seen AS (
    -- Position creation date
    SELECT chain_id, address, pool_id, MIN(height_time) as created_date
    FROM pool_points_by_holder
    GROUP BY chain_id, address, pool_id
),
pool_totals AS (
    -- Current pool total_points
    SELECT DISTINCT ON (chain_id, pool_id)
        chain_id, pool_id, total_points
    FROM pools
    ORDER BY chain_id, pool_id, height DESC
)
SELECT
    lp.chain_id,
    lp.address,
    lp.pool_id,
    lp.committee,
    lp.points,
    lp.liquidity_pool_points,
    lp.liquidity_pool_id,
    CASE WHEN pt.total_points > 0
         THEN (lp.points::numeric / pt.total_points * 100 * 1e6)::bigint
         ELSE 0
    END as pool_share_ppm,  -- parts per million (divide by 1e6 for percentage)
    fs.created_date as position_created_date,
    lp.points > 0 as is_position_active,
    lp.height,
    lp.height_time
FROM latest_positions lp
JOIN first_seen fs USING (chain_id, address, pool_id)
LEFT JOIN pool_totals pt USING (chain_id, pool_id);

COMMENT ON VIEW lp_position_snapshots IS 'Real-time LP position data derived from pool_points_by_holder';

-- ============================================================================
-- TVL SNAPSHOTS (Hourly)
-- ============================================================================

CREATE TABLE tvl_snapshots (
    chain_id        BIGINT NOT NULL,
    snapshot_hour   TIMESTAMPTZ NOT NULL,
    snapshot_height BIGINT NOT NULL,
    total_staked    BIGINT NOT NULL DEFAULT 0,
    total_pooled    BIGINT NOT NULL DEFAULT 0,
    total_tvl       BIGINT NOT NULL DEFAULT 0,
    computed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (chain_id, snapshot_hour)
);

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to update index_progress
CREATE OR REPLACE FUNCTION update_index_progress(
    p_chain_id BIGINT,
    p_height BIGINT
) RETURNS VOID AS $$
BEGIN
    INSERT INTO index_progress (chain_id, last_height, last_indexed_at, updated_at)
    VALUES (p_chain_id, p_height, NOW(), NOW())
    ON CONFLICT (chain_id) DO UPDATE SET
        last_height = GREATEST(index_progress.last_height, EXCLUDED.last_height),
        last_indexed_at = NOW(),
        updated_at = NOW();
END;
$$ LANGUAGE plpgsql;

-- Build block summary from indexed data at height H
-- Called after all indexers complete for a block
CREATE OR REPLACE FUNCTION build_block_summary(
    p_chain_id BIGINT,
    p_height BIGINT,
    p_height_time TIMESTAMPTZ
) RETURNS VOID AS $$
DECLARE
    -- Transaction counts
    v_total_transactions BIGINT;
    v_num_txs INT;
    v_num_txs_send INT;
    v_num_txs_stake INT;
    v_num_txs_unstake INT;
    v_num_txs_edit_stake INT;
    v_num_txs_start_poll INT;
    v_num_txs_vote_poll INT;
    v_num_txs_lock_order INT;
    v_num_txs_close_order INT;
    v_num_txs_unknown INT;
    v_num_txs_pause INT;
    v_num_txs_unpause INT;
    v_num_txs_change_parameter INT;
    v_num_txs_dao_transfer INT;
    v_num_txs_certificate_results INT;
    v_num_txs_subsidy INT;
    v_num_txs_create_order INT;
    v_num_txs_edit_order INT;
    v_num_txs_delete_order INT;
    v_num_txs_dex_limit_order INT;
    v_num_txs_dex_liquidity_deposit INT;
    v_num_txs_dex_liquidity_withdraw INT;

    -- Account counts
    v_num_accounts INT;
    v_num_accounts_new INT;

    -- Event counts
    v_num_events INT;
    v_num_events_reward INT;
    v_num_events_slash INT;
    v_num_events_dex_liquidity_deposit INT;
    v_num_events_dex_liquidity_withdraw INT;
    v_num_events_dex_swap INT;
    v_num_events_order_book_swap INT;
    v_num_events_automatic_pause INT;
    v_num_events_automatic_begin_unstaking INT;
    v_num_events_automatic_finish_unstaking INT;

    -- Validator counts
    v_num_validators INT;
    v_num_validators_new INT;
    v_num_validators_active INT;
    v_num_validators_paused INT;
    v_num_validators_unstaking INT;
    v_num_validator_non_signing_info INT;
    v_num_validator_non_signing_info_new INT;
    v_num_validator_double_signing_info INT;

    -- Committee counts
    v_num_committees INT;
    v_num_committees_new INT;
    v_num_committees_subsidized INT;
    v_num_committees_retired INT;
    v_num_committee_validators INT;
    v_num_committee_payments INT;

    -- Pool counts
    v_num_pools INT;
    v_num_pools_new INT;
    v_num_pool_points_holders INT;
    v_num_pool_points_holders_new INT;

    -- Order counts
    v_num_orders INT;
    v_num_orders_new INT;
    v_num_orders_open INT;
    v_num_orders_filled INT;
    v_num_orders_cancelled INT;

    -- DEX counts
    v_num_dex_prices INT;
    v_num_dex_orders INT;
    v_num_dex_orders_future INT;
    v_num_dex_orders_locked INT;
    v_num_dex_orders_complete INT;
    v_num_dex_orders_success INT;
    v_num_dex_orders_failed INT;
    v_num_dex_deposits INT;
    v_num_dex_deposits_pending INT;
    v_num_dex_deposits_locked INT;
    v_num_dex_deposits_complete INT;
    v_num_dex_withdrawals INT;
    v_num_dex_withdrawals_pending INT;
    v_num_dex_withdrawals_locked INT;
    v_num_dex_withdrawals_complete INT;

    -- Params and supply
    v_params_changed BOOLEAN;
    v_supply_changed BOOLEAN;
    v_supply_total BIGINT;
    v_supply_staked BIGINT;
    v_supply_delegated_only BIGINT;
BEGIN
    -- ========================================================================
    -- TRANSACTIONS (from blocks table for total, txs table for counts by type)
    -- ========================================================================
    SELECT COALESCE(total_txs, 0) INTO v_total_transactions
    FROM blocks WHERE chain_id = p_chain_id AND height = p_height;

    SELECT
        COUNT(*),
        COUNT(*) FILTER (WHERE message_type = 'send'),
        COUNT(*) FILTER (WHERE message_type = 'stake'),
        COUNT(*) FILTER (WHERE message_type = 'unstake'),
        COUNT(*) FILTER (WHERE message_type = 'editStake'),
        COUNT(*) FILTER (WHERE message_type = 'startPoll'),
        COUNT(*) FILTER (WHERE message_type = 'votePoll'),
        COUNT(*) FILTER (WHERE message_type = 'lockOrder'),
        COUNT(*) FILTER (WHERE message_type = 'closeOrder'),
        COUNT(*) FILTER (WHERE message_type NOT IN (
            'send', 'stake', 'unstake', 'editStake', 'startPoll', 'votePoll',
            'lockOrder', 'closeOrder', 'pause', 'unpause', 'changeParameter',
            'daoTransfer', 'certificateResults', 'subsidy', 'createOrder',
            'editOrder', 'deleteOrder', 'dexLimitOrder', 'dexLiquidityDeposit',
            'dexLiquidityWithdraw'
        )),
        COUNT(*) FILTER (WHERE message_type = 'pause'),
        COUNT(*) FILTER (WHERE message_type = 'unpause'),
        COUNT(*) FILTER (WHERE message_type = 'changeParameter'),
        COUNT(*) FILTER (WHERE message_type = 'daoTransfer'),
        COUNT(*) FILTER (WHERE message_type = 'certificateResults'),
        COUNT(*) FILTER (WHERE message_type = 'subsidy'),
        COUNT(*) FILTER (WHERE message_type = 'createOrder'),
        COUNT(*) FILTER (WHERE message_type = 'editOrder'),
        COUNT(*) FILTER (WHERE message_type = 'deleteOrder'),
        COUNT(*) FILTER (WHERE message_type = 'dexLimitOrder'),
        COUNT(*) FILTER (WHERE message_type = 'dexLiquidityDeposit'),
        COUNT(*) FILTER (WHERE message_type = 'dexLiquidityWithdraw')
    INTO
        v_num_txs, v_num_txs_send, v_num_txs_stake, v_num_txs_unstake,
        v_num_txs_edit_stake, v_num_txs_start_poll, v_num_txs_vote_poll,
        v_num_txs_lock_order, v_num_txs_close_order, v_num_txs_unknown,
        v_num_txs_pause, v_num_txs_unpause, v_num_txs_change_parameter,
        v_num_txs_dao_transfer, v_num_txs_certificate_results, v_num_txs_subsidy,
        v_num_txs_create_order, v_num_txs_edit_order, v_num_txs_delete_order,
        v_num_txs_dex_limit_order, v_num_txs_dex_liquidity_deposit,
        v_num_txs_dex_liquidity_withdraw
    FROM txs WHERE chain_id = p_chain_id AND height = p_height;

    -- ========================================================================
    -- ACCOUNTS (count at H, new = at H but not at H-1)
    -- ========================================================================
    SELECT COUNT(*) INTO v_num_accounts
    FROM accounts WHERE chain_id = p_chain_id AND height = p_height;

    SELECT COUNT(*) INTO v_num_accounts_new
    FROM accounts a
    WHERE a.chain_id = p_chain_id AND a.height = p_height
      AND NOT EXISTS (
          SELECT 1 FROM accounts prev
          WHERE prev.chain_id = p_chain_id
            AND prev.height = p_height - 1
            AND prev.address = a.address
      );

    -- ========================================================================
    -- EVENTS
    -- ========================================================================
    SELECT
        COUNT(*),
        COUNT(*) FILTER (WHERE event_type = 'reward'),
        COUNT(*) FILTER (WHERE event_type = 'slash'),
        COUNT(*) FILTER (WHERE event_type = 'dex-liquidity-deposit'),
        COUNT(*) FILTER (WHERE event_type = 'dex-liquidity-withdraw'),
        COUNT(*) FILTER (WHERE event_type = 'dex-swap'),
        COUNT(*) FILTER (WHERE event_type = 'order-book-swap'),
        COUNT(*) FILTER (WHERE event_type = 'automatic-pause'),
        COUNT(*) FILTER (WHERE event_type = 'automatic-begin-unstaking'),
        COUNT(*) FILTER (WHERE event_type = 'automatic-finish-unstaking')
    INTO
        v_num_events, v_num_events_reward, v_num_events_slash,
        v_num_events_dex_liquidity_deposit, v_num_events_dex_liquidity_withdraw,
        v_num_events_dex_swap, v_num_events_order_book_swap,
        v_num_events_automatic_pause, v_num_events_automatic_begin_unstaking,
        v_num_events_automatic_finish_unstaking
    FROM events WHERE chain_id = p_chain_id AND height = p_height;

    -- ========================================================================
    -- VALIDATORS
    -- ========================================================================
    SELECT
        COUNT(*),
        COUNT(*) FILTER (WHERE status = 'active'),
        COUNT(*) FILTER (WHERE status = 'paused'),
        COUNT(*) FILTER (WHERE status = 'unstaking')
    INTO v_num_validators, v_num_validators_active, v_num_validators_paused, v_num_validators_unstaking
    FROM validators WHERE chain_id = p_chain_id AND height = p_height;

    SELECT COUNT(*) INTO v_num_validators_new
    FROM validators v
    WHERE v.chain_id = p_chain_id AND v.height = p_height
      AND NOT EXISTS (
          SELECT 1 FROM validators prev
          WHERE prev.chain_id = p_chain_id
            AND prev.height = p_height - 1
            AND prev.address = v.address
      );

    -- Validator non-signing info
    SELECT COUNT(*) INTO v_num_validator_non_signing_info
    FROM validator_non_signing_info WHERE chain_id = p_chain_id AND height = p_height;

    SELECT COUNT(*) INTO v_num_validator_non_signing_info_new
    FROM validator_non_signing_info v
    WHERE v.chain_id = p_chain_id AND v.height = p_height
      AND NOT EXISTS (
          SELECT 1 FROM validator_non_signing_info prev
          WHERE prev.chain_id = p_chain_id
            AND prev.height = p_height - 1
            AND prev.address = v.address
      );

    -- Validator double-signing info
    SELECT COUNT(*) INTO v_num_validator_double_signing_info
    FROM validator_double_signing_info WHERE chain_id = p_chain_id AND height = p_height;

    -- ========================================================================
    -- COMMITTEES
    -- ========================================================================
    SELECT
        COUNT(*),
        COUNT(*) FILTER (WHERE subsidized = true),
        COUNT(*) FILTER (WHERE retired = true)
    INTO v_num_committees, v_num_committees_subsidized, v_num_committees_retired
    FROM committees WHERE chain_id = p_chain_id AND height = p_height;

    SELECT COUNT(*) INTO v_num_committees_new
    FROM committees c
    WHERE c.chain_id = p_chain_id AND c.height = p_height
      AND NOT EXISTS (
          SELECT 1 FROM committees prev
          WHERE prev.chain_id = p_chain_id
            AND prev.height = p_height - 1
            AND prev.committee_chain_id = c.committee_chain_id
      );

    -- Committee validators
    SELECT COUNT(*) INTO v_num_committee_validators
    FROM committee_validators WHERE chain_id = p_chain_id AND height = p_height;

    -- Committee payments
    SELECT COUNT(*) INTO v_num_committee_payments
    FROM committee_payments WHERE chain_id = p_chain_id AND height = p_height;

    -- ========================================================================
    -- POOLS
    -- ========================================================================
    SELECT COUNT(*) INTO v_num_pools
    FROM pools WHERE chain_id = p_chain_id AND height = p_height;

    SELECT COUNT(*) INTO v_num_pools_new
    FROM pools p
    WHERE p.chain_id = p_chain_id AND p.height = p_height
      AND NOT EXISTS (
          SELECT 1 FROM pools prev
          WHERE prev.chain_id = p_chain_id
            AND prev.height = p_height - 1
            AND prev.pool_id = p.pool_id
      );

    -- Pool points holders
    SELECT COUNT(*) INTO v_num_pool_points_holders
    FROM pool_points_by_holder WHERE chain_id = p_chain_id AND height = p_height;

    SELECT COUNT(*) INTO v_num_pool_points_holders_new
    FROM pool_points_by_holder h
    WHERE h.chain_id = p_chain_id AND h.height = p_height
      AND NOT EXISTS (
          SELECT 1 FROM pool_points_by_holder prev
          WHERE prev.chain_id = p_chain_id
            AND prev.height = p_height - 1
            AND prev.address = h.address
            AND prev.pool_id = h.pool_id
      );

    -- ========================================================================
    -- ORDERS (order book) - enum values: open, complete, canceled
    -- ========================================================================
    SELECT
        COUNT(*),
        COUNT(*) FILTER (WHERE status = 'open'),
        COUNT(*) FILTER (WHERE status = 'complete'),
        COUNT(*) FILTER (WHERE status = 'canceled')
    INTO v_num_orders, v_num_orders_open, v_num_orders_filled, v_num_orders_cancelled
    FROM orders WHERE chain_id = p_chain_id AND height = p_height;

    SELECT COUNT(*) INTO v_num_orders_new
    FROM orders o
    WHERE o.chain_id = p_chain_id AND o.height = p_height
      AND NOT EXISTS (
          SELECT 1 FROM orders prev
          WHERE prev.chain_id = p_chain_id
            AND prev.height = p_height - 1
            AND prev.order_id = o.order_id
      );

    -- ========================================================================
    -- DEX PRICES
    -- ========================================================================
    SELECT COUNT(*) INTO v_num_dex_prices
    FROM dex_prices WHERE chain_id = p_chain_id AND height = p_height;

    -- ========================================================================
    -- DEX ORDERS
    -- ========================================================================
    SELECT
        COUNT(*),
        COUNT(*) FILTER (WHERE state = 'future'),
        COUNT(*) FILTER (WHERE state = 'locked'),
        COUNT(*) FILTER (WHERE state = 'complete'),
        COUNT(*) FILTER (WHERE state = 'complete' AND success = true),
        COUNT(*) FILTER (WHERE state = 'complete' AND success = false)
    INTO
        v_num_dex_orders, v_num_dex_orders_future, v_num_dex_orders_locked,
        v_num_dex_orders_complete, v_num_dex_orders_success, v_num_dex_orders_failed
    FROM dex_orders WHERE chain_id = p_chain_id AND height = p_height;

    -- ========================================================================
    -- DEX DEPOSITS
    -- ========================================================================
    SELECT
        COUNT(*),
        COUNT(*) FILTER (WHERE state = 'pending'),
        COUNT(*) FILTER (WHERE state = 'locked'),
        COUNT(*) FILTER (WHERE state = 'complete')
    INTO v_num_dex_deposits, v_num_dex_deposits_pending, v_num_dex_deposits_locked, v_num_dex_deposits_complete
    FROM dex_deposits WHERE chain_id = p_chain_id AND height = p_height;

    -- ========================================================================
    -- DEX WITHDRAWALS
    -- ========================================================================
    SELECT
        COUNT(*),
        COUNT(*) FILTER (WHERE state = 'pending'),
        COUNT(*) FILTER (WHERE state = 'locked'),
        COUNT(*) FILTER (WHERE state = 'complete')
    INTO v_num_dex_withdrawals, v_num_dex_withdrawals_pending, v_num_dex_withdrawals_locked, v_num_dex_withdrawals_complete
    FROM dex_withdrawals WHERE chain_id = p_chain_id AND height = p_height;

    -- ========================================================================
    -- PARAMS (check if row exists at H - means params changed)
    -- ========================================================================
    SELECT EXISTS(
        SELECT 1 FROM params WHERE chain_id = p_chain_id AND height = p_height
    ) INTO v_params_changed;

    -- ========================================================================
    -- SUPPLY (check if row exists at H, get values)
    -- ========================================================================
    SELECT
        true, total, staked, delegated_only
    INTO v_supply_changed, v_supply_total, v_supply_staked, v_supply_delegated_only
    FROM supply WHERE chain_id = p_chain_id AND height = p_height;

    IF NOT FOUND THEN
        v_supply_changed := false;
        v_supply_total := 0;
        v_supply_staked := 0;
        v_supply_delegated_only := 0;
    END IF;

    -- ========================================================================
    -- INSERT BLOCK SUMMARY
    -- ========================================================================
    INSERT INTO block_summaries (
        chain_id, height, height_time, total_transactions,
        -- Transactions
        num_txs, num_txs_send, num_txs_stake, num_txs_unstake, num_txs_edit_stake,
        num_txs_start_poll, num_txs_vote_poll, num_txs_lock_order, num_txs_close_order,
        num_txs_unknown, num_txs_pause, num_txs_unpause, num_txs_change_parameter,
        num_txs_dao_transfer, num_txs_certificate_results, num_txs_subsidy,
        num_txs_create_order, num_txs_edit_order, num_txs_delete_order,
        num_txs_dex_limit_order, num_txs_dex_liquidity_deposit, num_txs_dex_liquidity_withdraw,
        -- Accounts
        num_accounts, num_accounts_new,
        -- Events
        num_events, num_events_reward, num_events_slash,
        num_events_dex_liquidity_deposit, num_events_dex_liquidity_withdraw,
        num_events_dex_swap, num_events_order_book_swap, num_events_automatic_pause,
        num_events_automatic_begin_unstaking, num_events_automatic_finish_unstaking,
        -- Orders
        num_orders, num_orders_new, num_orders_open, num_orders_filled, num_orders_cancelled,
        -- Pools
        num_pools, num_pools_new,
        -- DEX prices
        num_dex_prices,
        -- DEX orders
        num_dex_orders, num_dex_orders_future, num_dex_orders_locked,
        num_dex_orders_complete, num_dex_orders_success, num_dex_orders_failed,
        -- DEX deposits
        num_dex_deposits, num_dex_deposits_pending, num_dex_deposits_locked, num_dex_deposits_complete,
        -- DEX withdrawals
        num_dex_withdrawals, num_dex_withdrawals_pending, num_dex_withdrawals_locked, num_dex_withdrawals_complete,
        -- Pool points holders
        num_pool_points_holders, num_pool_points_holders_new,
        -- Params
        params_changed,
        -- Validators
        num_validators, num_validators_new, num_validators_active, num_validators_paused, num_validators_unstaking,
        num_validator_non_signing_info, num_validator_non_signing_info_new, num_validator_double_signing_info,
        -- Committees
        num_committees, num_committees_new, num_committees_subsidized, num_committees_retired,
        num_committee_validators, num_committee_payments,
        -- Supply
        supply_changed, supply_total, supply_staked, supply_delegated_only
    ) VALUES (
        p_chain_id, p_height, p_height_time, COALESCE(v_total_transactions, 0),
        -- Transactions
        COALESCE(v_num_txs, 0), COALESCE(v_num_txs_send, 0), COALESCE(v_num_txs_stake, 0),
        COALESCE(v_num_txs_unstake, 0), COALESCE(v_num_txs_edit_stake, 0),
        COALESCE(v_num_txs_start_poll, 0), COALESCE(v_num_txs_vote_poll, 0),
        COALESCE(v_num_txs_lock_order, 0), COALESCE(v_num_txs_close_order, 0),
        COALESCE(v_num_txs_unknown, 0), COALESCE(v_num_txs_pause, 0), COALESCE(v_num_txs_unpause, 0),
        COALESCE(v_num_txs_change_parameter, 0), COALESCE(v_num_txs_dao_transfer, 0),
        COALESCE(v_num_txs_certificate_results, 0), COALESCE(v_num_txs_subsidy, 0),
        COALESCE(v_num_txs_create_order, 0), COALESCE(v_num_txs_edit_order, 0),
        COALESCE(v_num_txs_delete_order, 0), COALESCE(v_num_txs_dex_limit_order, 0),
        COALESCE(v_num_txs_dex_liquidity_deposit, 0), COALESCE(v_num_txs_dex_liquidity_withdraw, 0),
        -- Accounts
        COALESCE(v_num_accounts, 0), COALESCE(v_num_accounts_new, 0),
        -- Events
        COALESCE(v_num_events, 0), COALESCE(v_num_events_reward, 0), COALESCE(v_num_events_slash, 0),
        COALESCE(v_num_events_dex_liquidity_deposit, 0), COALESCE(v_num_events_dex_liquidity_withdraw, 0),
        COALESCE(v_num_events_dex_swap, 0), COALESCE(v_num_events_order_book_swap, 0),
        COALESCE(v_num_events_automatic_pause, 0), COALESCE(v_num_events_automatic_begin_unstaking, 0),
        COALESCE(v_num_events_automatic_finish_unstaking, 0),
        -- Orders
        COALESCE(v_num_orders, 0), COALESCE(v_num_orders_new, 0), COALESCE(v_num_orders_open, 0),
        COALESCE(v_num_orders_filled, 0), COALESCE(v_num_orders_cancelled, 0),
        -- Pools
        COALESCE(v_num_pools, 0), COALESCE(v_num_pools_new, 0),
        -- DEX prices
        COALESCE(v_num_dex_prices, 0),
        -- DEX orders
        COALESCE(v_num_dex_orders, 0), COALESCE(v_num_dex_orders_future, 0),
        COALESCE(v_num_dex_orders_locked, 0), COALESCE(v_num_dex_orders_complete, 0),
        COALESCE(v_num_dex_orders_success, 0), COALESCE(v_num_dex_orders_failed, 0),
        -- DEX deposits
        COALESCE(v_num_dex_deposits, 0), COALESCE(v_num_dex_deposits_pending, 0),
        COALESCE(v_num_dex_deposits_locked, 0), COALESCE(v_num_dex_deposits_complete, 0),
        -- DEX withdrawals
        COALESCE(v_num_dex_withdrawals, 0), COALESCE(v_num_dex_withdrawals_pending, 0),
        COALESCE(v_num_dex_withdrawals_locked, 0), COALESCE(v_num_dex_withdrawals_complete, 0),
        -- Pool points holders
        COALESCE(v_num_pool_points_holders, 0), COALESCE(v_num_pool_points_holders_new, 0),
        -- Params
        COALESCE(v_params_changed, false),
        -- Validators
        COALESCE(v_num_validators, 0), COALESCE(v_num_validators_new, 0),
        COALESCE(v_num_validators_active, 0), COALESCE(v_num_validators_paused, 0),
        COALESCE(v_num_validators_unstaking, 0), COALESCE(v_num_validator_non_signing_info, 0),
        COALESCE(v_num_validator_non_signing_info_new, 0), COALESCE(v_num_validator_double_signing_info, 0),
        -- Committees
        COALESCE(v_num_committees, 0), COALESCE(v_num_committees_new, 0),
        COALESCE(v_num_committees_subsidized, 0), COALESCE(v_num_committees_retired, 0),
        COALESCE(v_num_committee_validators, 0), COALESCE(v_num_committee_payments, 0),
        -- Supply
        COALESCE(v_supply_changed, false), COALESCE(v_supply_total, 0),
        COALESCE(v_supply_staked, 0), COALESCE(v_supply_delegated_only, 0)
    )
    ON CONFLICT (chain_id, height) DO UPDATE SET
        total_transactions = EXCLUDED.total_transactions,
        num_txs = EXCLUDED.num_txs,
        num_accounts = EXCLUDED.num_accounts,
        num_events = EXCLUDED.num_events,
        num_validators = EXCLUDED.num_validators,
        num_committees = EXCLUDED.num_committees,
        num_pools = EXCLUDED.num_pools,
        supply_total = EXCLUDED.supply_total;
END;
$$ LANGUAGE plpgsql;
