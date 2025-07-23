#!/bin/bash
# Script to build Lambda functions using Nx for SAM deployment

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Building Lambda functions with Nx...${NC}"

# Function to build a specific lambda
build_lambda() {
    local app_name=$1
    local output_path=$2
    
    echo -e "${YELLOW}Building ${app_name}...${NC}"
    npx nx run ${app_name}:build-lambda
    
    # Check if bootstrap was created
    if [ -f "apps/${app_name}/bootstrap" ]; then
        echo -e "${GREEN}✓ ${app_name} built successfully${NC}"
        # Copy to SAM expected location if provided
        if [ -n "$output_path" ]; then
            mkdir -p $(dirname "$output_path")
            cp "apps/${app_name}/bootstrap" "$output_path"
            echo -e "${GREEN}✓ Copied bootstrap to ${output_path}${NC}"
        fi
    else
        echo -e "${RED}✗ Failed to build ${app_name}${NC}"
        exit 1
    fi
}

# Check if specific app is requested
if [ $# -eq 0 ]; then
    # Build all Lambda functions
    echo -e "${GREEN}Building all Lambda functions...${NC}"
    npx nx run-many --target=build-lambda --projects=subscription-processor,webhook-receiver,webhook-processor,dlq-processor
else
    # Build specific app
    APP_NAME=$1
    OUTPUT_PATH=$2
    build_lambda "$APP_NAME" "$OUTPUT_PATH"
fi

echo -e "${GREEN}Lambda build complete!${NC}"