#!/bin/bash

# Script to run delegation server locally with proper environment setup

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting Delegation Server (Local Environment)${NC}"

# Check if .env.local exists, otherwise use .env
if [ -f ".env.local" ]; then
    echo -e "${YELLOW}Loading environment from .env.local${NC}"
    export $(cat .env.local | grep -v '^#' | xargs)
else
    echo -e "${YELLOW}Loading environment from .env${NC}"
    export $(cat .env | grep -v '^#' | xargs)
fi

# Override with any system environment variables
if [ ! -z "$DATABASE_URL" ]; then
    echo -e "${GREEN}Using DATABASE_URL from system environment${NC}"
fi

# Check if database URL is set
if [ -z "$DATABASE_URL" ] && [ -z "$DATABASE_CONNECTION_STRING" ]; then
    echo -e "${RED}ERROR: No database connection string found!${NC}"
    echo "Please set DATABASE_URL in your .env file or as an environment variable"
    echo "Example: DATABASE_URL=postgres://apiuser:apipassword@localhost:5432/cyphera?sslmode=disable"
    exit 1
fi

# Display connection info (hide password)
if [ ! -z "$DATABASE_URL" ]; then
    SANITIZED_URL=$(echo $DATABASE_URL | sed 's/:\/\/[^:]*:[^@]*@/:\/\/*****:*****@/')
    echo -e "${GREEN}Database URL: ${SANITIZED_URL}${NC}"
fi

# Check if PostgreSQL is running locally
if [[ "$DATABASE_URL" == *"localhost"* ]] || [[ "$DATABASE_URL" == *"127.0.0.1"* ]]; then
    echo -e "${YELLOW}Checking local PostgreSQL...${NC}"
    if ! pg_isready -h localhost -p 5432 >/dev/null 2>&1; then
        echo -e "${RED}WARNING: PostgreSQL doesn't seem to be running on localhost:5432${NC}"
        echo "You may need to start it with: brew services start postgresql@15"
    else
        echo -e "${GREEN}PostgreSQL is running${NC}"
    fi
fi

# Run the delegation server
echo -e "${GREEN}Starting delegation server on port ${GRPC_PORT:-50051}...${NC}"
npm run dev