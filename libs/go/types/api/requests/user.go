package requests

import "github.com/google/uuid"

// CreateUserRequest represents the request body for creating a user
type CreateUserRequest struct {
	Web3AuthID     string                 `json:"web3auth_id,omitempty"` // Web3Auth user ID
	Verifier       string                 `json:"verifier,omitempty"`    // Login method (google, discord, etc.)
	VerifierID     string                 `json:"verifier_id,omitempty"` // ID from the verifier
	Email          string                 `json:"email" binding:"required,email"`
	AccountID      uuid.UUID              `json:"account_id" binding:"required"`
	Role           string                 `json:"role" binding:"required,oneof=admin support developer"`
	IsAccountOwner bool                   `json:"is_account_owner"`
	FirstName      string                 `json:"first_name,omitempty"`
	LastName       string                 `json:"last_name,omitempty"`
	AddressLine1   string                 `json:"address_line_1,omitempty"`
	AddressLine2   string                 `json:"address_line_2,omitempty"`
	City           string                 `json:"city,omitempty"`
	StateRegion    string                 `json:"state_region,omitempty"`
	PostalCode     string                 `json:"postal_code,omitempty"`
	Country        string                 `json:"country,omitempty"`
	DisplayName    string                 `json:"display_name,omitempty"`
	PictureURL     string                 `json:"picture_url,omitempty"`
	Phone          string                 `json:"phone,omitempty"`
	Timezone       string                 `json:"timezone,omitempty"`
	Locale         string                 `json:"locale,omitempty"`
	EmailVerified  bool                   `json:"email_verified"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	Email            string                 `json:"email,omitempty"`
	FirstName        string                 `json:"first_name,omitempty"`
	LastName         string                 `json:"last_name,omitempty"`
	AddressLine1     string                 `json:"address_line_1,omitempty"`
	AddressLine2     string                 `json:"address_line_2,omitempty"`
	City             string                 `json:"city,omitempty"`
	StateRegion      string                 `json:"state_region,omitempty"`
	PostalCode       string                 `json:"postal_code,omitempty"`
	Country          string                 `json:"country,omitempty"`
	DisplayName      string                 `json:"display_name,omitempty"`
	PictureURL       string                 `json:"picture_url,omitempty"`
	Phone            string                 `json:"phone,omitempty"`
	Timezone         string                 `json:"timezone,omitempty"`
	Locale           string                 `json:"locale,omitempty"`
	EmailVerified    *bool                  `json:"email_verified,omitempty"`
	TwoFactorEnabled *bool                  `json:"two_factor_enabled,omitempty"`
	Status           string                 `json:"status,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// AddUserToAccountRequest represents the request to add a user to an account
type AddUserToAccountRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required,oneof=admin support developer"`
}
