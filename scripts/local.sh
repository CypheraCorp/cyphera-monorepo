#!/bin/bash
set -e

# Load environment variables
if [ -f .env.local ]; then
    export $(cat .env.local | grep -v '^#' | xargs)
    echo "Using .env.local"
else
    export $(cat .env | grep -v '^#' | xargs)
    echo "Using .env"
fi

# Build the binary
go build -o bin/cyphera-api cmd/api/local/main.go

# Run the binary with environment variables
echo "Starting local server..."
./bin/cyphera-api