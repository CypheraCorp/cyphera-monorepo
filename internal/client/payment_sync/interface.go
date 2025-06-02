package payment_sync

import "context"

// Customer represents a customer entity.
type Customer struct {
	ID              string // Cyphera's internal ID, if available
	ExternalID      string // ID from the external platform
	Email           string
	Name            string
	Phone           string
	Metadata        map[string]string // For custom fields or additional platform-specific data
	BillingAddress  *Address          `json:"billing_address,omitempty"`
	ShippingAddress *Address          `json:"shipping_address,omitempty"`
	TaxIDs          []TaxID           `json:"tax_ids,omitempty"`
	PreferredLocale string            `json:"preferred_locale,omitempty"`
	// BillingAddress, ShippingAddress, etc. can be added here or as separate structs
}

// Address represents a physical address.
type Address struct {
	Line1      string `json:"line1,omitempty"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"` // Or Province/Region
	PostalCode string `json:"postal_code,omitempty"`
	Country    string `json:"country,omitempty"` // ISO 3166-1 alpha-2 code
}

// TaxID represents a customer's tax identification number.
type TaxID struct {
	Type  string `json:"type,omitempty"` // e.g., "eu_vat", "au_abn" (maps to Stripe's tax_id types)
	Value string `json:"value,omitempty"`
	// Country string `json:"country,omitempty"` // Optional: Country of the tax ID, if not implied by type
}

// Product represents a product or service offering.
type Product struct {
	ID          string
	ExternalID  string
	Name        string
	Description string
	Active      bool
	Type        string // e.g., "service", "good"
	Metadata    map[string]string
	TaxCode     string `json:"tax_code,omitempty"`
	UnitLabel   string `json:"unit_label,omitempty"`
	Shippable   bool   `json:"shippable,omitempty"`
}

// Price represents the pricing details for a product.
type Price struct {
	ID                    string
	ExternalID            string
	ProductID             string // External Product ID this price belongs to
	Active                bool
	Amount                int64  // In the smallest currency unit (e.g., cents)
	Currency              string // ISO currency code (e.g., "usd")
	Recurring             *RecurringInterval
	BillingScheme         string // e.g., "per_unit", "tiered"
	Type                  string // e.g., "one_time", "recurring"
	Metadata              map[string]string
	TaxBehavior           string             `json:"tax_behavior,omitempty"` // e.g., "inclusive", "exclusive", "unspecified"
	Tiers                 []PriceTier        `json:"tiers,omitempty"`
	TiersMode             string             `json:"tiers_mode,omitempty"` // e.g., "graduated", "volume"
	TransformQuantityData *TransformQuantity `json:"transform_quantity,omitempty"`
}

// PriceTier defines a tier for tiered pricing.
type PriceTier struct {
	UpTo       int64 `json:"up_to,omitempty"`       // For graduated/volume, null for flat fee for the tier
	UnitAmount int64 `json:"unit_amount,omitempty"` // Amount for this tier (per unit)
	FlatAmount int64 `json:"flat_amount,omitempty"` // Flat amount for this tier (for volume tiers)
}

// TransformQuantity defines rules for transforming usage quantity before applying price.
type TransformQuantity struct {
	DivideBy int64  `json:"divide_by,omitempty"`
	Round    string `json:"round,omitempty"` // e.g., "up", "down"
}

// RecurringInterval defines the billing frequency for a recurring price.
type RecurringInterval struct {
	Interval      string // e.g., "day", "week", "month", "year"
	IntervalCount int    // e.g., for "every 3 months", Interval="month", IntervalCount=3
}

// Subscription represents a customer's subscription to a product.
type Subscription struct {
	ID                     string
	ExternalID             string
	CustomerID             string // External Customer ID
	Status                 string // e.g., "active", "past_due", "canceled", "incomplete", "trialing"
	CurrentPeriodStart     int64  // Unix timestamp
	CurrentPeriodEnd       int64  // Unix timestamp
	TrialStartDate         int64  // Unix timestamp, if applicable
	TrialEndDate           int64  // Unix timestamp, if applicable
	CancelAtPeriodEnd      bool
	CanceledAt             int64  // Unix timestamp, if canceled
	EndedAt                int64  // Unix timestamp, if ended completely
	DefaultPaymentMethodID string // External ID of the default payment method
	Items                  []SubscriptionItem
	Metadata               map[string]string
	LatestInvoiceID        string   // External ID of the latest invoice for this subscription
	BillingCycleAnchor     int64    `json:"billing_cycle_anchor,omitempty"`
	CollectionMethod       string   `json:"collection_method,omitempty"` // e.g., "charge_automatically", "send_invoice"
	DefaultTaxRateIDs      []string `json:"default_tax_rate_ids,omitempty"`
}

// SubscriptionItem represents an item within a subscription (e.g., a specific plan or add-on).
type SubscriptionItem struct {
	ID         string
	ExternalID string
	PriceID    string // External Price ID
	Quantity   int
	Metadata   map[string]string
	TaxRateIDs []string `json:"tax_rate_ids,omitempty"`
}

// TaxAmount represents a tax amount breakdown.
type TaxAmount struct {
	Amount        int64  `json:"amount,omitempty"`         // The amount of tax, in the smallest currency unit.
	RateID        string `json:"rate_id,omitempty"`        // External ID of the TaxRate object applied.
	TaxableAmount int64  `json:"taxable_amount,omitempty"` // The amount on which tax was calculated.
	Inclusive     bool   `json:"inclusive,omitempty"`      // Whether this tax amount is inclusive in the total.
}

// Invoice represents a bill issued to a customer.
type Invoice struct {
	ID                 string
	ExternalID         string
	CustomerID         string // External Customer ID
	SubscriptionID     string // External Subscription ID (if applicable)
	Status             string // e.g., "draft", "open", "paid", "void", "uncollectible"
	CollectionMethod   string // e.g., "charge_automatically", "send_invoice"
	AmountDue          int64
	AmountPaid         int64
	AmountRemaining    int64
	Currency           string
	DueDate            int64  // Unix timestamp
	PaidAt             int64  // Unix timestamp, if paid
	InvoicePDF         string // URL to the PDF, if available
	HostedInvoiceURL   string // URL to a hosted invoice page, if available
	Lines              []InvoiceLineItem
	Metadata           map[string]string
	AttemptCount       int         // Number of payment attempts
	NextPaymentAttempt int64       // Unix timestamp for next attempt
	ChargeID           string      // External ID of the associated Charge
	PaymentIntentID    string      // External ID of the associated PaymentIntent/Charge
	Tax                int64       `json:"tax,omitempty"` // Total tax amount on the invoice.
	TotalTaxAmounts    []TaxAmount `json:"total_tax_amounts,omitempty"`
	BillingReason      string      `json:"billing_reason,omitempty"` // e.g., subscription_cycle, manual
	PaidOutOfBand      bool        `json:"paid_out_of_band,omitempty"`
}

// InvoiceLineItem represents a line item on an invoice.
type InvoiceLineItem struct {
	ID                 string
	ExternalID         string
	Description        string
	Amount             int64
	Quantity           int
	PriceID            string // External Price ID (if applicable)
	ProductID          string // External Product ID (if applicable)
	Period             *Period
	Type               string // e.g., "subscription", "invoiceitem"
	Proration          bool
	Metadata           map[string]string
	TaxAmounts         []TaxAmount `json:"tax_amounts,omitempty"`
	SubscriptionItemID string      `json:"subscription_item_id,omitempty"` // External ID of the generating ps.SubscriptionItem
}

// Period defines a start and end time for a line item or subscription period.
type Period struct {
	Start int64 // Unix timestamp
	End   int64 // Unix timestamp
}

// Transaction represents a financial transaction in our system.
// It's designed to be a canonical model that can represent crypto payments,
// links to fiat on/off ramps, or interactions with Stripe PaymentIntents if used.
type Transaction struct {
	ExternalID  string // Your system's unique ID for this transaction
	UserID      string // Your system's user ID
	Type        string // e.g., "one_time_crypto", "subscription_crypto", "off_ramp_start_crypto", "fiat_on_ramp_stripe_pi"
	Status      string // e.g., "pending_onchain", "confirmed_onchain", "provider_processing", "completed_fiat_settlement", "failed"
	Amount      int64  // For crypto, smallest unit (sats, wei). For fiat, smallest unit (cents).
	Currency    string // e.g., "BTC", "ETH", "USDC", "USD"
	CreatedAt   int64  // Unix timestamp
	UpdatedAt   int64  // Unix timestamp
	Description string // Optional user-facing description or system note
	Metadata    map[string]string

	// Crypto Specific (if Type involves crypto)
	SourceWalletAddress       string
	DestinationWalletAddress  string
	BlockchainTransactionHash string
	NetworkConfirmationCount  int

	// Off-Ramp / On-Ramp Provider Specific (e.g., Bridge.xyz, or Stripe for on-ramping)
	ProviderTransactionID string // e.g., Bridge ID, or Stripe PaymentIntent ID if Stripe is the provider for this leg
	ProviderType          string // e.g., "bridge_xyz", "stripe" (to know which provider this ID refers to)
	ProviderFee           int64  // Fee charged by the provider, in smallest unit of ProviderFiatCurrency
	ProviderFiatAmount    int64  // Fiat amount at the provider, smallest unit
	ProviderFiatCurrency  string // e.g., "USD"
	ProviderStatus        string // Granular status from the provider API

	// Links to other ps objects
	SubscriptionID string // If this transaction pays for a ps.Subscription
	InvoiceID      string // If this transaction pays for a ps.Invoice
	CustomerID     string // Link to ps.Customer
}

// ExternalAccount represents a bank account or other external payout method, typically for off-ramping.
type ExternalAccount struct {
	ExternalID        string // Your system's unique ID for this stored external account
	UserID            string
	Provider          string // e.g., "bridge_xyz" (or "stripe" if Stripe was used for direct payouts)
	ProviderAccountID string // The ID of this account at the external provider
	Type              string // e.g., "bank_account", "wallet" (if provider supports crypto-to-crypto payouts)
	Status            string // e.g., "new", "verified", "verification_failed", "active", "errored"
	IsDefault         bool
	AccountHolderName string
	BankName          string // If type is bank_account
	Last4             string // e.g., last 4 of account number or wallet address
	RoutingNumber     string
	Country           string
	Currency          string // Intended payout currency for this account
	CreatedAt         int64
	UpdatedAt         int64
	Metadata          map[string]string
}

// WebhookEvent represents a normalized webhook event from any payment provider.
type WebhookEvent struct {
	ProviderEventID string      // The event ID from the provider (e.g., Stripe evt_xxx, Bridge event_id_yyy)
	Provider        string      // e.g., "stripe", "bridge_xyz"
	EventType       string      // e.g., "invoice.paid" (Stripe), "payout.succeeded" (Bridge)
	ReceivedAt      int64       // Unix timestamp when our system received/processed it
	Data            interface{} // The mapped canonical ps struct (e.g., ps.Invoice, ps.Transaction)
	RawData         []byte      // Raw event data for auditing/debugging
	SignatureValid  bool        // Whether the webhook signature was successfully verified
}

// ListParams provides common parameters for listing resources.
type ListParams struct {
	Limit         int
	StartingAfter string // Cursor for pagination (usually an ExternalID)
	EndingBefore  string // Cursor for pagination
	CreatedAfter  int64  // Unix timestamp
	CreatedBefore int64  // Unix timestamp
	IDs           []string
	Filters       map[string]interface{}
}

// PaymentSyncService defines the interface for interacting with various subscription management platforms.
// Implementations of this interface will handle the specifics of communicating with each platform (e.g., Stripe, Chargebee).
// All methods should aim for idempotency where the external platform supports it or by internal tracking.
type PaymentSyncService interface {
	// GetServiceName returns a unique identifier for the service implementation (e.g., "stripe", "chargebee").
	GetServiceName() string

	// Configure initializes the service with necessary credentials and settings.
	// config is a map containing platform-specific keys like "api_key", "webhook_secret", "base_url".
	Configure(ctx context.Context, config map[string]string) error

	// CheckConnection verifies connectivity to the external platform and authenticates if necessary.
	CheckConnection(ctx context.Context) error

	// --- Customer Management ---
	// CreateCustomer creates a customer on the external platform.
	// Returns the external ID of the newly created customer.
	CreateCustomer(ctx context.Context, customerData Customer) (externalID string, err error)
	// GetCustomer retrieves a customer by their external ID.
	GetCustomer(ctx context.Context, externalID string) (Customer, error)
	// UpdateCustomer updates an existing customer.
	UpdateCustomer(ctx context.Context, externalID string, customerData Customer) (Customer, error)
	// DeleteCustomer deletes a customer. Behavior might vary (soft vs. hard delete).
	DeleteCustomer(ctx context.Context, externalID string) error
	// ListCustomers retrieves a list of customers.
	ListCustomers(ctx context.Context, params ListParams) ([]Customer, string, error) // Returns customers, next page cursor, error

	// --- Product & Price Management ---
	CreateProduct(ctx context.Context, productData Product) (externalID string, err error)
	GetProduct(ctx context.Context, externalID string) (Product, error)
	UpdateProduct(ctx context.Context, externalID string, productData Product) (Product, error)
	DeleteProduct(ctx context.Context, externalID string) error
	ListProducts(ctx context.Context, params ListParams) ([]Product, string, error)

	CreatePrice(ctx context.Context, priceData Price) (externalID string, err error)
	GetPrice(ctx context.Context, externalID string) (Price, error)
	UpdatePrice(ctx context.Context, externalID string, priceData Price) (Price, error)
	// ListPrices can be filtered by productExternalID.
	ListPrices(ctx context.Context, productExternalID string, params ListParams) ([]Price, string, error)

	// --- Subscription Management ---
	CreateSubscription(ctx context.Context, subscriptionData Subscription) (Subscription, error)
	GetSubscription(ctx context.Context, externalID string) (Subscription, error)
	UpdateSubscription(ctx context.Context, externalID string, itemsToUpdate []SubscriptionItem, prorationBehavior string, metadata map[string]string, otherUpdateFields map[string]interface{}) (Subscription, error)
	CancelSubscription(ctx context.Context, externalID string, cancelAtPeriodEnd bool, invoiceNow bool) (Subscription, error)
	ListSubscriptions(ctx context.Context, params ListParams) ([]Subscription, string, error)
	ReactivateSubscription(ctx context.Context, externalID string) (Subscription, error)

	// --- Invoice Management ---
	GetInvoice(ctx context.Context, externalID string) (Invoice, error)
	ListInvoices(ctx context.Context, params ListParams) ([]Invoice, string, error)
	CreateInvoice(ctx context.Context, customerExternalID string, subscriptionExternalID string, items []InvoiceLineItem, autoAdvance bool) (Invoice, error)
	PayInvoice(ctx context.Context, externalInvoiceID string, paymentMethodExternalID string) (Invoice, error)
	VoidInvoice(ctx context.Context, externalInvoiceID string) (Invoice, error)
	FinalizeInvoice(ctx context.Context, externalInvoiceID string) (Invoice, error)
	SendInvoice(ctx context.Context, externalInvoiceID string) (Invoice, error) // Sends the invoice to the customer

	// --- Transaction/Payment Management ---
	GetTransaction(ctx context.Context, externalID string) (Transaction, error)
	ListTransactions(ctx context.Context, params ListParams) ([]Transaction, string, error)
	// CreatePaymentIntent creates an intent to collect payment.
	CreatePaymentIntent(ctx context.Context, amount int64, currency string, customerExternalID string, paymentMethodExternalID string, confirm bool, offSession bool, metadata map[string]string) (Transaction, error)
	// CapturePaymentIntent captures a previously created PaymentIntent.
	CapturePaymentIntent(ctx context.Context, paymentIntentExternalID string, amountToCapture int64) (Transaction, error)
	// CreateRefund refunds a charge or payment.
	CreateRefund(ctx context.Context, chargeExternalID string, amount int64, reason string, metadata map[string]string) (Transaction, error)

	// --- External Account / Payment Method Management ---
	CreateExternalAccount(ctx context.Context, customerExternalID string, accountData ExternalAccount, setAsDefault bool) (ExternalAccount, error)
	GetExternalAccount(ctx context.Context, customerExternalID string, externalAccountID string) (ExternalAccount, error)
	UpdateExternalAccount(ctx context.Context, customerExternalID string, externalAccountID string, accountData ExternalAccount) (ExternalAccount, error)
	DeleteExternalAccount(ctx context.Context, customerExternalID string, externalAccountID string) error
	ListExternalAccounts(ctx context.Context, customerExternalID string, params ListParams) ([]ExternalAccount, string, error)
	SetDefaultExternalAccount(ctx context.Context, customerExternalID string, externalAccountID string) error

	// --- Webhook Handling ---
	// HandleWebhook parses and validates an incoming webhook request from the external platform.
	// It should verify signatures if the platform supports it (e.g., Stripe's webhook signatures).
	// Returns a structured WebhookEvent or an error if validation fails or parsing is not possible.
	// The signatureHeader is the raw value from the HTTP header (e.g., "Stripe-Signature").
	HandleWebhook(ctx context.Context, requestBody []byte, signatureHeader string) (WebhookEvent, error)

	// --- Initial Sync Management ---
	// StartInitialSync initiates a complete initial data synchronization from the payment provider.
	// This method creates a sync session and processes all specified entity types.
	// Returns the created sync session immediately, while the sync process runs asynchronously.
	StartInitialSync(ctx context.Context, workspaceID string, config InitialSyncConfig) (SyncSession, error)
}

// InitialSyncConfig holds configuration for initial sync operations
type InitialSyncConfig struct {
	BatchSize     int      `json:"batch_size"`
	EntityTypes   []string `json:"entity_types"`
	FullSync      bool     `json:"full_sync"`
	StartingAfter string   `json:"starting_after,omitempty"`
	EndingBefore  string   `json:"ending_before,omitempty"`
	MaxRetries    int      `json:"max_retries"`
	RetryDelay    int      `json:"retry_delay_seconds"`
}

// SyncSession represents a sync session for tracking sync progress
type SyncSession struct {
	ID           string                 `json:"id"`
	WorkspaceID  string                 `json:"workspace_id"`
	Provider     string                 `json:"provider"`
	SessionType  string                 `json:"session_type"`
	Status       string                 `json:"status"`
	EntityTypes  []string               `json:"entity_types"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Progress     map[string]interface{} `json:"progress,omitempty"`
	ErrorSummary map[string]interface{} `json:"error_summary,omitempty"`
	StartedAt    *int64                 `json:"started_at,omitempty"`
	CompletedAt  *int64                 `json:"completed_at,omitempty"`
	CreatedAt    int64                  `json:"created_at"`
	UpdatedAt    int64                  `json:"updated_at"`
}
