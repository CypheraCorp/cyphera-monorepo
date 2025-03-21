package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"testing"
	"time"

	"cyphera-api/internal/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

// mockDelegationService implements a mock gRPC delegation service for testing
type mockDelegationService struct {
	proto.UnimplementedDelegationServiceServer
}

// RedeemDelegation implements the DelegationService interface for testing
func (m *mockDelegationService) RedeemDelegation(ctx context.Context, req *proto.RedeemDelegationRequest) (*proto.RedeemDelegationResponse, error) {
	// Check if the delegation data is present
	if len(req.DelegationData) == 0 {
		return &proto.RedeemDelegationResponse{
			TransactionHash: "",
			Success:         false,
			ErrorMessage:    "Delegation data is empty or invalid",
		}, nil
	}

	// Try to parse the delegation
	var delegation Delegation
	err := json.Unmarshal(req.DelegationData, &delegation)
	if err != nil {
		return &proto.RedeemDelegationResponse{
			TransactionHash: "",
			Success:         false,
			ErrorMessage:    "Invalid delegation format: " + err.Error(),
		}, nil
	}

	// Validate the delegation
	if delegation.Delegator == "" {
		return &proto.RedeemDelegationResponse{
			TransactionHash: "",
			Success:         false,
			ErrorMessage:    "Invalid delegation: missing delegator",
		}, nil
	}

	// Validate the address format
	if !isValidEthereumAddress(delegation.Delegator) {
		return &proto.RedeemDelegationResponse{
			TransactionHash: "",
			Success:         false,
			ErrorMessage:    "Invalid delegator address format",
		}, nil
	}

	if !isValidEthereumAddress(delegation.Delegate) {
		return &proto.RedeemDelegationResponse{
			TransactionHash: "",
			Success:         false,
			ErrorMessage:    "Invalid delegate address format",
		}, nil
	}

	// Check if delegation has expired
	if delegation.Expiry > 0 {
		currentTime := time.Now().Unix()
		if delegation.Expiry < currentTime {
			return &proto.RedeemDelegationResponse{
				TransactionHash: "",
				Success:         false,
				ErrorMessage:    "Delegation is expired (expiry: " + fmt.Sprintf("%d", delegation.Expiry) + ", now: " + fmt.Sprintf("%d", currentTime) + ")",
			}, nil
		}
	}

	// All checks passed, return a mock transaction hash
	return &proto.RedeemDelegationResponse{
		TransactionHash: "0xmocktransactionhash",
		Success:         true,
		ErrorMessage:    "",
	}, nil
}

// isValidEthereumAddress checks if the provided string is a valid Ethereum address
func isValidEthereumAddress(address string) bool {
	// Simple check for the 0x prefix and length
	if len(address) != 42 {
		return false
	}

	if !strings.HasPrefix(address, "0x") {
		return false
	}

	// Check if the address contains only hex characters after the 0x prefix
	for _, c := range address[2:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}

// Setup the mock gRPC server
func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	proto.RegisterDelegationServiceServer(s, &mockDelegationService{})
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()
}

// Custom dialer for the mock gRPC server
func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

// setupMockServer creates a mock gRPC server for testing
func setupMockServer() (net.Listener, *grpc.Server, *mockDelegationService, error) {
	// Create a listener on a random port
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to listen: %v", err)
	}

	// Create a new server instance
	mockServer := &mockDelegationService{}

	// Create a new gRPC server
	grpcServer := grpc.NewServer()
	proto.RegisterDelegationServiceServer(grpcServer, mockServer)

	// Start the server in a goroutine
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	return lis, grpcServer, mockServer, nil
}

// TestCreateDelegation tests that valid delegation JSON can be created
func TestCreateDelegation(t *testing.T) {
	delegation := Delegation{
		Delegator: "0x1234567890123456789012345678901234567890",
		Delegate:  "0x0987654321098765432109876543210987654321",
		Signature: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		Expiry:    time.Now().Unix() + 3600,
		Caveats:   []string{},
		Salt:      "0x123456789",
		Authority: struct {
			Scheme    string `json:"scheme"`
			Signature string `json:"signature"`
			Signer    string `json:"signer"`
		}{
			Scheme:    "0x00",
			Signature: "0xsig",
			Signer:    "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789",
		},
	}

	delegationJSON, err := json.Marshal(delegation)
	if err != nil {
		t.Fatalf("Failed to marshal delegation to JSON: %v", err)
	}

	if len(delegationJSON) == 0 {
		t.Fatalf("Delegation JSON should not be empty")
	}
}

// TestRedeemDelegation tests the delegation redemption with a mock server
func TestRedeemDelegation(t *testing.T) {
	// Create a gRPC connection to the mock server
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	// Create the delegation client
	client := proto.NewDelegationServiceClient(conn)

	// Create a test delegation
	delegation := Delegation{
		Delegator: "0x1234567890123456789012345678901234567890",
		Delegate:  "0x0987654321098765432109876543210987654321",
		Signature: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		Expiry:    time.Now().Unix() + 3600,
		Caveats:   []string{},
		Salt:      "0x123456789",
		Authority: struct {
			Scheme    string `json:"scheme"`
			Signature string `json:"signature"`
			Signer    string `json:"signer"`
		}{
			Scheme:    "0x00",
			Signature: "0xsig",
			Signer:    "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789",
		},
	}

	// Convert the delegation to JSON
	delegationJSON, err := json.Marshal(delegation)
	if err != nil {
		t.Fatalf("Failed to marshal delegation to JSON: %v", err)
	}

	// Call the gRPC service to redeem the delegation
	resp, err := client.RedeemDelegation(ctx, &proto.RedeemDelegationRequest{
		DelegationData: delegationJSON,
	})

	if err != nil {
		t.Fatalf("RedeemDelegation failed: %v", err)
	}

	// Verify the response
	if !resp.Success {
		t.Errorf("Expected delegation redemption to succeed, but it failed with error: %s", resp.ErrorMessage)
	}

	if resp.TransactionHash != "0xmocktransactionhash" {
		t.Errorf("Expected transaction hash '0xmocktransactionhash', got '%s'", resp.TransactionHash)
	}
}

// TestRedeemEmptyDelegation tests error handling for empty delegation data
func TestRedeemEmptyDelegation(t *testing.T) {
	// Create a gRPC connection to the mock server
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	// Create the delegation client
	client := proto.NewDelegationServiceClient(conn)

	// Call the gRPC service with empty delegation data
	resp, err := client.RedeemDelegation(ctx, &proto.RedeemDelegationRequest{
		DelegationData: []byte{},
	})

	if err != nil {
		t.Fatalf("RedeemDelegation failed: %v", err)
	}

	// Verify the error response
	if resp.Success {
		t.Error("Expected delegation redemption to fail for empty data, but it succeeded")
	}

	if resp.ErrorMessage == "" {
		t.Error("Expected error message for empty delegation data, but got empty string")
	}
}

// TestRedeemExpiredDelegation tests the error handling for expired delegation
func TestRedeemExpiredDelegation(t *testing.T) {
	// Start a mock server
	lis, grpcServer, _, err := setupMockServer()
	if err != nil {
		t.Fatalf("Failed to setup mock server: %v", err)
	}
	defer lis.Close()
	defer grpcServer.GracefulStop()

	// Create a client
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()

	client := proto.NewDelegationServiceClient(conn)

	// Create a test delegation with expired timestamp (1 hour in the past)
	pastTime := time.Now().Add(-1 * time.Hour).Unix()
	delegation := Delegation{
		Delegator: "0x1234567890123456789012345678901234567890",
		Delegate:  "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
		Signature: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		Expiry:    pastTime,
		Salt:      "0x123456789",
		Caveats:   []string{},
		Authority: struct {
			Scheme    string `json:"scheme"`
			Signature string `json:"signature"`
			Signer    string `json:"signer"`
		}{
			Scheme:    "0x00",
			Signature: "0xsig",
			Signer:    "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789",
		},
	}

	// Add signature to make it valid
	delegationJSON, err := json.Marshal(delegation)
	if err != nil {
		t.Fatalf("Failed to marshal delegation: %v", err)
	}

	// Try to redeem the delegation
	resp, err := client.RedeemDelegation(context.Background(), &proto.RedeemDelegationRequest{
		DelegationData: delegationJSON,
	})

	// Redemption should fail because of expired timestamp
	if err != nil {
		t.Fatalf("Unexpected error calling RedeemDelegation: %v", err)
	}
	if resp.Success {
		t.Errorf("Expected redemption to fail due to expired timestamp, but it succeeded")
	}
	if !strings.Contains(resp.ErrorMessage, "expired") {
		t.Errorf("Expected error message to contain 'expired', got %s", resp.ErrorMessage)
	}
}

// TestRedeemInvalidDelegatorAddress tests the error handling for an invalid delegator address
func TestRedeemInvalidDelegatorAddress(t *testing.T) {
	// Start a mock server
	lis, grpcServer, _, err := setupMockServer()
	if err != nil {
		t.Fatalf("Failed to setup mock server: %v", err)
	}
	defer lis.Close()
	defer grpcServer.GracefulStop()

	// Create a client
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()

	client := proto.NewDelegationServiceClient(conn)

	// Create a test delegation with invalid delegator address
	delegation := Delegation{
		Delegator: "0xinvalid",                                  // Invalid address
		Delegate:  "0x1234567890123456789012345678901234567890", // Valid address
		Signature: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		Expiry:    time.Now().Add(1 * time.Hour).Unix(), // Future expiry
		Salt:      "0x123456789",
		Caveats:   []string{},
		Authority: struct {
			Scheme    string `json:"scheme"`
			Signature string `json:"signature"`
			Signer    string `json:"signer"`
		}{
			Scheme:    "0x00",
			Signature: "0xsig",
			Signer:    "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789",
		},
	}

	delegationJSON, err := json.Marshal(delegation)
	if err != nil {
		t.Fatalf("Failed to marshal delegation: %v", err)
	}

	// Create a timeout context for the request
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try to redeem the delegation
	resp, err := client.RedeemDelegation(ctx, &proto.RedeemDelegationRequest{
		DelegationData: delegationJSON,
	})

	// Redemption should fail because of invalid address
	if err != nil {
		t.Fatalf("Unexpected error calling RedeemDelegation: %v", err)
	}
	if resp.Success {
		t.Errorf("Expected redemption to fail due to invalid delegator address, but it succeeded")
	}
	if !strings.Contains(resp.ErrorMessage, "Invalid delegator address format") {
		t.Errorf("Expected error message to contain 'Invalid delegator address format', got %s", resp.ErrorMessage)
	}
}

// TestRedeemInvalidDelegateAddress tests the error handling for an invalid delegate address
func TestRedeemInvalidDelegateAddress(t *testing.T) {
	// Start a mock server
	lis, grpcServer, _, err := setupMockServer()
	if err != nil {
		t.Fatalf("Failed to setup mock server: %v", err)
	}
	defer lis.Close()
	defer grpcServer.GracefulStop()

	// Create a client
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()

	client := proto.NewDelegationServiceClient(conn)

	// Create a test delegation with invalid delegate address
	delegation := Delegation{
		Delegator: "0x1234567890123456789012345678901234567890", // Valid address
		Delegate:  "0xinvalid",                                  // Invalid address
		Signature: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		Expiry:    time.Now().Add(1 * time.Hour).Unix(), // Future expiry
		Salt:      "0x123456789",
		Caveats:   []string{},
		Authority: struct {
			Scheme    string `json:"scheme"`
			Signature string `json:"signature"`
			Signer    string `json:"signer"`
		}{
			Scheme:    "0x00",
			Signature: "0xsig",
			Signer:    "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789",
		},
	}

	delegationJSON, err := json.Marshal(delegation)
	if err != nil {
		t.Fatalf("Failed to marshal delegation: %v", err)
	}

	// Create a timeout context for the request
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try to redeem the delegation
	resp, err := client.RedeemDelegation(ctx, &proto.RedeemDelegationRequest{
		DelegationData: delegationJSON,
	})

	// Redemption should fail because of invalid address
	if err != nil {
		t.Fatalf("Unexpected error calling RedeemDelegation: %v", err)
	}
	if resp.Success {
		t.Errorf("Expected redemption to fail due to invalid delegate address, but it succeeded")
	}
	if !strings.Contains(resp.ErrorMessage, "Invalid delegate address format") {
		t.Errorf("Expected error message to contain 'Invalid delegate address format', got %s", resp.ErrorMessage)
	}
}

// TestRedeemNonHexAddress tests the error handling for non-hex characters in an address
func TestRedeemNonHexAddress(t *testing.T) {
	// Start a mock server
	lis, grpcServer, _, err := setupMockServer()
	if err != nil {
		t.Fatalf("Failed to setup mock server: %v", err)
	}
	defer lis.Close()
	defer grpcServer.GracefulStop()

	// Create a client
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial server: %v", err)
	}
	defer conn.Close()

	client := proto.NewDelegationServiceClient(conn)

	// Create a test delegation with non-hex delegator address
	delegation := Delegation{
		Delegator: "0xGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGGG", // Non-hex address
		Delegate:  "0x1234567890123456789012345678901234567890", // Valid address
		Signature: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		Expiry:    time.Now().Add(1 * time.Hour).Unix(), // Future expiry
		Salt:      "0x123456789",
		Caveats:   []string{},
		Authority: struct {
			Scheme    string `json:"scheme"`
			Signature string `json:"signature"`
			Signer    string `json:"signer"`
		}{
			Scheme:    "0x00",
			Signature: "0xsig",
			Signer:    "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789",
		},
	}

	delegationJSON, err := json.Marshal(delegation)
	if err != nil {
		t.Fatalf("Failed to marshal delegation: %v", err)
	}

	// Create a timeout context for the request
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Try to redeem the delegation
	resp, err := client.RedeemDelegation(ctx, &proto.RedeemDelegationRequest{
		DelegationData: delegationJSON,
	})

	// Redemption should fail because of invalid address
	if err != nil {
		t.Fatalf("Unexpected error calling RedeemDelegation: %v", err)
	}
	if resp.Success {
		t.Errorf("Expected redemption to fail due to non-hex address, but it succeeded")
	}
	if !strings.Contains(resp.ErrorMessage, "Invalid delegator address format") {
		t.Errorf("Expected error message to contain 'Invalid delegator address format', got %s", resp.ErrorMessage)
	}
}

// TestConnectionError tests the error handling for connection issues
func TestConnectionError(t *testing.T) {
	// Use a port that is likely to be unavailable
	invalidAddr := "localhost:65535"

	// Try to connect to an unavailable server
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use WithBlock() but with a timeout to avoid hanging
	conn, err := grpc.DialContext(ctx, invalidAddr, grpc.WithInsecure(), grpc.WithBlock())

	// We expect a connection error
	if err == nil {
		conn.Close()
		t.Fatalf("Expected connection error, but got none")
	}

	// Try to use our client wrapper with an invalid address
	_, err = NewDelegationClient(invalidAddr)
	if err == nil {
		t.Fatalf("Expected error creating client with invalid address, but got none")
	}
}

// NewDelegationClient creates a new delegation client with proper error handling
func NewDelegationClient(serverAddr string) (proto.DelegationServiceClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, serverAddr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to delegation server: %w", err)
	}

	return proto.NewDelegationServiceClient(conn), nil
}
