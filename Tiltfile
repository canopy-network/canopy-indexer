# Tiltfile for Canopy Development Environment
# Integrated setup with blockchain nodes, indexer, and supporting services
# Hot-reload enabled for all Go services

print("🚀 Starting Canopy Full Stack Development Environment")

# =============================================================================
# Configuration Constants
# =============================================================================

PGPASSWORD = "canopy-indexer123"
DATABASE_URL = "postgres://canopy-indexer:canopy-indexer123@localhost:5434/indexer?sslmode=disable"
PG_HOST = "localhost"
PG_PORT = "5434"
PG_USER = "canopy-indexer"
PG_DB = "indexer"

# =============================================================================
# Helper Functions
# =============================================================================

def go_build(name, path, restart_container=None):
    """Build a Go binary for Linux and optionally restart its container."""
    cmd = 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/{}-linux {}'.format(name, path)
    if restart_container:
        cmd += ' && docker restart {} 2>/dev/null || true'.format(restart_container)
    return cmd

# Load docker-compose files
docker_compose('docker-compose.yml')
docker_compose('docker-compose.blockchain.yml')

# =============================================================================
# Database Resources
# =============================================================================

# PostgreSQL
dc_resource('postgres', labels=['database'])

# Redis
dc_resource('redis', labels=['database'])

# RPC Mock service
dc_resource('rpc-mock', labels=['mock'])

# Run migrations after postgres is ready
local_resource(
    'db-migrate',
    cmd='''
        sleep 3
        export PGPASSWORD={pgpassword}
        export DATABASE_URL="{database_url}"

        # Check if admin schema tables exist
        TABLE_COUNT=$(psql -h {pg_host} -p {pg_port} -U {pg_user} -d {pg_db} -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='admin'" 2>/dev/null || echo "0")

        if [ "$TABLE_COUNT" -eq 0 ] || [ "$TABLE_COUNT" = "" ]; then
            echo "Database is empty, running migrations..."
            psql -h {pg_host} -p {pg_port} -U {pg_user} -d {pg_db} -f migrations/001_initial.sql
            echo "Migrations complete!"
        else
            echo "Database has $TABLE_COUNT tables, skipping migrations"
        fi
    '''.format(
        pgpassword=PGPASSWORD,
        database_url=DATABASE_URL,
        pg_host=PG_HOST,
        pg_port=PG_PORT,
        pg_user=PG_USER,
        pg_db=PG_DB
    ),
    resource_deps=['postgres'],
    labels=['database'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=True,
)

# Database initialization - ensure indexer database exists
local_resource(
    'db-init',
    cmd='''
        echo "Ensuring indexer database exists..."
        export PGPASSWORD={pgpassword}
        psql -h {pg_host} -p {pg_port} -U {pg_user} -d postgres -c "CREATE DATABASE \\"{pg_db}\\";" 2>/dev/null || echo "Database {pg_db} already exists"
        echo "Running initial migrations..."
        psql -h {pg_host} -p {pg_port} -U {pg_user} -d {pg_db} -f migrations/001_initial.sql 2>/dev/null || echo "Migrations may have already been run"
        echo "Database initialization complete."
    '''.format(
        pgpassword=PGPASSWORD,
        pg_host=PG_HOST,
        pg_port=PG_PORT,
        pg_user=PG_USER,
        pg_db=PG_DB,
    ),
    allow_parallel=True,
)

# Database reset - complete clean slate (drops and recreates indexer database with all schemas)
local_resource(
    'db-reset',
    cmd='''
        echo "Dropping and recreating indexer database..."
        echo "Stopping all services..."
        docker stop canopy-indexer-indexer 2>/dev/null || true
        docker stop canopy-indexer-backfill 2>/dev/null || true

        echo "Dropping and recreating database..."
        export PGPASSWORD={pgpassword}

        # Drop the indexer database (contains all schemas: admin, crosschain, chain_*)
        echo "Dropping database: {pg_db}"
        psql -h {pg_host} -p {pg_port} -U {pg_user} -d postgres -c "DROP DATABASE IF EXISTS \\"{pg_db}\\" WITH (FORCE);" 2>/dev/null || true

        # Recreate the indexer database
        echo "Creating database: {pg_db}"
        psql -h {pg_host} -p {pg_port} -U {pg_user} -d postgres -c "CREATE DATABASE \\"{pg_db}\\";"

        echo "Running migrations..."
        psql -h {pg_host} -p {pg_port} -U {pg_user} -d {pg_db} -f migrations/001_initial.sql

        echo "Restarting indexer service..."
        docker restart canopy-indexer-indexer 2>/dev/null || true

        echo "Database reset complete! Services will restart automatically."
        echo "Note: Chain schemas will be recreated dynamically when chains are added."
    '''.format(
        pgpassword=PGPASSWORD,
        pg_host=PG_HOST,
        pg_port=PG_PORT,
        pg_user=PG_USER,
        pg_db=PG_DB
    ),
    labels=['database'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False,
)

# Add canopy nodes using the add-canopy-node script
local_resource(
    'add-nodes',
    cmd='./scripts/add-canopy-node.sh',
    labels=['database'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False,
)

# Clear Redis streams
local_resource(
    'redis-clear',
    cmd='''
        echo "Clearing Redis streams..."
        docker exec canopy-indexer-redis redis-cli -a canopy-redis-dev FLUSHDB
        echo "Done!"
    ''',
    labels=['database'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False,
)

# =============================================================================
# Indexer Service
# =============================================================================

# Build indexer binary locally for hot-reload
local_resource(
    'indexer-compile',
    cmd=go_build('indexer', './cmd/indexer', 'canopy-indexer-indexer'),
    deps=['cmd/indexer', 'internal', 'pkg'],
    ignore=['**/*_test.go', '**/.git', '**/bin', '**/*.md', '**/.claude'],
    labels=['indexer'],
    resource_deps=['db-init', 'redis'],
    allow_parallel=True,
)

# Build backfill binary locally
local_resource(
    'backfill-compile',
    cmd=go_build('backfill', './cmd/backfill'),
    deps=['cmd/backfill', 'internal', 'pkg'],
    ignore=['**/*_test.go', '**/.git', '**/bin', '**/*.md', '**/.claude'],
    labels=['indexer'],
    resource_deps=['db-init'],
    allow_parallel=True,
)

# Manually trigger backfill recompile and restart
local_resource(
    'backfill-run',
    cmd='''
        echo "Restarting backfill container..."
        docker restart canopy-indexer-backfill
        echo "Done! Check backfill logs for output."
    ''',
    resource_deps=['backfill-compile'],
    labels=['indexer'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False,
)

# Indexer service (worker - waits for Redis jobs)
dc_resource(
    'indexer',
    resource_deps=['indexer-compile', 'rpc-mock'],
    labels=['indexer'],
    trigger_mode=TRIGGER_MODE_AUTO,
)

# Backfill service (runs once to index all mock blocks)
dc_resource(
    'backfill',
    resource_deps=['backfill-compile', 'rpc-mock'],
    labels=['indexer'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=True,
)

# =============================================================================
# Canopy Blockchain Nodes
# =============================================================================

# Build canopy blockchain binary locally for hot-reload
local_resource(
    'canopy-compile',
    cmd='cd ~/canopy/canopy/blob && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/canopy-linux ./cmd/main/... && docker restart canopy-node-1 canopy-node-2 2>/dev/null || true',
    deps=[
        '~/canopy/canopy/blob/cmd',
        '~/canopy/canopy/blob/bft',
        '~/canopy/canopy/blob/controller',
        '~/canopy/canopy/blob/fsm',
        '~/canopy/canopy/blob/lib',
        '~/canopy/canopy/blob/p2p',
        '~/canopy/canopy/blob/store',
    ],
    ignore=['**/*_test.go', '**/.git', '**/bin', '**/*.md', '**/node_modules', '**/.github'],
    labels=['blockchain'],
    allow_parallel=True,
)

# Canopy blockchain node 1
dc_resource(
    'canopy-node-1',
    resource_deps=['canopy-compile'],
    labels=['blockchain'],
    trigger_mode=TRIGGER_MODE_AUTO,
)

# Canopy blockchain node 2
dc_resource(
    'canopy-node-2',
    resource_deps=['canopy-compile'],
    labels=['blockchain'],
    trigger_mode=TRIGGER_MODE_AUTO,
)

# =============================================================================
# Utility Commands
# =============================================================================

# View index progress
local_resource(
    'show-progress',
    cmd='''
        export PGPASSWORD={pgpassword}
        psql -h {pg_host} -p {pg_port} -U {pg_user} -d {pg_db} -c "
            SELECT
                chain_id,
                last_height,
                last_indexed_at,
                updated_at
            FROM index_progress
            ORDER BY chain_id;
        "
    '''.format(
        pgpassword=PGPASSWORD,
        pg_host=PG_HOST,
        pg_port=PG_PORT,
        pg_user=PG_USER,
        pg_db=PG_DB
    ),
    labels=['utils'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False,
)

# Show block count
local_resource(
    'show-stats',
    cmd='''
        export PGPASSWORD={pgpassword}
        psql -h {pg_host} -p {pg_port} -U {pg_user} -d {pg_db} -c "
            SELECT
                'blocks' as table_name, COUNT(*) as count FROM blocks
            UNION ALL
            SELECT 'txs', COUNT(*) FROM txs
            UNION ALL
            SELECT 'events', COUNT(*) FROM events
            UNION ALL
            SELECT 'accounts', COUNT(*) FROM accounts
            UNION ALL
            SELECT 'validators', COUNT(*) FROM validators
            ORDER BY table_name;
        "
    '''.format(
        pgpassword=PGPASSWORD,
        pg_host=PG_HOST,
        pg_port=PG_PORT,
        pg_user=PG_USER,
        pg_db=PG_DB
    ),
    labels=['utils'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False,
)

# Tail indexer logs
local_resource(
    'logs',
    cmd='docker logs -f canopy-indexer-indexer --tail 100',
    labels=['utils'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False,
)

# =============================================================================
# Startup Message
# =============================================================================

print("")
print("✅ Tilt started!")
print("")
print("📌 Database & Infrastructure:")
print("   • PostgreSQL: {}:{}".format(PG_HOST, PG_PORT))
print("   • Redis: localhost:6381")
print("")
print("⛓️  Canopy Blockchain:")
print("   • Node 1 RPC: http://localhost:50002")
print("   • Node 1 Wallet: http://localhost:50000")
print("   • Node 1 Explorer: http://localhost:50001")
print("   • Node 2 RPC: http://localhost:40002")
print("   • Node 2 Wallet: http://localhost:40000")
print("   • Node 2 Explorer: http://localhost:40001")
print("")
print("📊 Indexer Services:")
print("   • RPC Mock: 100 chains (IDs 1000-1099) on ports 61000-61099")
print("   • Indexer: worker (waits for Redis jobs, WS disabled)")
print("   • Backfill: indexes all 100 chains (runs automatically)")
print("")
print("🔧 Utility commands (trigger manually):")
print("   • backfill-run: Restart backfill container")
print("   • db-init: Initialize indexer database and run migrations")
print("   • db-reset: DROP and RECREATE indexer database (with all schemas: admin, crosschain, chain_*)")
print("   • add-nodes: Add canopy nodes via script")
print("   • redis-clear: Clear Redis streams")
print("   • show-progress: View indexing progress")
print("   • show-stats: View table row counts")
print("   • logs: Tail indexer logs")
print("")
print("⚡ Hot-reload enabled:")
print("   • indexer-compile: Auto-rebuilds on code changes (parallel)")
print("   • backfill-compile: Auto-rebuilds on code changes (parallel)")
print("   • canopy-compile: Auto-rebuilds blockchain on code changes (parallel)")
print("")
print("🎛️  Tilt UI: http://localhost:10370")
print("")
