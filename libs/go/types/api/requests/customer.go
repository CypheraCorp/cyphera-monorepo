package requests

// CreateCustomerRequest represents the request body for creating a customer
type CreateCustomerRequest struct {
	ExternalID         string                 `json:"external_id,omitempty"`
	Email              string                 `json:"email" binding:"required,email"`
	Name               string                 `json:"name,omitempty"`
	Phone              string                 `json:"phone,omitempty"`
	Description        string                 `json:"description,omitempty"`
	FinishedOnboarding *bool                  `json:"finished_onboarding,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateCustomerRequest represents the request body for updating a customer
type UpdateCustomerRequest struct {
	ExternalID         *string                `json:"external_id,omitempty"`
	Email              *string                `json:"email,omitempty" binding:"omitempty,email"`
	Name               *string                `json:"name,omitempty"`
	Phone              *string                `json:"phone,omitempty"`
	Description        *string                `json:"description,omitempty"`
	FinishedOnboarding *bool                  `json:"finished_onboarding,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// SignInRegisterCustomerRequest represents the request body for customer sign-in/register
type SignInRegisterCustomerRequest struct {
	Email    string                 `json:"email" binding:"required,email"`
	Name     string                 `json:"name,omitempty"`
	Phone    string                 `json:"phone,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// Web3Auth wallet data to be created during registration
	WalletData *CustomerWalletRequest `json:"wallet_data,omitempty"`
}

// CustomerWalletRequest represents wallet data for customer registration
type CustomerWalletRequest struct {
	WalletAddress string                 `json:"wallet_address" binding:"required"`
	NetworkType   string                 `json:"network_type" binding:"required"`
	Nickname      string                 `json:"nickname,omitempty"`
	ENS           string                 `json:"ens,omitempty"`
	IsPrimary     bool                   `json:"is_primary"`
	Verified      bool                   `json:"verified"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateCustomerOnboardingStatusRequest represents the request to update customer onboarding status
type UpdateCustomerOnboardingStatusRequest struct {
	FinishedOnboarding bool `json:"finished_onboarding" binding:"required"`
}
