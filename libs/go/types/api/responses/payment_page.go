package responses

import "time"

// PaymentPageDataResponse represents the data needed to render a payment page
type PaymentPageDataResponse struct {
	PaymentLink       PaymentLinkData     `json:"payment_link"`
	Product           *ProductData        `json:"product,omitempty"`
	Price             *PriceData          `json:"price,omitempty"`
	Workspace         WorkspaceData       `json:"workspace"`
	SupportedNetworks []NetworkData       `json:"supported_networks"`
	AcceptedTokens    []AcceptedTokenData `json:"accepted_tokens"`
	GasSponsorship    *GasSponsorshipData `json:"gas_sponsorship,omitempty"`
}

// PaymentLinkData represents payment link information for the payment page
type PaymentLinkData struct {
	ID              string     `json:"id"`
	Slug            string     `json:"slug"`
	Status          string     `json:"status"`
	AmountCents     *int64     `json:"amount_cents,omitempty"`
	Currency        string     `json:"currency"`
	PaymentType     string     `json:"payment_type"`
	CollectEmail    bool       `json:"collect_email"`
	CollectShipping bool       `json:"collect_shipping"`
	CollectName     bool       `json:"collect_name"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	MaxUses         *int32     `json:"max_uses,omitempty"`
	UsedCount       int32      `json:"used_count"`
	QRCodeData      *string    `json:"qr_code_data,omitempty"`
}

// ProductData represents product information for the payment page
type ProductData struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ImageURL    string `json:"image_url,omitempty"`
}

// PriceData represents price information for the payment page
type PriceData struct {
	ID                  string `json:"id"`
	UnitAmountInPennies int32  `json:"unit_amount_in_pennies"`
	Currency            string `json:"currency"`
	Type                string `json:"type"`
	IntervalType        string `json:"interval_type,omitempty"`
	IntervalCount       int32  `json:"interval_count,omitempty"`
}

// WorkspaceData represents workspace information for the payment page
type WorkspaceData struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	LogoURL    string `json:"logo_url,omitempty"`
	BrandColor string `json:"brand_color,omitempty"`
}

// NetworkData represents supported network information
type NetworkData struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ChainID   string `json:"chain_id"`
	Type      string `json:"type"`
	IsTestnet bool   `json:"is_testnet"`
}

// AcceptedTokenData represents accepted token information
type AcceptedTokenData struct {
	ID              string `json:"id"`
	Symbol          string `json:"symbol"`
	Name            string `json:"name"`
	NetworkID       string `json:"network_id"`
	NetworkName     string `json:"network_name"`
	ContractAddress string `json:"contract_address"`
	Decimals        int32  `json:"decimals"`
}

// GasSponsorshipData represents gas sponsorship information
type GasSponsorshipData struct {
	IsSponsored  bool   `json:"is_sponsored"`
	SponsorType  string `json:"sponsor_type,omitempty"`
	CoverageType string `json:"coverage_type,omitempty"`
}

// PaymentIntentResponse represents the response for a payment intent
type PaymentIntentResponse struct {
	IntentID         string                 `json:"intent_id"`
	Status           string                 `json:"status"`
	AmountCents      int64                  `json:"amount_cents"`
	Currency         string                 `json:"currency"`
	NetworkID        string                 `json:"network_id"`
	TokenID          string                 `json:"token_id"`
	RecipientAddress string                 `json:"recipient_address"`
	ExpiresAt        *time.Time             `json:"expires_at,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}
