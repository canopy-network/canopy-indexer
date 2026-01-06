# Tiltfile for pgindexer Development
# Simple Docker Compose setup with hot-reload for Go indexer

print("🚀 Starting pgindexer Tilt Environment")

# Load docker-compose file
docker_compose('docker-compose.yml')

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
        export PGPASSWORD=pgindexer123
        export DATABASE_URL="postgres://pgindexer:pgindexer123@localhost:5434/pgindexer?sslmode=disable"

        # Check if tables exist
        TABLE_COUNT=$(psql -h localhost -p 5434 -U pgindexer -d pgindexer -tAc "SELECT COUNT(*) FROM pg_tables WHERE schemaname='public'" 2>/dev/null || echo "0")

        if [ "$TABLE_COUNT" -eq 0 ] || [ "$TABLE_COUNT" = "" ]; then
            echo "Database is empty, running migrations..."
            psql -h localhost -p 5434 -U pgindexer -d pgindexer -f migrations/001_initial.sql
            echo "Migrations complete!"
        else
            echo "Database has $TABLE_COUNT tables, skipping migrations"
        fi
    ''',
    resource_deps=['postgres'],
    labels=['database'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=True,
)

# Database reset - clean slate (uses tilt trigger to preserve log streaming)
local_resource(
    'db-reset',
    cmd='''
        echo "Stopping indexer..."
        docker stop pgindexer-indexer 2>/dev/null || true

        echo "Resetting database..."
        export PGPASSWORD=pgindexer123
        psql -h localhost -p 5434 -U pgindexer -d pgindexer -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

        echo "Running migrations..."
        psql -h localhost -p 5434 -U pgindexer -d pgindexer -f migrations/001_initial.sql

        echo "Restarting indexer via Tilt..."
        tilt trigger --port 10370 indexer-compile

        echo "Done!"
    ''',
    labels=['database'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False,
)

# Clear Redis streams
local_resource(
    'redis-clear',
    cmd='''
        echo "Clearing Redis streams..."
        docker exec pgindexer-redis redis-cli -a canopy-redis-dev FLUSHDB
        echo "Done!"
    ''',
    labels=['database'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False,
)

# =============================================================================
# Indexer Service
# =============================================================================

# Build indexer and backfill binaries locally
local_resource(
    'indexer-compile',
    cmd='''
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/indexer-linux ./cmd/indexer
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/backfill-linux ./cmd/backfill
        docker restart pgindexer-indexer 2>/dev/null || true
    ''',
    deps=['cmd/indexer', 'cmd/backfill', 'internal', 'pkg'],
    labels=['indexer'],
    resource_deps=['db-migrate', 'redis'],
)

# Recompile and run backfill (use this instead of restarting backfill dc_resource)
local_resource(
    'backfill-run',
    cmd='''
        echo "Compiling backfill..."
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/backfill-linux ./cmd/backfill
        echo "Restarting backfill container..."
        docker restart pgindexer-backfill
        echo "Done! Check backfill logs for output."
    ''',
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
    resource_deps=['indexer-compile', 'rpc-mock'],
    labels=['indexer'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=True,
)

# =============================================================================
# Utility Commands
# =============================================================================

# View index progress
local_resource(
    'show-progress',
    cmd='''
        export PGPASSWORD=pgindexer123
        psql -h localhost -p 5434 -U pgindexer -d pgindexer -c "
            SELECT
                chain_id,
                last_height,
                last_indexed_at,
                updated_at
            FROM index_progress
            ORDER BY chain_id;
        "
    ''',
    labels=['utils'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False,
)

# Show block count
local_resource(
    'show-stats',
    cmd='''
        export PGPASSWORD=pgindexer123
        psql -h localhost -p 5434 -U pgindexer -d pgindexer -c "
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
    ''',
    labels=['utils'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False,
)

# Tail indexer logs
local_resource(
    'logs',
    cmd='docker logs -f pgindexer-indexer --tail 100',
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
print("📌 Services:")
print("   • PostgreSQL: localhost:5434")
print("   • Redis: localhost:6381")
print("   • RPC Mock: 100 chains (IDs 1000-1099) on ports 60000-60099")
print("   • Backfill: indexes all 100 chains (runs automatically)")
print("   • Indexer: worker (waits for Redis jobs, WS disabled)")
print("")
print("🔧 Utility commands (trigger manually):")
print("   • backfill-run: Recompile and run backfill")
print("   • db-reset: Reset database and re-run migrations")
print("   • redis-clear: Clear Redis streams")
print("   • show-progress: View indexing progress")
print("   • show-stats: View table row counts")
print("   • logs: Tail indexer logs")
print("")
print("🎛️  Tilt UI: http://localhost:10370")
print("")
