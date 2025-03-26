# Delegation gRPC Integration Test & API

This directory contains tools for working with the delegation gRPC service, including both a command-line integration test and an HTTP API server.

## Overview

The delegation tools provide:

1. A CLI test client that creates sample delegations and sends them to the gRPC server
2. An HTTP API server that accepts delegation redemption requests
3. Support for both mock and live server modes

## Requirements

- Go 1.16 or later
- Node.js 18 or later (for mock server mode)
- npm (for mock server mode)

## Running Options

### Using the test script (Recommended)

The test script handles everything automatically:

```bash
# From the project root
./cmd/delegation-integration-test/test.sh [OPTIONS]
```

### Options

```
--server            Run in HTTP server mode
--cli               Run in CLI mode (default)
--mock              Use mock server (default)
--live              Use live server
--port PORT         Specify HTTP server port (default: 8000)
--help              Show this help message
```

### Examples

Run in CLI mode with mock server (default):
```bash
./cmd/delegation-integration-test/test.sh
```

Run as HTTP server with mock gRPC server:
```bash
./cmd/delegation-integration-test/test.sh --server
```

Run as HTTP server with live gRPC server:
```bash
./cmd/delegation-integration-test/test.sh --server --live
```

Run CLI test with live server:
```bash
./cmd/delegation-integration-test/test.sh --cli --live
```

### HTTP API Usage

When running in server mode, you can interact with the HTTP API:

```bash
# Example request to redeem a delegation
curl -X POST -H "Content-Type: application/json" \
  -d '{"delegationData": "{\"delegate\":\"0x1234...\",\"delegator\":\"0x0987...\",\"authority\":{\"scheme\":\"0x00\",\"signature\":\"0xsig\",\"signer\":\"0x5FF1...\"},\"caveats\":[],\"salt\":\"0x1234...\",\"signature\":\"0xabcd...\"}"}' \
  http://localhost:8000/api/delegations/redeem
```

### Manual Running

If you prefer to run the tools manually:

1. Start the Node.js server (if using mock mode):

```bash
# From the delegation-server directory
cd delegation-server
MOCK_MODE=true npm run dev
```

2. Build and run in CLI mode:

```bash
# From the project root
go build -o cmd/delegation-integration-test/integration-test cmd/delegation-integration-test/main.go
./cmd/delegation-integration-test/integration-test -verbose
```

3. Or build and run in server mode:

```bash
# From the project root
go build -o cmd/delegation-integration-test/integration-test cmd/delegation-integration-test/main.go
./cmd/delegation-integration-test/integration-test -server -port 8000
```

## Command Line Options

The integration test client supports the following command line flags:

- `-server`: Run in HTTP server mode
- `-port`: Specify HTTP server port when in server mode (default: "8000")
- `-delegator`: Delegator address (default: "0x1234...")
- `-delegate`: Delegate address (default: "0x0987...")
- `-signature`: Delegation signature (default: "0xabcd...")
- `-salt`: Delegation salt (default: "0x123456789")
- `-verbose`: Enable verbose output (default: false) 