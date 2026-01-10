#!/bin/bash
# Script to add canopy-node-1 and canopy-node-2 as chains in the indexer
# Uses the admin API to register the chains for indexing

set -e

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
ADMIN_TOKEN="${ADMIN_TOKEN:-devtoken}"

# Function to add a chain
add_chain() {
    local CHAIN_ID=$1
    local CHAIN_NAME=$2
    local RPC_URL=$3
    local NOTES=$4

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
        return 0
    else
        echo "✗ Failed to add chain (HTTP $HTTP_CODE)"
        echo "Response: $BODY"
        return 1
    fi
}

# Add canopy-node-1 (Chain ID 1)
add_chain 1 "Canopy Node 1" "http://canopy-node-1:50002" "Local Canopy blockchain node 1 for development"

# Add canopy-node-2 (Chain ID 2)
add_chain 2 "Canopy Node 2" "http://canopy-node-2:50002" "Local Canopy blockchain node 2 for development"

echo "The indexer will discover these chains within 30 seconds."
echo "You can verify by checking the indexer logs:"
echo "  docker logs canopy-indexer-indexer --tail 20"
