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

// mockDelegationService implements the DelegationService interface for testing
type mockDelegationService struct {
	proto.UnimplementedDelegationServiceServer
}

// RedeemDelegation implements the DelegationService interface
func (m *mockDelegationService) RedeemDelegation(ctx context.Context, req *proto.RedeemDelegationRequest) (*proto.RedeemDelegationResponse, error) {
	// Validate the request
	if len(req.Signature) == 0 {
		return &proto.RedeemDelegationResponse{
			Success:      false,
			ErrorMessage: "empty signature",
		}, nil
	}

	// Parse the delegation
	var delegation Delegation
	err := json.Unmarshal(req.Signature, &delegation)
	if err != nil {
		return &proto.RedeemDelegationResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to parse delegation: %v", err),
		}, nil
	}

	// Validate delegator address
	if !isValidEthereumAddress(delegation.Delegator) {
		return &proto.RedeemDelegationResponse{
			Success:      false,
			ErrorMessage: "Invalid delegator address format: must be a valid Ethereum address (0x + 40 hex chars)",
		}, nil
	}

	// Validate delegate address
	if !isValidEthereumAddress(delegation.Delegate) {
		return &proto.RedeemDelegationResponse{
			Success:      false,
			ErrorMessage: "Invalid delegate address format: must be a valid Ethereum address (0x + 40 hex chars)",
		}, nil
	}

	// Mock successful response
	return &proto.RedeemDelegationResponse{
		Success:         true,
		TransactionHash: "0xmocktransactionhash",
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
func setupMockServer(t *testing.T) (*grpc.Server, string, string, string) {
	// Create a listener on a random port
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
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

	delegator := "0x1234567890123456789012345678901234567890"
	delegate := "0x0987654321098765432109876543210987654321"

	return grpcServer, lis.Addr().String(), delegator, delegate
}

// createSampleDelegation creates a sample delegation for testing
func createSampleDelegation(delegator, delegate string) Delegation {
	return Delegation{
		Delegator: delegator,
		Delegate:  delegate,
		Signature: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
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
}

// TestCreateDelegation tests that valid delegation JSON can be created
func TestCreateDelegation(t *testing.T) {
	delegation := Delegation{
		Delegator: "0x1234567890123456789012345678901234567890",
		Delegate:  "0x0987654321098765432109876543210987654321",
		Signature: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
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

// TestRedeemDelegation tests the successful redemption of a delegation
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
	delegation := createSampleDelegation(
		"0x1234567890123456789012345678901234567890",
		"0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
	)

	// Marshal the delegation to JSON
	delegationJSON, err := json.Marshal(delegation)
	if err != nil {
		t.Fatalf("Failed to marshal delegation to JSON: %v", err)
	}

	// Call the gRPC service to redeem the delegation
	resp, err := client.RedeemDelegation(ctx, &proto.RedeemDelegationRequest{
		Signature:            delegationJSON,
		MerchantAddress:      "0x1234567890123456789012345678901234567890",
		TokenContractAddress: "0x1234567890123456789012345678901234567890",
		Price:                "1000000",
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
		Signature:            []byte{},
		MerchantAddress:      "0x0000000000000000000000000000000000000000",
		TokenContractAddress: "0x0000000000000000000000000000000000000000",
		Price:                "0",
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

// TestRedeemInvalidDelegatorAddress tests the error handling for an invalid delegator address
func TestRedeemInvalidDelegatorAddress(t *testing.T) {
	// Start a mock server
	grpcServer, serverAddr, _, _ := setupMockServer(t)
	defer grpcServer.GracefulStop()

	// Create a client
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
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
		Signature:            delegationJSON,
		MerchantAddress:      "0x1234567890123456789012345678901234567890",
		TokenContractAddress: "0x1234567890123456789012345678901234567890",
		Price:                "1000000",
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
	grpcServer, serverAddr, _, _ := setupMockServer(t)
	defer grpcServer.GracefulStop()

	// Create a client
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
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
		Signature:            delegationJSON,
		MerchantAddress:      "0x1234567890123456789012345678901234567890",
		TokenContractAddress: "0x1234567890123456789012345678901234567890",
		Price:                "1000000",
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
	grpcServer, serverAddr, _, _ := setupMockServer(t)
	defer grpcServer.GracefulStop()

	// Create a client
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
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
		Signature:            delegationJSON,
		MerchantAddress:      "0x1234567890123456789012345678901234567890",
		TokenContractAddress: "0x1234567890123456789012345678901234567890",
		Price:                "1000000",
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

// MockDelegationServiceClient implements the proto.DelegationServiceClient interface for testing
type MockDelegationServiceClient struct {
	proto.DelegationServiceClient
	RedeemDelegationFunc func(ctx context.Context, in *proto.RedeemDelegationRequest, opts ...grpc.CallOption) (*proto.RedeemDelegationResponse, error)
}

func (m *MockDelegationServiceClient) RedeemDelegation(ctx context.Context, in *proto.RedeemDelegationRequest, opts ...grpc.CallOption) (*proto.RedeemDelegationResponse, error) {
	if m.RedeemDelegationFunc != nil {
		return m.RedeemDelegationFunc(ctx, in, opts...)
	}
	return &proto.RedeemDelegationResponse{
		TransactionHash: "0xmocktransactionhash",
		Success:         true,
		ErrorMessage:    "",
	}, nil
}
