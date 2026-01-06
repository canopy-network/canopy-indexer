#!/bin/sh
# Wrapper script for development - restarts binary when it changes

BINARY="/app/bin/indexer"

echo "Starting indexer wrapper..."

while true; do
    if [ -f "$BINARY" ]; then
        echo "Starting indexer..."
        "$BINARY"
        EXIT_CODE=$?
        echo "Indexer exited with code $EXIT_CODE"

        if [ $EXIT_CODE -eq 0 ]; then
            echo "Clean exit, stopping wrapper"
            exit 0
        fi

        echo "Restarting in 2 seconds..."
        sleep 2
    else
        echo "Waiting for binary at $BINARY..."
        sleep 2
    fi
done
