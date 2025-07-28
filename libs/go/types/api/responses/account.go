package responses

// AccountResponse represents the standardized API response for account operations
type AccountResponse struct {
	ID                 string                 `json:"id"`
	Object             string                 `json:"object"`
	Name               string                 `json:"name"`
	AccountType        string                 `json:"account_type"`
	BusinessName       string                 `json:"business_name,omitempty"`
	BusinessType       string                 `json:"business_type,omitempty"`
	WebsiteURL         string                 `json:"website_url,omitempty"`
	SupportEmail       string                 `json:"support_email,omitempty"`
	SupportPhone       string                 `json:"support_phone,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	FinishedOnboarding bool                   `json:"finished_onboarding"`
	CreatedAt          int64                  `json:"created_at"`
	UpdatedAt          int64                  `json:"updated_at"`
	Workspaces         []WorkspaceResponse    `json:"workspaces,omitempty"`
}

// AccountDetailsResponse represents detailed account information with user and wallets
type AccountDetailsResponse struct {
	Account AccountResponse  `json:"account"`
	User    UserResponse     `json:"user,omitempty"`
	Wallets []WalletResponse `json:"wallets,omitempty"`
}

// AccountAccessResponse represents account access information
type AccountAccessResponse struct {
	AccountID string `json:"account_id"`
	HasAuth   bool   `json:"has_auth"`
}

// CircleWalletData represents Circle-specific wallet data
type CircleWalletData struct {
	CircleWalletID string `json:"circle_wallet_id"`
	CircleUserID   string `json:"circle_user_id"`
	ChainID        int32  `json:"chain_id"`
	State          string `json:"state"`
}

// WalletResponse represents wallet information in API responses
type WalletResponse struct {
	ID            string                 `json:"id"`
	Object        string                 `json:"object"`
	WorkspaceID   string                 `json:"workspace_id"`
	WalletType    string                 `json:"wallet_type"` // 'wallet' or 'circle_wallet'
	WalletAddress string                 `json:"wallet_address"`
	NetworkType   string                 `json:"network_type"`
	NetworkID     string                 `json:"network_id,omitempty"`
	Nickname      string                 `json:"nickname,omitempty"`
	ENS           string                 `json:"ens,omitempty"`
	IsPrimary     bool                   `json:"is_primary"`
	Verified      bool                   `json:"verified"`
	LastUsedAt    *int64                 `json:"last_used_at,omitempty"`
	Active        bool                   `json:"active"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CircleData    *CircleWalletData      `json:"circle_data,omitempty"` // Only present for circle wallets
	CreatedAt     int64                  `json:"created_at"`
	UpdatedAt     int64                  `json:"updated_at"`
}

// WalletListResponse represents a list of wallets
type WalletListResponse struct {
	Object  string           `json:"object"`
	Data    []WalletResponse `json:"data"`
	HasMore bool             `json:"has_more,omitempty"`
	Total   int64            `json:"total,omitempty"`
}
