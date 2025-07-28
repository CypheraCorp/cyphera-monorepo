package responses

// UserResponse represents the standardized API response for user operations
type UserResponse struct {
	ID                 string                 `json:"id"`
	Object             string                 `json:"object"`
	Web3AuthID         string                 `json:"web3auth_id,omitempty"`
	Verifier           string                 `json:"verifier,omitempty"`
	VerifierID         string                 `json:"verifier_id,omitempty"`
	Email              string                 `json:"email"`
	FirstName          string                 `json:"first_name,omitempty"`
	LastName           string                 `json:"last_name,omitempty"`
	AddressLine1       string                 `json:"address_line_1,omitempty"`
	AddressLine2       string                 `json:"address_line_2,omitempty"`
	City               string                 `json:"city,omitempty"`
	StateRegion        string                 `json:"state_region,omitempty"`
	PostalCode         string                 `json:"postal_code,omitempty"`
	Country            string                 `json:"country,omitempty"`
	DisplayName        string                 `json:"display_name,omitempty"`
	PictureURL         string                 `json:"picture_url,omitempty"`
	Phone              string                 `json:"phone,omitempty"`
	Timezone           string                 `json:"timezone,omitempty"`
	Locale             string                 `json:"locale,omitempty"`
	EmailVerified      bool                   `json:"email_verified"`
	TwoFactorEnabled   bool                   `json:"two_factor_enabled"`
	FinishedOnboarding bool                   `json:"finished_onboarding"`
	Status             string                 `json:"status"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          int64                  `json:"created_at"`
	UpdatedAt          int64                  `json:"updated_at"`
}

// UserAccountResponse represents a user's relationship with an account
type UserAccountResponse struct {
	UserResponse
	AccountName string `json:"account_name"`
	Role        string `json:"role"`
	IsOwner     bool   `json:"is_owner"`
}
