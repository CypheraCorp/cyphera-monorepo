package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"cyphera-api/internal/client"
)

// Sample delegation JSON structure based on the expected format by the Node.js server
type Delegation struct {
	Delegator string   `json:"delegator"`
	Delegate  string   `json:"delegate"`
	Signature string   `json:"signature"`
	Expiry    int64    `json:"expiry"`
	Caveats   []string `json:"caveats"`
	Salt      string   `json:"salt"`
	Authority struct {
		Scheme    string `json:"scheme"`
		Signature string `json:"signature"`
		Signer    string `json:"signer"`
	} `json:"authority"`
}

// RedeemDelegationRequest contains the data needed to redeem a delegation
type RedeemDelegationRequest struct {
	DelegationData string `json:"delegationData"`
}

// RedeemDelegationResponse is the response for a delegation redemption
type RedeemDelegationResponse struct {
	Success         bool   `json:"success"`
	TransactionHash string `json:"transactionHash,omitempty"`
	Error           string `json:"error,omitempty"`
}

// DelegationService handles delegation-related operations
type DelegationService struct {
	delegationClient *client.DelegationClient
}

// NewDelegationService creates a new DelegationService
func NewDelegationService() (*DelegationService, error) {
	// Initialize the delegation client
	delegationClient, err := client.NewDelegationClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create delegation client: %w", err)
	}

	return &DelegationService{
		delegationClient: delegationClient,
	}, nil
}

// RedeemDelegationHandler is an HTTP handler for redeeming delegations
func (s *DelegationService) RedeemDelegationHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req RedeemDelegationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Decode the delegation data
	delegationData := []byte(req.DelegationData)
	if len(delegationData) == 0 {
		http.Error(w, "Delegation data is required", http.StatusBadRequest)
		return
	}

	// Call the delegation client to redeem the delegation
	txHash, err := s.delegationClient.RedeemDelegationDirectly(r.Context(), delegationData)
	if err != nil {
		log.Printf("Error redeeming delegation: %v", err)
		sendJSONResponse(w, RedeemDelegationResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to redeem delegation: %v", err),
		}, http.StatusInternalServerError)
		return
	}

	// Return success response
	sendJSONResponse(w, RedeemDelegationResponse{
		Success:         true,
		TransactionHash: txHash,
	}, http.StatusOK)
}

// Helper function to send JSON responses
func sendJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func main() {
	// Define and parse command line flags
	serverMode := flag.Bool("server", false, "Run in server mode with HTTP API")
	serverPort := flag.String("port", "8080", "HTTP server port (when in server mode)")
	delegatorFlag := flag.String("delegator", "0x1234567890123456789012345678901234567890", "Delegator address")
	delegateFlag := flag.String("delegate", "0x0987654321098765432109876543210987654321", "Delegate address")
	signatureFlag := flag.String("signature", "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", "Delegation signature")
	expiryFlag := flag.Int64("expiry", time.Now().Unix()+3600, "Expiry timestamp (default: 1 hour from now)")
	saltFlag := flag.String("salt", "0x123456789", "Delegation salt")
	verboseFlag := flag.Bool("verbose", false, "Enable verbose output")
	flag.Parse()

	// Initialize the delegation service
	delegationService, err := NewDelegationService()
	if err != nil {
		log.Fatalf("Failed to initialize delegation service: %v", err)
	}

	// Handle server mode
	if *serverMode {
		log.Printf("Starting HTTP server on port %s...", *serverPort)

		// Register the delegation redemption endpoint
		http.HandleFunc("/api/delegations/redeem", delegationService.RedeemDelegationHandler)

		// Register a health check endpoint
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		// Start the server
		log.Printf("Server is ready to accept requests at http://localhost:%s/api/delegations/redeem", *serverPort)
		log.Fatal(http.ListenAndServe(":"+*serverPort, nil))
		return
	}

	// CLI mode - Create a sample delegation
	delegation := Delegation{
		Delegator: *delegatorFlag,
		Delegate:  *delegateFlag,
		Signature: *signatureFlag,
		Expiry:    *expiryFlag,
		Caveats:   []string{},
		Salt:      *saltFlag,
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

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Close the client when done
	defer delegationService.delegationClient.Close()

	// Notify about test start
	log.Println("Starting delegation integration test...")
	log.Printf("Using delegator: %s", delegation.Delegator)
	log.Printf("Using delegate: %s", delegation.Delegate)
	log.Printf("Delegation expires at: %d (%s)", delegation.Expiry, time.Unix(delegation.Expiry, 0).Format(time.RFC3339))

	// Convert the delegation to JSON
	delegationJSON, err := json.Marshal(delegation)
	if err != nil {
		log.Fatalf("Failed to marshal delegation to JSON: %v", err)
	}

	if *verboseFlag {
		log.Printf("Delegation JSON: %s", string(delegationJSON))
	}

	// Call the gRPC service to redeem the delegation
	log.Println("Sending delegation to gRPC service...")
	txHash, err := delegationService.delegationClient.RedeemDelegation(ctx, delegationJSON)
	if err != nil {
		log.Fatalf("Delegation redemption failed: %v", err)
	}

	// Print the result
	log.Printf("Delegation successfully redeemed! Transaction hash: %s", txHash)

	// Exit successfully
	os.Exit(0)
}
