package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"cyphera-api/internal/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// ExecutionObject represents the execution details for a delegation
type ExecutionObject struct {
	MerchantAddress      string
	TokenContractAddress string
	Price                string
}

// DelegationData represents the delegation information stored in the database
type DelegationData struct {
	Delegate  string          `json:"delegate"`
	Delegator string          `json:"delegator"`
	Authority string          `json:"authority"`
	Caveats   json.RawMessage `json:"caveats"`
	Salt      string          `json:"salt"`
	Signature string          `json:"signature"`
}

// DelegationClient handles communication with the gRPC delegation service.
// It provides methods to redeem delegations and manage the gRPC connection.
type DelegationClient struct {
	conn       *grpc.ClientConn
	client     proto.DelegationServiceClient
	rpcTimeout time.Duration // Timeout for RPC calls
}

// NewDelegationClient creates a new client for the delegation service.
// It establishes a connection to the gRPC server specified by the DELEGATION_GRPC_ADDR
// environment variable, or falls back to localhost:50051 if not specified.
//
// Returns:
//   - A fully initialized DelegationClient
//   - Error if the connection failed
func NewDelegationClient() (*DelegationClient, error) {
	// Get gRPC server address from environment or use default
	grpcServerAddr := os.Getenv("DELEGATION_GRPC_ADDR")
	if grpcServerAddr == "" {
		grpcServerAddr = "localhost:50051"
	}

	// Get timeout from environment or use default (3 minutes)
	timeoutStr := os.Getenv("DELEGATION_RPC_TIMEOUT")
	timeout := 3 * time.Minute // Default 3 minutes for blockchain operations
	if timeoutStr != "" {
		if parsedTimeout, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsedTimeout
		} else {
			log.Printf("Warning: Invalid DELEGATION_RPC_TIMEOUT value: %s, using default", timeoutStr)
		}
	}

	// Check if we're in local development mode
	// Set DELEGATION_LOCAL_MODE=true for dev/test environments
	useLocalMode := os.Getenv("DELEGATION_LOCAL_MODE") == "true"

	var conn *grpc.ClientConn
	var err error

	// Configure gRPC dial options for better timeout handling
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(), // Make connection establishment blocking
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(20*1024*1024), // 20MB
			grpc.MaxCallSendMsgSize(20*1024*1024), // 20MB
		),
	}

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer dialCancel()

	if useLocalMode {
		// Use passthrough mode for local development/testing
		// This bypasses DNS resolution and connects directly
		conn, err = grpc.DialContext(
			dialCtx,
			fmt.Sprintf("passthrough:///%s", grpcServerAddr),
			dialOpts...,
		)
	} else {
		// Use default DNS resolution for production
		// This allows for service discovery and load balancing
		conn, err = grpc.DialContext(
			dialCtx,
			grpcServerAddr,
			dialOpts...,
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to delegation gRPC server: %w", err)
	}

	// Create a client stub
	client := proto.NewDelegationServiceClient(conn)

	return &DelegationClient{
		conn:       conn,
		client:     client,
		rpcTimeout: timeout,
	}, nil
}

// RedeemDelegation redeems a delegation with the provided signature and execution details
func (c *DelegationClient) RedeemDelegation(ctx context.Context, signature []byte, executionObject ExecutionObject) (string, error) {
	// Validate inputs
	if len(signature) == 0 {
		return "", fmt.Errorf("signature cannot be empty")
	}

	if executionObject.MerchantAddress == "" || executionObject.MerchantAddress == "0x0000000000000000000000000000000000000000" {
		return "", fmt.Errorf("valid merchant address is required")
	}

	if executionObject.TokenContractAddress == "" || executionObject.TokenContractAddress == "0x0000000000000000000000000000000000000000" {
		return "", fmt.Errorf("valid token contract address is required")
	}

	if executionObject.Price == "" || executionObject.Price == "0" {
		return "", fmt.Errorf("valid price is required")
	}

	// Create a context with the configured timeout (default is now 3 minutes)
	// Using a longer timeout to accommodate blockchain operations
	log.Printf("Using timeout of %v for delegation redemption", c.rpcTimeout)
	ctx, cancel := context.WithTimeout(ctx, c.rpcTimeout)
	defer cancel()

	// Create the redemption request
	req := &proto.RedeemDelegationRequest{
		Signature:            signature,
		MerchantAddress:      executionObject.MerchantAddress,
		TokenContractAddress: executionObject.TokenContractAddress,
		Price:                executionObject.Price,
	}

	// Call the service
	res, err := c.client.RedeemDelegation(ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			return "", fmt.Errorf("failed to redeem delegation: %s", st.Message())
		}
		return "", fmt.Errorf("failed to redeem delegation: %v", err)
	}

	// Log the full response for debugging
	log.Printf("Got response from server: %+v", res)

	// Check if the operation was successful based on the success field
	if !res.GetSuccess() {
		errorMsg := res.GetErrorMessage()
		// The field might be in snake_case in the response if the server is using keepCase: true
		if errorMsg == "" {
			// Check if we can extract the error message from another field or property
			errorMsgBytes, _ := json.Marshal(res)
			log.Printf("JSON response: %s", string(errorMsgBytes))
			if errorMsgBytes != nil {
				var respMap map[string]interface{}
				if err := json.Unmarshal(errorMsgBytes, &respMap); err == nil {
					log.Printf("Response map: %+v", respMap)
					// Try different field name variations
					if snake, ok := respMap["error_message"].(string); ok && snake != "" {
						errorMsg = snake
					} else if camel, ok := respMap["errorMessage"].(string); ok && camel != "" {
						errorMsg = camel
					}
				}
			}
		}

		if errorMsg == "" {
			errorMsg = "unknown error (empty error message from server)"
		}
		return "", fmt.Errorf("delegation redemption failed: %s", errorMsg)
	}

	txHash := res.GetTransactionHash()
	log.Printf("Transaction hash from server: %s", txHash)
	if txHash == "" {
		return "", fmt.Errorf("delegation redemption failed: empty transaction hash returned")
	}

	return txHash, nil
}

// RedeemDelegationDirectly attempts to redeem a delegation by calling the delegation service directly
func (c *DelegationClient) RedeemDelegationDirectly(ctx context.Context, delegationData []byte, merchantAddress, tokenAddress, price string) (string, error) {
	log.Printf("Attempting to redeem delegation, data size: %d bytes", len(delegationData))
	log.Printf("Using merchant address: %s, token address: %s, price: %s", merchantAddress, tokenAddress, price)
	log.Printf("Using RPC timeout of %v for delegation redemption", c.rpcTimeout)

	// Create execution object with actual values
	executionObject := ExecutionObject{
		MerchantAddress:      merchantAddress,
		TokenContractAddress: tokenAddress,
		Price:                price,
	}

	// Call the client to redeem the delegation
	txHash, err := c.RedeemDelegation(ctx, delegationData, executionObject)
	if err != nil {
		log.Printf("Delegation redemption failed: %v", err)
		return "", fmt.Errorf("delegation redemption failed: %w", err)
	}

	log.Printf("Delegation successfully redeemed, tx hash: %s", txHash)
	return txHash, nil
}

// HealthCheck checks if the delegation server is available
// by making a minimal gRPC request
//
// Parameters:
//   - ctx: Context for the request
//
// Returns:
//   - nil if the server is available
//   - Error if the server is unavailable
func (c *DelegationClient) HealthCheck(ctx context.Context) error {
	// Create a short timeout context for health check
	// Health checks should be quick, so we use a shorter timeout than for redemptions
	healthCheckTimeout := 10 * time.Second // 10 seconds for health check
	timeoutCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	// Create a minimal request (will be rejected by server but that's fine for checking connection)
	req := &proto.RedeemDelegationRequest{
		Signature:            []byte{},
		MerchantAddress:      "0x0000000000000000000000000000000000000000",
		TokenContractAddress: "0x0000000000000000000000000000000000000000",
		Price:                "0",
	}

	// Try to call the service
	_, err := c.client.RedeemDelegation(timeoutCtx, req)
	if err != nil {
		// We expect an error here since we're sending empty data
		// But we want to distinguish between connection errors and validation errors

		// Extract the error details
		st, _ := status.FromError(err)

		// Check if the error is a connection error or a validation error
		// Status codes like Unavailable (14) indicate connection issues
		// while codes like InvalidArgument (3) indicate validation issues
		if st.Code() == 14 { // Unavailable
			return fmt.Errorf("delegation server unavailable: %s", st.Message())
		}

		// If we got an error with a different code, it means we connected
		// to the server but it rejected our request, which is expected
		// This indicates the server is alive
		return nil
	}

	// If we somehow got a successful response for our empty request,
	// the server is definitely available
	return nil
}

// Close closes the gRPC connection. This should be called when the client
// is no longer needed to free up resources.
func (c *DelegationClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
