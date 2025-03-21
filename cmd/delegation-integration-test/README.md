# Delegation gRPC Integration Test

This directory contains an integration test for the delegation gRPC service that demonstrates how to use the Go client to communicate with the Node.js gRPC server.

## Overview

The integration test:

1. Creates a sample delegation with test values
2. Connects to the Node.js gRPC delegation server
3. Sends the delegation data to be redeemed
4. Verifies that the response contains a transaction hash

## Requirements

- Go 1.16 or later
- Node.js 18 or later
- npm

## Running the Test

You can run the test in two ways:

### 1. Using the test script (Recommended)

The test script handles everything automatically:
- Starts the Node.js gRPC server in mock mode
- Runs the Go client
- Shuts down the server when done

```bash
# From the project root
./cmd/delegation-integration-test/test.sh
```

### 2. Manually

If you prefer to run the test manually:

1. Start the Node.js server in mock mode:

```bash
# From the delegation-server directory
cd delegation-server
MOCK_MODE=true npm run dev
```

2. In another terminal, build and run the test client:

```bash
# From the project root
go build -o cmd/delegation-integration-test/integration-test cmd/delegation-integration-test/main.go
./cmd/delegation-integration-test/integration-test -verbose
```

## Command Line Options

The integration test client supports the following command line flags:

- `-delegator`: Delegator address (default: "0x1234567890123456789012345678901234567890")
- `-delegate`: Delegate address (default: "0x0987654321098765432109876543210987654321")
- `-signature`: Delegation signature (default: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
- `-expiry`: Expiry timestamp (default: 1 hour from now)
- `-salt`: Delegation salt (default: "0x123456789")
- `-verbose`: Enable verbose output (default: false) 