#!/bin/bash

# Coverage threshold checker script
# Usage: ./scripts/check-coverage.sh <threshold>

set -e

THRESHOLD=${1:-60}
COVERAGE_FILE="coverage.out"

if [ ! -f "$COVERAGE_FILE" ]; then
    echo "❌ Coverage file not found: $COVERAGE_FILE"
    echo "Run 'make test-coverage' first"
    exit 1
fi

# Extract total coverage percentage
COVERAGE=$(go tool cover -func=$COVERAGE_FILE | grep total: | awk '{print $3}' | sed 's/%//')

if [ -z "$COVERAGE" ]; then
    echo "❌ Failed to extract coverage percentage"
    exit 1
fi

echo "📊 Current coverage: ${COVERAGE}%"
echo "🎯 Required threshold: ${THRESHOLD}%"

# Use bc for floating point comparison
if command -v bc >/dev/null; then
    RESULT=$(echo "$COVERAGE >= $THRESHOLD" | bc -l)
else
    # Fallback to integer comparison if bc is not available
    COVERAGE_INT=${COVERAGE%.*}
    THRESHOLD_INT=${THRESHOLD%.*}
    if [ "$COVERAGE_INT" -ge "$THRESHOLD_INT" ]; then
        RESULT=1
    else
        RESULT=0
    fi
fi

if [ "$RESULT" -eq 1 ]; then
    echo "✅ Coverage threshold met!"
    exit 0
else
    echo "❌ Coverage below threshold!"
    echo "   Need: ${THRESHOLD}%, Got: ${COVERAGE}%"
    echo "   Add more tests to increase coverage"
    exit 1
fi