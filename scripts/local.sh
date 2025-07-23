#!/bin/bash
set -e

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
    echo "Using .env"
else
    echo "No .env file found"
fi

# Build the binary
go build -o bin/cyphera-api apps/api/cmd/local/main.go

# Run the binary with environment variables
echo "Starting local server..."
./bin/cyphera-api