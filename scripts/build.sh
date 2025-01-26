#!/bin/bash
set -e  # Exit on error

# Cleanup any existing artifacts
rm -f bootstrap function.zip

# Build the binary
echo "Building Go binary for AWS Lambda (ARM64)..."
GOARCH=arm64 \
GOOS=linux \
CGO_ENABLED=0 \
GOARM=7 \
go build \
  -tags lambda.norpc \
  -ldflags="-s -w -X main.Version=1.0.0" \
  -trimpath \
  -o bootstrap \
  cmd/api/main/main.go

# Strip binary (additional size reduction)
echo "Stripping binary..."
strip bootstrap 2>/dev/null || true

# Compress the binary
echo "Compressing binary with UPX..."
if command -v upx >/dev/null 2>&1; then
    upx -9 bootstrap
else
    echo "UPX not installed, skipping compression"
fi

# Verify the binary
echo "Verifying binary architecture..."
if ! file bootstrap | grep -q "aarch64"; then
    echo "Error: Binary is not compiled for ARM64"
    echo "Binary details:"
    file bootstrap
    exit 1
fi

# Make the binary executable
chmod +x bootstrap

# Print binary information
echo "Binary details:"
file bootstrap
ls -lh bootstrap  # Using -h for human-readable sizes

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
