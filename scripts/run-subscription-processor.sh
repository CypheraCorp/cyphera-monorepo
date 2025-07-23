#!/bin/bash

# Script to run subscription processor on an interval for local development
# Default interval is 10 seconds, can be overridden with SUBSCRIPTION_INTERVAL env var

INTERVAL=${SUBSCRIPTION_INTERVAL:-10}
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$SCRIPT_DIR/.."

echo "Starting subscription processor with interval: ${INTERVAL}s"
echo "Press Ctrl+C to stop"

# Change to subscription processor directory
cd "$PROJECT_ROOT/apps/subscription-processor" || exit 1

# Run immediately on startup
echo "[$(date '+%Y-%m-%d %H:%M:%S')] Running subscription processor..."
go run cmd/main.go

# Run in a loop
while true; do
    sleep "$INTERVAL"
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Running subscription processor..."
    go run cmd/main.go
done