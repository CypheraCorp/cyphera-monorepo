#!/bin/bash
set -e  # Exit on error

# Cleanup any existing artifacts
rm -f bootstrap function.zip

# Build the binary
echo "Building Go binary..."
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap cmd/api/main.go

# Create deployment package
echo "Creating deployment package..."
zip -j function.zip bootstrap

# Verify the zip file was created
if [ -f function.zip ]; then
    echo "Successfully created function.zip"
else
    echo "Failed to create function.zip"
    exit 1
fi

# Cleanup bootstrap binary
rm bootstrap
