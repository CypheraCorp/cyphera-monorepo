package helpers

import (
	"encoding/json"
	"log"

	"github.com/cyphera/cyphera-api/libs/go/db"
)

// CustomerResponse represents the standardized API response for customer operations
type CustomerResponse struct {
	ID                 string                 `json:"id"`
	Object             string                 `json:"object"`
	ExternalID         string                 `json:"external_id,omitempty"`
	Email              string                 `json:"email"`
	Name               string                 `json:"name,omitempty"`
	Phone              string                 `json:"phone,omitempty"`
	Description        string                 `json:"description,omitempty"`
	FinishedOnboarding bool                   `json:"finished_onboarding"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          int64                  `json:"created_at"`
	UpdatedAt          int64                  `json:"updated_at"`
}

// CustomerWalletResponse represents the customer wallet response for the sign-in/register API
type CustomerWalletResponse struct {
	ID            string                 `json:"id"`
	Object        string                 `json:"object"`
	CustomerID    string                 `json:"customer_id"`
	WalletAddress string                 `json:"wallet_address"`
	NetworkType   string                 `json:"network_type"`
	Nickname      string                 `json:"nickname,omitempty"`
	ENS           string                 `json:"ens,omitempty"`
	IsPrimary     bool                   `json:"is_primary"`
	Verified      bool                   `json:"verified"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt     int64                  `json:"created_at"`
	UpdatedAt     int64                  `json:"updated_at"`
}

// CustomerDetailsResponse represents the response for customer sign-in/register
type CustomerDetailsResponse struct {
	Customer CustomerResponse       `json:"customer"`
	Wallet   CustomerWalletResponse `json:"wallet,omitempty"`
}

// ToCustomerResponse converts database model to API response
func ToCustomerResponse(c db.Customer) CustomerResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(c.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling customer metadata: %v", err)
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	return CustomerResponse{
		ID:                 c.ID.String(),
		Object:             "customer",
		ExternalID:         c.ExternalID.String,
		Email:              c.Email.String,
		Name:               c.Name.String,
		Phone:              c.Phone.String,
		Description:        c.Description.String,
		FinishedOnboarding: c.FinishedOnboarding.Bool,
		Metadata:           metadata,
		CreatedAt:          c.CreatedAt.Time.Unix(),
		UpdatedAt:          c.UpdatedAt.Time.Unix(),
	}
}

// ToCustomerWalletResponse converts database CustomerWallet to API response
func ToCustomerWalletResponse(w db.CustomerWallet) CustomerWalletResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(w.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling wallet metadata: %v", err)
		metadata = make(map[string]interface{})
	}

	return CustomerWalletResponse{
		ID:            w.ID.String(),
		Object:        "customer_wallet",
		CustomerID:    w.CustomerID.String(),
		WalletAddress: w.WalletAddress,
		NetworkType:   string(w.NetworkType),
		Nickname:      w.Nickname.String,
		ENS:           w.Ens.String,
		IsPrimary:     w.IsPrimary.Bool,
		Verified:      w.Verified.Bool,
		Metadata:      metadata,
		CreatedAt:     w.CreatedAt.Time.Unix(),
		UpdatedAt:     w.UpdatedAt.Time.Unix(),
	}
}

// ToCustomerDetailsResponse creates a customer details response
func ToCustomerDetailsResponse(customer db.Customer, wallet *db.CustomerWallet) CustomerDetailsResponse {
	response := CustomerDetailsResponse{
		Customer: ToCustomerResponse(customer),
	}

	if wallet != nil {
		response.Wallet = ToCustomerWalletResponse(*wallet)
	}

	return response
}
