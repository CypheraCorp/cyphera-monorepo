# MetaMask Delegation Redemption System

## Overview

This system enables our Go API to redeem MetaMask delegations using a dedicated Node.js gRPC service, allowing users to complete transactions without needing to sign them directly.

## System Components

1. **Node.js gRPC Server**
   - Handles the actual delegation redemption process
   - Communicates with blockchain providers and bundlers
   - Uses MetaMask's delegation libraries for account creation and transaction submission

2. **Go API Integration**
   - Exposes HTTP endpoints for delegation redemption
   - Validates and forwards delegation data to the Node.js service
   - Handles response data and error reporting

## Key Features

- **Delegated Transactions**: Users can authorize transactions in advance that can be executed later
- **Smart Account Creation**: Creates MetaMask smart accounts using the configured private key
- **Gas Abstraction**: Handles gas fees and paymaster interactions
- **Error Handling**: Provides detailed error messages for debugging

## Architecture

```
                ┌───────────────────┐         ┌────────────────────┐
                │                   │         │                    │
 ┌─────────┐    │    Go API         │ gRPC    │   Node.js Server   │
 │         │    │  ┌─────────────┐  │ call    │  ┌──────────────┐  │
 │  User   ├────┼──┤ HTTP Routes ├──┼─────────┼──┤ gRPC Service │  │
 │         │    │  └─────┬───────┘  │         │  └──────┬───────┘  │
 └─────────┘    │        │          │         │         │          │
                │        ▼          │         │         ▼          │
                │  ┌─────────────┐  │         │  ┌──────────────┐  │
                │  │ Delegation  │  │         │  │  Blockchain  │  │
                │  │   Client    │  │         │  │  Operations  │  │         ┌─────────────┐
                │  └─────────────┘  │         │  └──────┬───────┘  │         │             │
                │                   │         │         │          │         │ Blockchain  │
                └───────────────────┘         │         ▼          │         │             │
                                              │  ┌──────────────┐  │         │  ┌───────┐  │
                                              │  │Smart Account │  │user op  │  │       │  │
                                              │  │  Creation    ├──┼─────────┼─►│ ERC20 │  │
                                              │  └──────────────┘  │         │  │       │  │
                                              │                    │         │  └───────┘  │
                                              └────────────────────┘         │             │
                                                                             └─────────────┘
```

## Components

### Go API Components

1. **Delegation Client** (`internal/client/delegation_client.go`)
   - Provides a reusable client for the gRPC service
   - Handles connection management and error handling
   - Used by both the direct handler and the HTTP handlers

2. **Delegation Handler** (`internal/handler/delegation_handler.go`)
   - Provides a simple interface for direct delegation redemption
   - Used in microservices or background jobs

3. **HTTP Handler** (`internal/handlers/delegation_handlers.go`)
   - Exposes HTTP endpoints for delegation redemption
   - Handles request/response formatting and error reporting
   - Used for user-facing API endpoints

4. **Proto Definitions** (`internal/proto/delegation.proto`)
   - Defines the gRPC service interface and message types

### Node.js Server Components

1. **Node.js gRPC Server**
   - Handles the actual delegation redemption process
   - Communicates with blockchain providers and bundlers
   - Uses MetaMask's delegation libraries for account creation and transaction submission

## Setup and Integration

For detailed setup instructions, refer to:
- [README.md](js-server/README.md) for Node.js server setup
- [INTEGRATION_GUIDE.md](INTEGRATION_GUIDE.md) for complete integration details
- [TESTING_GUIDE.md](TESTING_GUIDE.md) for testing procedures

## Directory Structure

```
├── install-grpc-client.sh         # Script to generate Go gRPC client code
├── internal/
│   ├── handler/
│   │   └── delegation_handler.go  # Go delegation handler
│   └── proto/
│       └── delegation.proto       # Protocol buffer definition
├── js-server/                     # Node.js gRPC server
│   ├── src/
│   │   ├── blockchain.ts          # Blockchain operations
│   │   ├── grpc-server.ts         # gRPC server implementation
│   │   └── service.ts             # Service implementation
│   ├── .env.example               # Environment variable template
│   ├── package.json               # Dependencies and scripts
│   └── run.sh                     # Server startup script
└── scripts/
    ├── test_delegation.js         # JS test script
    └── test_delegation.sh         # Shell test script
```

## Usage Examples

### Go API (Server-Side)

```go
// Example handler function
func RedeemDelegationHandler(c *gin.Context) {
    // Parse delegation data from request
    var req struct {
        DelegationData string `json:"delegationData" binding:"required"`
    }
    if err := c.BindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }

    // Call the delegation handler
    txHash, err := delegationHandler.RedeemDelegation(c, []byte(req.DelegationData))
    if err != nil {
        c.JSON(500, gin.H{"success": false, "error": err.Error()})
        return
    }

    // Return success response
    c.JSON(200, gin.H{"success": true, "transactionHash": txHash})
}
```

### Client-Side (JavaScript)

```javascript
// Example client-side code to redeem a delegation
async function redeemDelegation(delegationData) {
  const response = await fetch('/api/v1/delegations/redeem', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${apiToken}`
    },
    body: JSON.stringify({ delegationData })
  });
  
  const result = await response.json();
  
  if (!result.success) {
    throw new Error(result.error);
  }
  
  return result.transactionHash;
}
```

## Security Considerations

1. **Private Key Management**: The private key used for signing transactions is sensitive. Use secure environment variables or a key management service.

2. **API Access Control**: Ensure proper authentication and authorization for the delegation redemption endpoint.

3. **Request Validation**: Validate all delegation data before processing to prevent malicious inputs.

4. **Network Configuration**: Use TLS for all communication between the Go API and Node.js gRPC server in production.

## Monitoring and Maintenance

- **Logging**: Both the Go API and Node.js server have proper logging for debugging.
- **Health Checks**: Implement regular health checks to ensure the services are running.
- **Error Alerts**: Set up alerting for critical errors in the redemption process.

## Integration

### Using the Delegation Client Directly

```go
import (
    "context"
    "encoding/json"
    "log"
    "time"
    
    "cyphera-api/internal/client"
)

func redeemUserDelegation() {
    // Create a delegation client
    delegationClient, err := client.NewDelegationClient()
    if err != nil {
        log.Fatalf("Failed to create delegation client: %v", err)
    }
    defer delegationClient.Close()
    
    // Prepare delegation data (from your database or request)
    delegation := getDelegationFromDatabase()
    
    // Serialize to JSON
    delegationJSON, _ := json.Marshal(delegation)
    
    // Create context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    // Call the service
    txHash, err := delegationClient.RedeemDelegation(ctx, delegationJSON)
    if err != nil {
        log.Printf("Redemption failed: %v", err)
        return
    }
    
    log.Printf("Delegation redeemed: %s", txHash)
}
```

### Using the HTTP Handler in Your API

// ... existing code ... 