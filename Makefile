.PHONY: help build build-linux build-backfill build-all run test clean migrate psql tilt \
	docker-build docker-build-dev fmt lint generate show-progress show-stats \
	backfill backfill-stats backfill-dry-run backfill-range backfill-fast show-gaps

.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Build targets:"
	@echo "  build, build-backfill, build-all, build-linux"
	@echo ""
	@echo "Run targets:"
	@echo "  run, test"
	@echo ""
	@echo "Database targets:"
	@echo "  migrate, psql, db-reset, show-progress, show-stats, show-gaps"
	@echo ""
	@echo "Backfill targets:"
	@echo "  backfill, backfill-stats, backfill-dry-run, backfill-range, backfill-fast"
	@echo ""
	@echo "Development targets:"
	@echo "  tilt, tilt-down, fmt, lint, generate"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build, docker-build-dev"

# Build the indexer (native)
build:
	go build -o bin/indexer ./cmd/indexer

# Build the backfill tool (native)
build-backfill:
	go build -o bin/backfill ./cmd/backfill

# Build all binaries
build-all: build build-backfill

# Build all binaries for Linux (for Docker)
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/indexer-linux ./cmd/indexer
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/backfill-linux ./cmd/backfill

# Run the indexer
run: build
	./bin/indexer

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Run database migrations (local postgres on 5434)
migrate:
	PGPASSWORD=canopy-indexer123 psql -h localhost -p 5434 -U canopy-indexer -d indexer -f migrations/001_initial.sql

# Connect to psql (interactive)
psql:
	PGPASSWORD=canopy-indexer123 psql -h localhost -p 5434 -U canopy-indexer -d indexer

# Reset database (drop and recreate)
db-reset:
	@echo "Resetting all schemas in indexer database..."
	PGPASSWORD=canopy-indexer123 psql -h localhost -p 5434 -U canopy-indexer -d indexer -c "DROP SCHEMA IF EXISTS admin CASCADE; DROP SCHEMA IF EXISTS crosschain CASCADE;"
	PGPASSWORD=canopy-indexer123 psql -h localhost -p 5434 -U canopy-indexer -d indexer -tAc "SELECT 'DROP SCHEMA ' || schema_name || ' CASCADE;' FROM information_schema.schemata WHERE schema_name LIKE 'chain_%'" | PGPASSWORD=canopy-indexer123 psql -h localhost -p 5434 -U canopy-indexer -d indexer
	$(MAKE) migrate

# Start Tilt development environment (port 10370 to avoid conflicts)
tilt:
	tilt up --port 10370

# Stop Tilt (kill server on port 10370 and tear down resources)
tilt-down:
	fuser -k 10370/tcp 2>/dev/null || true
	tilt down

# Docker build (production)
docker-build:
	docker build -t canopy-indexer:latest .

# Docker build (development - for Tilt)
docker-build-dev:
	docker build -f Dockerfile.dev -t canopy-indexer:latest .

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Lint code
lint:
	golangci-lint run

# Generate mocks (if needed)
generate:
	go generate ./...

# Show indexing progress
show-progress:
	PGPASSWORD=canopy-indexer123 psql -h localhost -p 5434 -U canopy-indexer -d indexer -c "SELECT * FROM index_progress ORDER BY chain_id;"

# Show table stats
show-stats:
	PGPASSWORD=canopy-indexer123 psql -h localhost -p 5434 -U canopy-indexer -d indexer -c "\
		SELECT 'blocks' as table_name, COUNT(*) as count FROM blocks \
		UNION ALL SELECT 'txs', COUNT(*) FROM txs \
		UNION ALL SELECT 'events', COUNT(*) FROM events \
		UNION ALL SELECT 'accounts', COUNT(*) FROM accounts \
		UNION ALL SELECT 'validators', COUNT(*) FROM validators \
		ORDER BY table_name;"

# ============================================================================
# Backfill targets
# ============================================================================

# Run backfill (index all missing blocks)
backfill: build-backfill
	./bin/backfill -once

# Show gap statistics only (no indexing)
backfill-stats: build-backfill
	./bin/backfill -stats

# Dry run backfill (report gaps without indexing)
backfill-dry-run: build-backfill
	./bin/backfill -dry-run

# Backfill a specific range (usage: make backfill-range START=1000 END=5000)
backfill-range: build-backfill
	./bin/backfill -start $(START) -end $(END)

# High-throughput backfill (50 concurrent workers)
backfill-fast: build-backfill
	./bin/backfill -concurrency 50 -batch 5000

# Show gaps in blocks table (SQL query)
show-gaps:
	PGPASSWORD=canopy-indexer123 psql -h localhost -p 5434 -U canopy-indexer -d indexer -c "\
		WITH stats AS ( \
			SELECT \
				COUNT(*) as indexed, \
				MIN(height) as min_height, \
				MAX(height) as max_height \
			FROM blocks WHERE chain_id = 1 \
		) \
		SELECT \
			indexed, \
			min_height, \
			max_height, \
			max_height - min_height + 1 as expected, \
			max_height - min_height + 1 - indexed as missing \
		FROM stats;"
