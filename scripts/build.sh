#!/bin/bash
set -e  # Exit on error

# Cleanup any existing artifacts
rm -f bootstrap function.zip

# Build the binary
echo "Building Go binary..."
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap cmd/api/main/main.go

# Create deployment package (optional, for local testing)
if [ "${CREATE_ZIP:-false}" = "true" ]; then
    echo "Creating deployment package..."
    zip -j function.zip bootstrap
    
    # Verify the zip file was created
    if [ -f function.zip ]; then
        echo "Successfully created function.zip"
    else
        echo "Failed to create function.zip"
        exit 1
    fi
fi

# Verify the bootstrap file exists
if [ -f bootstrap ]; then
    echo "Successfully created bootstrap binary"
else
    echo "Failed to create bootstrap binary"
    exit 1
fi
