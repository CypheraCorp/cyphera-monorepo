#!/bin/bash
set -e  # Exit on error

# Cleanup any existing artifacts
rm -f bootstrap function.zip

# Build the binary
echo "Building Go binary for AWS Lambda (x86_64)..."
GOOS=linux \
GOARCH=amd64 \
CGO_ENABLED=0 \
go build \
  -tags lambda.norpc \
  -ldflags="-s -w" \
  -o bootstrap \
  apps/api/cmd/main/main.go

# Print binary information
echo "Binary details:"
file bootstrap
ls -lh bootstrap

# Verify the binary architecture
echo "Verifying binary architecture..."
if ! file bootstrap | grep -q "x86-64"; then
    echo "Error: Binary is not compiled for x86_64"
    echo "Binary details:"
    file bootstrap
    exit 1
fi

# Make the binary executable
chmod +x bootstrap

# Print final binary information
echo "Final binary details:"
file bootstrap
ls -lh bootstrap

# Create deployment package (optional, for local testing)
if [ "${CREATE_ZIP:-false}" = "true" ]; then
    echo "Creating deployment package..."
    zip -j function.zip bootstrap
    
    # Verify the zip file was created
    if [ -f function.zip ]; then
        echo "Successfully created function.zip"
        echo "Zip file size:"
        ls -lh function.zip
    else
        echo "Failed to create function.zip"
        exit 1
    fi
fi
