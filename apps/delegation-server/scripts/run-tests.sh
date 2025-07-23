#!/bin/bash

set -e

# Change to the project root directory if run from scripts folder
if [[ "$(basename $(pwd))" == "scripts" ]]; then
  cd ..
fi

echo "🧪 Running delegation server tests..."

# Run unit tests first
echo "🔬 Running unit tests..."
npm test

# Check if server is running for integration tests
if nc -z localhost 50051 >/dev/null 2>&1; then
  echo "🔌 Delegation server detected - running integration tests..."
  npm run test:integration
else
  echo "⚠️ Delegation server not running - skipping integration tests"
  echo "To run integration tests, start the server with 'npm run dev' or 'npm start' in another terminal"
fi

echo "✅ Test suite completed" 