#!/bin/bash
# Script to create default chains via the admin API
# Creates 100 mock chains (IDs 1000-1099) and 2 real Canopy nodes (IDs 1 and 2)

set -e

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
ADMIN_TOKEN="${ADMIN_TOKEN:-devtoken}"
RPC_MOCK_HOST="${RPC_MOCK_HOST:-rpc-mock}"
START_CHAIN_ID="${START_CHAIN_ID:-1000}"
NUM_CHAINS="${NUM_CHAINS:-100}"

echo "Creating chains..."
echo "API URL: $API_BASE_URL"
echo "RPC Mock Host: $RPC_MOCK_HOST"

# Wait for API to be ready
echo "Waiting for API to be ready..."
for i in {1..30}; do
    if curl -s -f "$API_BASE_URL/api/health" > /dev/null 2>&1; then
        echo "API is ready"
        break
    fi
    echo "Waiting for API... ($i/30)"
    sleep 2
done

if ! curl -s -f "$API_BASE_URL/api/health" > /dev/null 2>&1; then
    echo "API is not ready after 60 seconds, exiting"
    exit 1
fi

# Function to create a chain via API
create_chain() {
    local CHAIN_ID="$1"
    local CHAIN_NAME="$2"
    local RPC_ENDPOINTS="$3"
    local NOTES="$4"

    echo "Creating chain $CHAIN_ID ($CHAIN_NAME)..."

    # Create the chain via API
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$API_BASE_URL/api/chains" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $ADMIN_TOKEN" \
        -d "{
            \"chain_id\": $CHAIN_ID,
            \"chain_name\": \"$CHAIN_NAME\",
            \"rpc_endpoints\": [$RPC_ENDPOINTS],
            \"notes\": \"$NOTES\"
        }")

    # Extract HTTP status code (last line)
    HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
    # Extract response body (all but last line)
    BODY=$(echo "$RESPONSE" | head -n -1)

    if [ "$HTTP_CODE" -eq 200 ]; then
        echo "✓ Chain $CHAIN_ID created successfully"
    else
        echo "✗ Failed to create chain $CHAIN_ID (HTTP $HTTP_CODE): $BODY"
        exit 1
    fi
}

# Create the two real Canopy nodes first
echo "Creating real Canopy nodes..."
create_chain 1 "Canopy Node 1" "\"http://localhost:50002\"" "Real Canopy blockchain node 1 (from Tiltfile)"
create_chain 2 "Canopy Node 2" "\"http://localhost:40002\"" "Real Canopy blockchain node 2 (from Tiltfile)"

# Create the 100 mock chains
echo ""
echo "Creating $NUM_CHAINS mock chains (IDs $START_CHAIN_ID-$((START_CHAIN_ID + NUM_CHAINS - 1)))..."

for ((i=0; i<NUM_CHAINS; i++)); do
    CHAIN_ID=$((START_CHAIN_ID + i))
    RPC_PORT=$((61000 + i))
    RPC_URL="http://$RPC_MOCK_HOST:$RPC_PORT"

    create_chain $CHAIN_ID "Mock Chain $CHAIN_ID" "\"$RPC_URL\"" "Auto-created mock chain for development"
done

echo ""
echo "All chains created successfully!"
echo "  • 2 real Canopy nodes (IDs 1-2)"
echo "  • $NUM_CHAINS mock chains (IDs $START_CHAIN_ID-$((START_CHAIN_ID + NUM_CHAINS - 1)))"
echo ""
echo "You can verify the chains were created by running:"
echo "curl -H 'Authorization: Bearer $ADMIN_TOKEN' $API_BASE_URL/api/chains"</content>
<parameter name="filePath">scripts/create-mock-chains.sh