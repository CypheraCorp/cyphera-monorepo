package responses

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

// ListCustomersResult represents the result of listing customers
type ListCustomersResult struct {
	Customers []CustomerResponse `json:"customers"`
	Total     int64              `json:"total"`
}

// ListWorkspaceCustomersResult represents the result of listing workspace customers
type ListWorkspaceCustomersResult struct {
	Customers []CustomerResponse `json:"customers"`
	Total     int64              `json:"total"`
}
