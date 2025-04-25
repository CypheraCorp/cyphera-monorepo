#!/bin/sh
# .githooks/pre-commit/01-golangci-lint.sh

echo "[githooks] Running golangci-lint..."

# Ensure golangci-lint is available
if ! command -v golangci-lint > /dev/null; then
    echo "[githooks] Error: golangci-lint command not found." >&2
    echo "[githooks] Please install it: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" >&2
    exit 1
fi

# Lint all Go files in the project
golangci-lint run ./...
LINT_EXIT_CODE=$?

if [ $LINT_EXIT_CODE -ne 0 ]; then
    echo "[githooks] Error: golangci-lint found issues. Please fix them before committing." >&2
    exit 1 # Abort commit
fi

echo "[githooks] golangci-lint passed."
exit 0 # Proceed with commit 