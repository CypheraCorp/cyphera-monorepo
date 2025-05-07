package delegation_server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"cyphera-api/internal/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// ExecutionObject represents the execution details for a delegation,
// including network information.
type ExecutionObject struct {
	MerchantAddress      string
	TokenContractAddress string
	TokenAmount          int64
	TokenDecimals        int32
	ChainID              uint32
	NetworkName          string
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

type DelegationClientConfig struct {
	DelegationGRPCAddr string
	RPCTimeout         time.Duration
	UseLocalMode       bool
}

// NewDelegationClient creates a new client for the delegation service.
// It establishes a connection to the gRPC server specified by the DELEGATION_GRPC_ADDR
// environment variable, or falls back to localhost:50051 if not specified.
//
// Returns:
//   - A fully initialized DelegationClient
//   - Error if the connection failed
func NewDelegationClient(config DelegationClientConfig) (*DelegationClient, error) {
	// Get gRPC server address from environment or use default
	grpcServerAddr := config.DelegationGRPCAddr
	if grpcServerAddr == "" {
		return nil, fmt.Errorf("delegation gRPC address is required")
	}

	// Get timeout from environment or use default (3 minutes)
	timeout := config.RPCTimeout
	if timeout == 0 {
		timeout = 3 * time.Minute // Default 3 minutes for blockchain operations
	}

	// Check if we're in local development mode
	// set default is false
	useLocalMode := config.UseLocalMode

	var conn *grpc.ClientConn
	var err error

	// Configure gRPC dial options for better timeout handling
	var creds grpc.DialOption
	if useLocalMode {
		creds = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		// For non-local (dev/prod), use secure credentials.
		// This typically uses the system's CA pool to verify the server's certificate.
		// Ensure the ALB's certificate is issued by a trusted CA.
		creds = grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, ""))
	}

	dialOpts := []grpc.DialOption{
		creds, // Use dynamically set credentials
		// grpc.WithBlock(), // Make connection establishment blocking -- DEPRECATED
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(20*1024*1024), // 20MB
			grpc.MaxCallSendMsgSize(20*1024*1024), // 20MB
		),
		// grpc.WithTimeout(30 * time.Second), // Add connection timeout -- Also deprecated with NewClient
	}

	// dialCtx, dialCancel := context.WithTimeout(context.Background(), 30*time.Second)
	// defer dialCancel()

	if useLocalMode {
		// Use passthrough mode for local development/testing
		// This bypasses DNS resolution and connects directly
		conn, err = grpc.NewClient(
			fmt.Sprintf("passthrough:///%s", grpcServerAddr),
			dialOpts...,
		)
	} else {
		// Use default DNS resolution for production
		// This allows for service discovery and load balancing
		conn, err = grpc.NewClient(
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

// RedeemDelegation redeems a delegation using details from the ExecutionObject.
func (c *DelegationClient) RedeemDelegation(ctx context.Context, signature []byte, executionObject ExecutionObject) (string, error) {
	// Validate inputs, now including fields from ExecutionObject
	if err := c.validateRedemptionInputs(signature, executionObject); err != nil {
		return "error when validating redemption inputs", err
	}

	log.Printf("Using timeout of %v for delegation redemption", c.rpcTimeout)
	ctx, cancel := context.WithTimeout(ctx, c.rpcTimeout)
	defer cancel()

	// Create the redemption request using fields from ExecutionObject
	req := &proto.RedeemDelegationRequest{
		Signature:            signature,
		MerchantAddress:      executionObject.MerchantAddress,
		TokenContractAddress: executionObject.TokenContractAddress,
		TokenAmount:          executionObject.TokenAmount,
		TokenDecimals:        executionObject.TokenDecimals,
		ChainId:              executionObject.ChainID,     // Use field from struct
		NetworkName:          executionObject.NetworkName, // Use field from struct
	}

	// Call the service
	res, err := c.client.RedeemDelegation(ctx, req)
	if err != nil {
		return "error when RedeemDelegation()", c.formatRPCError(err)
	}

	log.Printf("Got response from server: %+v", res)

	// Process the response
	return c.processRedemptionResponse(res)
}

// validateRedemptionInputs validates the inputs for redemption, including network info
func (c *DelegationClient) validateRedemptionInputs(signature []byte, executionObject ExecutionObject) error {
	if len(signature) == 0 {
		return fmt.Errorf("signature cannot be empty")
	}
	if executionObject.MerchantAddress == "" || executionObject.MerchantAddress == "0x0000000000000000000000000000000000000000" {
		return fmt.Errorf("valid merchant address is required")
	}
	if executionObject.TokenContractAddress == "" || executionObject.TokenContractAddress == "0x0000000000000000000000000000000000000000" {
		return fmt.Errorf("valid token contract address is required")
	}
	if executionObject.TokenAmount == 0 {
		return fmt.Errorf("valid token amount is required")
	}
	if executionObject.TokenDecimals == 0 {
		return fmt.Errorf("valid token decimals is required")
	}
	// Add validation for new fields
	if executionObject.ChainID == 0 {
		return fmt.Errorf("chain ID cannot be zero")
	}
	if executionObject.NetworkName == "" {
		return fmt.Errorf("network name cannot be empty")
	}
	return nil
}

// formatRPCError formats gRPC errors into more readable format
func (c *DelegationClient) formatRPCError(err error) error {
	st, ok := status.FromError(err)
	if ok {
		return fmt.Errorf("failed to redeem delegation: %s", st.Message())
	}
	return fmt.Errorf("failed to redeem delegation: %v", err)
}

// processRedemptionResponse processes the response from the delegation server
func (c *DelegationClient) processRedemptionResponse(res *proto.RedeemDelegationResponse) (string, error) {
	// Check if the operation was successful based on the success field
	if !res.GetSuccess() {
		errorMsg := c.extractErrorMessage(res)
		return "", fmt.Errorf("delegation redemption failed: %s", errorMsg)
	}

	txHash := res.GetTransactionHash()
	log.Printf("Transaction hash from server: %s", txHash)
	if txHash == "" {
		return "", fmt.Errorf("delegation redemption failed: empty transaction hash returned")
	}

	return txHash, nil
}

// extractErrorMessage extracts error message from the response
func (c *DelegationClient) extractErrorMessage(res *proto.RedeemDelegationResponse) string {
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

	return errorMsg
}

// RedeemDelegationDirectly attempts to redeem a delegation by calling the delegation service directly
// NOTE: This helper function passes default/zero values for chain/network info.
func (c *DelegationClient) RedeemDelegationDirectly(ctx context.Context, delegationData []byte, executionObject ExecutionObject) (string, error) {
	log.Printf("Attempting to redeem delegation (DIRECT - NOTE: chainId/networkName defaults used), data size: %d bytes", len(delegationData))
	log.Printf("Using RPC timeout of %v for delegation redemption", c.rpcTimeout)

	// Call the updated RedeemDelegation function
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
		TokenAmount:          0,
		TokenDecimals:        0,
		ChainId:              0,
		NetworkName:          "",
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
