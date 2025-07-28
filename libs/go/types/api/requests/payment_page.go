package requests

// PaymentIntentRequest represents a request to create a payment intent
type PaymentIntentRequest struct {
	CustomerEmail   string                 `json:"customer_email" binding:"required,email"`
	CustomerName    string                 `json:"customer_name"`
	WalletAddress   string                 `json:"wallet_address" binding:"required"`
	NetworkID       string                 `json:"network_id" binding:"required"`
	TokenID         string                 `json:"token_id" binding:"required"`
	ShippingAddress *ShippingAddressInput  `json:"shipping_address,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ShippingAddressInput represents shipping address input
type ShippingAddressInput struct {
	Line1      string `json:"line1" binding:"required"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city" binding:"required"`
	State      string `json:"state" binding:"required"`
	PostalCode string `json:"postal_code" binding:"required"`
	Country    string `json:"country" binding:"required"`
}
