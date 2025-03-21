#!/bin/bash

set -e

# Change to the project root directory if running from scripts folder
if [[ "$(basename "$(pwd)")" == "scripts" ]]; then
  cd ..
fi

# Function to check if required environment variables are set
check_env_vars() {
  local missing=0
  
  if [ -z "$RPC_URL" ]; then
    echo "❌ RPC_URL is not set"
    missing=1
  fi
  
  if [ -z "$BUNDLER_URL" ]; then
    echo "❌ BUNDLER_URL is not set"
    missing=1
  fi
  
  if [ -z "$PRIVATE_KEY" ]; then
    echo "❌ PRIVATE_KEY is not set"
    missing=1
  fi
  
  if [ "$missing" -eq 1 ]; then
    echo "Please set all required environment variables in .env file or export them"
    exit 1
  fi
}

# Check if .env file exists
if [ -f .env ]; then
  echo "📄 Loading environment variables from .env file"
  export $(grep -v '^#' .env | xargs)
  check_env_vars
else
  echo "⚠️ No .env file found, checking environment variables"
  check_env_vars
fi

# Check Node.js version
required_node_version="18"
node_version=$(node -v | cut -d. -f1 | tr -d 'v')

if [ "$node_version" -lt "$required_node_version" ]; then
  echo "❌ Node.js version $node_version detected. Version $required_node_version or higher is required."
  exit 1
fi

# Check if dependencies are installed
if [ ! -d "node_modules" ]; then
  echo "📦 Installing dependencies..."
  npm install --legacy-peer-deps
  
  if [ $? -ne 0 ]; then
    echo "❌ Failed to install dependencies"
    exit 1
  fi
fi

# Check for TypeScript transpiler
if ! command -v tsc &> /dev/null; then
  echo "📦 Installing TypeScript globally..."
  npm install -g typescript
fi

# Start the server based on environment
if [ "$NODE_ENV" = "production" ]; then
  echo "🚀 Starting server in production mode..."
  npm run build && npm start
else
  echo "🚀 Starting server in development mode..."
  npm run dev
fi 