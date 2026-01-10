#!/bin/bash
# Script to add canopy-node-1 as a chain in the indexer
# Uses the admin API to register the chain for indexing

set -e

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
ADMIN_TOKEN="${ADMIN_TOKEN:-devtoken}"

# Canopy node configuration
# Chain ID 1 is the default for the local dev Canopy network
CHAIN_ID="${CHAIN_ID:-1}"
CHAIN_NAME="${CHAIN_NAME:-Canopy Node 1}"
# Use container hostname for container-to-container access within Docker network
RPC_URL="${RPC_URL:-http://canopy-node-1:50002}"
NOTES="Local Canopy blockchain node for development"

echo "Adding Canopy node to indexer..."
echo "  API URL: $API_BASE_URL"
echo "  Chain ID: $CHAIN_ID"
echo "  Chain Name: $CHAIN_NAME"
echo "  RPC URL: $RPC_URL"
echo ""

# Create the chain via API
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/api/chains" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d "{
        \"chain_id\": $CHAIN_ID,
        \"chain_name\": \"$CHAIN_NAME\",
        \"rpc_endpoints\": [\"$RPC_URL\"],
        \"notes\": \"$NOTES\"
    }")

# Extract HTTP status code (last line)
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
# Extract response body (all but last line)
BODY=$(echo "$RESPONSE" | head -n -1)

if [ "$HTTP_CODE" -eq 200 ]; then
    echo "✓ Chain $CHAIN_ID ($CHAIN_NAME) added successfully!"
    echo ""
    echo "The indexer will discover this chain within 30 seconds."
    echo "You can verify by checking the indexer logs:"
    echo "  docker logs canopy-indexer-indexer --tail 20"
else
    echo "✗ Failed to add chain (HTTP $HTTP_CODE)"
    echo "Response: $BODY"
    exit 1
fi
