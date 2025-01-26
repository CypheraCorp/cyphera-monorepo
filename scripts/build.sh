#!/bin/bash
set -e  # Exit on error

# Cleanup any existing artifacts
rm -f bootstrap function.zip

# Build the binary
echo "Building Go binary for AWS Lambda (ARM64)..."
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build \
  -tags lambda.norpc \
  -ldflags="-s -w" \
  -o bootstrap \
  cmd/api/main/main.go

# Verify the binary
echo "Verifying binary architecture..."
if ! file bootstrap | grep -q "ARM aarch64"; then
    echo "Error: Binary is not compiled for ARM64"
    file bootstrap
    exit 1
fi

# Make the binary executable
chmod +x bootstrap

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

# Final verification
if [ -f bootstrap ]; then
    echo "Successfully created bootstrap binary"
    echo "Binary details:"
    file bootstrap
    ls -l bootstrap
else
    echo "Failed to create bootstrap binary"
    exit 1
fi
