#!/bin/bash
# load-env.sh - Standardized environment variable loader for the monorepo
# Usage: source ./scripts/load-env.sh [app-name]

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the app name from arguments (optional)
APP_NAME=$1

# Get the script directory and project root
if [ -n "${BASH_SOURCE[0]}" ]; then
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
else
    SCRIPT_DIR="$(pwd)/scripts"
fi
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

echo -e "${GREEN}Loading environment variables...${NC}"

# Function to load env file if it exists
load_env_file() {
    local env_file=$1
    if [ -f "$env_file" ]; then
        echo -e "${YELLOW}Loading: $env_file${NC}"
        # Export variables, ignoring comments and empty lines
        set -a
        source <(grep -v '^#' "$env_file" | grep -v '^$' || true)
        set +a
        return 0
    else
        return 1
    fi
}

# 1. Load root .env file (shared configuration)
if load_env_file "$PROJECT_ROOT/.env"; then
    echo -e "${GREEN}✓ Loaded root .env${NC}"
else
    echo -e "${YELLOW}⚠ No root .env file found${NC}"
fi

# 2. Load app-specific .env.local if app name provided
if [ -n "$APP_NAME" ]; then
    APP_DIR="$PROJECT_ROOT/apps/$APP_NAME"
    
    if [ ! -d "$APP_DIR" ]; then
        echo -e "${RED}✗ App directory not found: $APP_DIR${NC}"
        exit 1
    fi
    
    # For TypeScript/Node apps, load .env.local
    if [ -f "$APP_DIR/package.json" ]; then
        if load_env_file "$APP_DIR/.env.local"; then
            echo -e "${GREEN}✓ Loaded $APP_NAME/.env.local${NC}"
        else
            echo -e "${YELLOW}⚠ No .env.local found for $APP_NAME${NC}"
        fi
    fi
    
    # For Go apps, they should use root .env
    if [ -f "$APP_DIR/go.mod" ]; then
        echo -e "${GREEN}✓ Go app using root .env${NC}"
    fi
fi

# 3. Set default values for critical variables if not set
if [ -z "$DATABASE_URL" ]; then
    echo -e "${YELLOW}⚠ DATABASE_URL not set, using default${NC}"
    export DATABASE_URL="postgres://apiuser:apipassword@localhost:5432/cyphera?sslmode=disable"
fi

if [ -z "$STAGE" ]; then
    export STAGE="local"
    echo -e "${YELLOW}⚠ STAGE not set, defaulting to 'local'${NC}"
fi

# 4. Validate required environment variables based on app
validate_env_vars() {
    local required_vars=("$@")
    local missing_vars=()
    
    for var in "${required_vars[@]}"; do
        if [ -z "${!var}" ]; then
            missing_vars+=("$var")
        fi
    done
    
    if [ ${#missing_vars[@]} -gt 0 ]; then
        echo -e "${RED}✗ Missing required environment variables:${NC}"
        printf '%s\n' "${missing_vars[@]}"
        return 1
    fi
    
    return 0
}

# App-specific validation
case "$APP_NAME" in
    "api")
        validate_env_vars DATABASE_URL WEB3AUTH_CLIENT_ID CIRCLE_API_KEY || exit 1
        ;;
    "delegation-server")
        validate_env_vars GRPC_PORT || exit 1
        ;;
    "web-app")
        validate_env_vars NEXT_PUBLIC_API_URL || exit 1
        ;;
    "subscription-processor")
        validate_env_vars DATABASE_URL DELEGATION_SERVER_ADDRESS || exit 1
        ;;
esac

echo -e "${GREEN}✓ Environment loaded successfully${NC}"

# Display loaded configuration (with sensitive values masked)
if [ "${SHOW_ENV:-false}" = "true" ]; then
    echo -e "\n${YELLOW}Loaded configuration:${NC}"
    echo "STAGE=$STAGE"
    echo "DATABASE_URL=${DATABASE_URL:0:20}..."
    echo "API_PORT=${API_PORT:-8000}"
    [ -n "$APP_NAME" ] && echo "APP=$APP_NAME"
fi