package requests

// CreateAccountRequest represents the request body for creating an account
type CreateAccountRequest struct {
	Name               string                 `json:"name" binding:"required"`
	AccountType        string                 `json:"account_type" binding:"required,oneof=admin merchant"`
	Description        string                 `json:"description,omitempty"`
	BusinessName       string                 `json:"business_name,omitempty"`
	BusinessType       string                 `json:"business_type,omitempty"`
	WebsiteURL         string                 `json:"website_url,omitempty"`
	SupportEmail       string                 `json:"support_email,omitempty"`
	SupportPhone       string                 `json:"support_phone,omitempty"`
	FinishedOnboarding bool                   `json:"finished_onboarding,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	// Web3Auth embedded wallet data to be created during registration
	WalletData *EmbeddedWalletRequest `json:"wallet_data,omitempty"`
}

// EmbeddedWalletRequest represents the request body for creating an embedded wallet during account creation
type EmbeddedWalletRequest struct {
	WalletType    string                 `json:"wallet_type" binding:"required"` // 'web3auth', 'wallet', or 'circle_wallet'
	WalletAddress string                 `json:"wallet_address" binding:"required"`
	NetworkType   string                 `json:"network_type" binding:"required"`
	Nickname      string                 `json:"nickname,omitempty"`
	ENS           string                 `json:"ens,omitempty"`
	IsPrimary     bool                   `json:"is_primary"`
	Verified      bool                   `json:"verified"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	// Circle wallet specific fields
	CircleUserID   string `json:"circle_user_id,omitempty"`   // Only for circle wallets
	CircleWalletID string `json:"circle_wallet_id,omitempty"` // Only for circle wallets
	ChainID        int32  `json:"chain_id,omitempty"`         // Only for circle wallets
	State          string `json:"state,omitempty"`            // Only for circle wallets
}

// UpdateAccountRequest represents the request body for updating an account
type UpdateAccountRequest struct {
	Name               string                 `json:"name,omitempty"`
	Description        string                 `json:"description,omitempty"`
	BusinessName       string                 `json:"business_name,omitempty"`
	BusinessType       string                 `json:"business_type,omitempty"`
	WebsiteURL         string                 `json:"website_url,omitempty"`
	SupportEmail       string                 `json:"support_email,omitempty"`
	SupportPhone       string                 `json:"support_phone,omitempty"`
	AccountType        string                 `json:"account_type,omitempty" binding:"omitempty,oneof=admin merchant"`
	FinishedOnboarding bool                   `json:"finished_onboarding,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// OnboardAccountRequest represents the request body for onboarding an account
type OnboardAccountRequest struct {
	FirstName     string `json:"first_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
	AddressLine1  string `json:"address_line1"`
	AddressLine2  string `json:"address_line2"`
	City          string `json:"city"`
	State         string `json:"state"`
	PostalCode    string `json:"postal_code"`
	Country       string `json:"country"`
	TaxID         string `json:"tax_id,omitempty"`
	BusinessPhone string `json:"business_phone,omitempty"`
	WalletAddress string `json:"wallet_address,omitempty"`
}
