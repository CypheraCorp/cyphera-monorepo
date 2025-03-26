#!/bin/bash
# Load variables from delegation-server/.env
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi
# Log environment variables for debugging (with sensitive info redacted)
echo "Environment variables being used by delegation server:"
echo "MOCK_MODE=${MOCK_MODE}"
echo "GRPC_HOST=${GRPC_HOST}"
echo "GRPC_PORT=${GRPC_PORT}"
echo "RPC_URL=${RPC_URL}"
echo "BUNDLER_URL=${BUNDLER_URL}"
echo "CHAIN_ID=${CHAIN_ID}"
[ -n "${PRIVATE_KEY}" ] && echo "PRIVATE_KEY=[REDACTED]" || echo "PRIVATE_KEY=not set"
echo "LOG_LEVEL=${LOG_LEVEL}"
npm run dev
