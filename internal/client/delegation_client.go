package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"cyphera-api/internal/proto"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

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
	conn   *grpc.ClientConn
	client proto.DelegationServiceClient
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

	// Connect to the gRPC server
	conn, err := grpc.Dial(grpcServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to delegation gRPC server: %w", err)
	}

	// Create a client stub
	client := proto.NewDelegationServiceClient(conn)

	return &DelegationClient{
		conn:   conn,
		client: client,
	}, nil
}

// RedeemDelegation sends a delegation to the gRPC service for redemption.
// This method handles the communication with the delegation service and returns
// the transaction hash if successful.
//
// Parameters:
//   - ctx: Context for the request, which can include timeout or cancellation
//   - delegationData: The delegation data to be redeemed, typically as JSON bytes
//
// Returns:
//   - The transaction hash as a string
//   - Error if the redemption failed or the service returned an error
func (c *DelegationClient) RedeemDelegation(ctx context.Context, delegationData []byte) (string, error) {
	// Set a timeout for the gRPC call if not already set in context
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Create the redemption request
	req := &proto.RedeemDelegationRequest{
		DelegationData: delegationData,
	}

	// Call the gRPC service
	resp, err := c.client.RedeemDelegation(timeoutCtx, req)
	if err != nil {
		// Extract detailed error information from gRPC status
		st, _ := status.FromError(err)
		return "", fmt.Errorf("delegation redemption failed with code %s: %s", st.Code(), st.Message())
	}

	// Check if the service reported success
	if !resp.Success {
		return "", fmt.Errorf("delegation redemption failed: %s", resp.ErrorMessage)
	}

	// Return transaction hash
	return resp.TransactionHash, nil
}

// RedeemDelegationDirectly sends a redemption request to the gRPC service
// and returns the transaction hash or an error
//
// Parameters:
//   - ctx: Context for the request
//   - delegationData: The delegation data as a byte array
//
// Returns:
//   - The transaction hash as a string
//   - Error if the redemption failed
func (c *DelegationClient) RedeemDelegationDirectly(ctx context.Context, delegationData []byte) (string, error) {
	log.Printf("Attempting to redeem delegation, data size: %d bytes", len(delegationData))

	// Call the client to redeem the delegation
	txHash, err := c.RedeemDelegation(ctx, delegationData)
	if err != nil {
		log.Printf("Delegation redemption failed: %v", err)
		return "", fmt.Errorf("delegation redemption failed: %w", err)
	}

	log.Printf("Delegation successfully redeemed, tx hash: %s", txHash)
	return txHash, nil
}

// GetDelegationForSubscription retrieves delegation data for a subscription
// TODO: Replace this with actual database query implementation when integrating with a database
func (c *DelegationClient) GetDelegationForSubscription(ctx context.Context, subscriptionID uuid.UUID) (*DelegationData, error) {
	// Mock implementation - replace with real database query
	// This would be implemented in your database queries file
	return &DelegationData{
		Delegate:  "0x1234567890123456789012345678901234567890",
		Delegator: "0x0987654321098765432109876543210987654321",
		Authority: "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789",
		Caveats:   json.RawMessage("[]"),
		Salt:      "0x123456789",
		Signature: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
	}, nil
}

// Close closes the gRPC connection. This should be called when the client
// is no longer needed to free up resources.
func (c *DelegationClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
