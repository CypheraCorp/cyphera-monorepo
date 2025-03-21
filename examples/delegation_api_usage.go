package examples

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"cyphera-api/internal/handlers"
)

// This file demonstrates how to use the DelegationHandler in a Go API endpoint

// DelegationService handles delegation-related operations
type DelegationService struct {
	delegationHandler *handlers.DelegationHandler
	commonServices    *handlers.CommonServices
}

// NewDelegationService creates a new DelegationService
func NewDelegationService() (*DelegationService, error) {
	// Initialize common services (normally you would pass in DB, config, etc.)
	commonServices := &handlers.CommonServices{}

	// Initialize the delegation handler
	delegationHandler := handlers.NewDelegationHandler(commonServices)

	return &DelegationService{
		delegationHandler: delegationHandler,
		commonServices:    commonServices,
	}, nil
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

	// Call the delegation handler to redeem the delegation
	txHash, err := s.delegationHandler.RedeemDelegationDirectly(r.Context(), delegationData)
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

// ExampleRegisterDelegationHandler demonstrates how to register the handler in your HTTP server
func ExampleRegisterDelegationHandler() {
	delegationService, err := NewDelegationService()
	if err != nil {
		log.Fatalf("Failed to initialize delegation service: %v", err)
	}

	// Register the handler
	http.HandleFunc("/api/delegations/redeem", delegationService.RedeemDelegationHandler)

	// Start the server (in a real application)
	log.Println("Starting server on :8080")
	// If this were a real server, you would uncomment the following line:
	// if err := http.ListenAndServe(":8080", nil); err != nil {
	// 	log.Fatalf("Failed to start server: %v", err)
	// }
}
