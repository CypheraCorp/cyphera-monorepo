#!/bin/bash

set -e

# Change to the project root directory if run from scripts folder
if [[ "$(basename $(pwd))" == "scripts" ]]; then
  cd ..
fi

echo "🧪 Running delegation server integration test..."

# Check if server is running
if ! nc -z localhost 50051 >/dev/null 2>&1; then
  echo "❌ Error: Delegation server is not running"
  echo "Please start the server first with 'npm run dev' or 'npm start'"
  exit 1
fi

# Install ts-node if not already installed
if ! command -v ts-node &> /dev/null; then
  echo "📦 Installing ts-node globally..."
  npm install -g ts-node
fi

# Run the integration test
echo "🚀 Executing integration test..."
ts-node test/integration-test.ts

exit_code=$?

if [ $exit_code -eq 0 ]; then
  echo "✅ Integration test completed successfully"
else
  echo "❌ Integration test failed with exit code $exit_code"
fi

exit $exit_code 