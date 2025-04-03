#!/bin/bash
# Load variables from delegation-server/.env
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

# Use npx to find the locally installed ts-node
npx ts-node src/index.ts
