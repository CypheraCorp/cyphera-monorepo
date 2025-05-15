// Package offramp defines the interfaces and data structures for interacting with various off-ramp providers.
package offramp

import "time"

// OffRampProvider defines the common interface for all off-ramp providers.
// Each method is designed to abstract provider-specific implementations,
// allowing the core application to interact with them uniformly.
type OffRampProvider interface {
	// GetName returns the unique, human-readable name of the provider.
	// This can be used for logging, UI display, or internal identification.
	GetName() string

	// GetCapabilities describes what networks, currencies, and jurisdictions the provider supports,
	// as well as other relevant operational details like fee structures.
	GetCapabilities() (ProviderCapabilities, error)

	// InitiateKYC starts the Know Your Customer (KYC) or Know Your Business (KYB) process
	// for an end-user (individual or business entity) with the provider.
	// It should return information necessary for the user to proceed, such as a hosted KYC/KYB URL
	// or details about the next steps if the process is API-driven.
	// Note: Providers perform their own KYB/KYC. This method initiates that process,
	// which might involve redirecting to a provider's hosted form or submitting
	// collected data via API for provider verification.
	InitiateKYC(request KYCRequest) (KYCResponse, error)

	// GetKYCStatus retrieves the current KYC/KYB status of an end-user (individual or business entity) from the provider.
	// This is used to monitor the progress and outcome of the verification process.
	GetKYCStatus(request KYCStatusRequest) (KYCStatusResponse, error)

	// GetDepositAddress provides a crypto deposit address (custodial wallet, where applicable)
	// for the end-user to send funds to, for a specific cryptocurrency and network.
	// This address is managed by the off-ramp provider.
	GetDepositAddress(request DepositAddressRequest) (DepositAddressResponse, error)

	// InitiateOffRamp starts the crypto-to-fiat conversion and payout process.
	// This typically follows a successful KYC and the user having funds at a deposit address
	// or being ready to send funds based on provider instructions.
	InitiateOffRamp(request OffRampTransactionRequest) (OffRampTransactionResponse, error)

	// GetOffRampStatus retrieves the current status of an off-ramp transaction.
	// This allows tracking the progress from initiation through to completion or failure.
	GetOffRampStatus(request OffRampStatusRequest) (OffRampStatusResponse, error)

	// GetAllKYCLinks retrieves a list of KYC links, optionally filtered by criteria in the request.
	GetAllKYCLinks(request GetAllKYCLinksRequest) (GetAllKYCLinksResponse, error)

	// CreateWallet requests the provider to create a new wallet for a given customer on a specific network.
	CreateWallet(request CreateWalletRequest) (CreateWalletResponse, error)

	// GetWallet retrieves details for a specific wallet, including its balances.
	GetWallet(request GetWalletRequest) (GetWalletResponse, error)

	// GetAllWallets retrieves a list of wallets for a customer, with optional pagination.
	GetAllWallets(request GetAllWalletsRequest) (GetAllWalletsResponse, error)
}

// ProviderCapabilities describes the operational capabilities of an off-ramp provider.
// This information is crucial for dynamic routing and informing users/merchants.

type NetworkData struct {
	ChainID string `json:"chainId"` // Primary Key
	Network string `json:"network"`
}

type JurisdictionData struct {
	CountryCode     string `json:"countryCode"` // Primary Key
	CountryName     string `json:"countryName"`
	SubdivisionCode string `json:"subdivisionCode"`
	SubdivisionName string `json:"subdivisionName"`
}

type ProviderCapabilities struct {
	// SupportedNetworks is a list of blockchain networks supported. chainID and networkName
	SupportedChainIDs []NetworkData
	// SupportedCryptoCurrencies is a list of crypto ticker symbols supported (e.g., "USDC", "ETH", "BTC").
	SupportedCryptoCurrencies []string
	// SupportedFiatCurrencies is a list of fiat currency codes supported (e.g., "USD", "EUR", "GBP").
	SupportedFiatCurrencies []string
	// SupportedJurisdictions is a list of country codes (ISO 3166-1 alpha-2) where the provider can operate
	// or serve customers.
	SupportedJurisdictions []string
	// FeeStructureDetails provides a summary or link to documentation about the provider's fee structure.
	// This might be a textual description or a more structured representation in future iterations.
	FeeStructureDetails string
}

// Address represents a postal address, designed to be internationally compatible.
// All fields are optional to accommodate various address formats and data availability.
type Address struct {
	StreetLine1   string `json:"streetLine1,omitempty"`   // Street address, line 1
	StreetLine2   string `json:"streetLine2,omitempty"`   // Street address, line 2 (e.g., apartment, suite, unit, building)
	City          string `json:"city,omitempty"`          // City, district, suburb, town, or village
	StateProvince string `json:"stateProvince,omitempty"` // State, province, prefecture, or region
	PostalCode    string `json:"postalCode,omitempty"`    // ZIP or postal code
	Country       string `json:"country,omitempty"`       // ISO 3166-1 alpha-2 country code (e.g., "US", "CA", "GB")
}

// UserInfo contains basic information about an end-user, whether an individual or a business.
// This struct should be expanded based on the common minimum requirements for KYC/KYB
// across the integrated providers.
type UserInfo struct {
	// EntityType indicates if this UserInfo pertains to an "INDIVIDUAL" or "BUSINESS".
	// This helps in processing and validation.
	EntityType string `json:"entityType"` // e.g., "INDIVIDUAL", "BUSINESS"

	// For Individuals
	FirstName   string  `json:"firstName,omitempty"`
	LastName    string  `json:"lastName,omitempty"`
	Email       string  `json:"email,omitempty"`       // Can be for individual or business contact
	DateOfBirth string  `json:"dateOfBirth,omitempty"` // Expected format: "YYYY-MM-DD"
	Address     Address `json:"address,omitempty"`     // Physical address of the individual or registered address for sole prop.

	// For Businesses
	LegalBusinessName          string  `json:"legalBusinessName,omitempty"`
	BusinessType               string  `json:"businessType,omitempty"`               // e.g., "LLC", "CORPORATION", "SOLE_PROPRIETORSHIP", "PARTNERSHIP"
	BusinessRegistrationNumber string  `json:"businessRegistrationNumber,omitempty"` // e.g., EIN, Company Number
	RegistrationCountry        string  `json:"registrationCountry,omitempty"`        // ISO 3166-1 alpha-2 country code where business is registered
	DateOfIncorporation        string  `json:"dateOfIncorporation,omitempty"`        // Expected format: "YYYY-MM-DD"
	BusinessWebsite            string  `json:"businessWebsite,omitempty"`
	Industry                   string  `json:"industry,omitempty"`         // Nature of business or industry code
	PrincipalAddress           Address `json:"principalAddress,omitempty"` // Principal place of business address, if different from registered
	// TODO: Consider if further details like operating address or contact person for business are needed at this generic level.
}

// KYCRequest is used to initiate the KYC/KYB process for an end-user (individual or business).
type KYCRequest struct {
	// UserID is your internal unique identifier for the end-user.
	UserID string `json:"userId"`
	// UserDetails contains the collected information about the user (individual or business).
	UserDetails UserInfo `json:"userDetails"`
	// MerchantID is your internal unique identifier for the merchant initiating this request on behalf of the user.
	MerchantID string `json:"merchantId"`
	// IsSepa indicates if SEPA endorsement should be requested. Bridge only parameter.
	IsSepa *bool `json:"isSepa,omitempty"` // Optional: Bridge only parameter
}

// KYCResponse contains the result of initiating a KYC/KYB process (for an individual or business).
type KYCResponse struct {
	// KYCStatus indicates the immediate status after initiation (e.g., "PENDING_SUBMISSION", "INITIATED", "FAILED_VALIDATION").
	// This status pertains to the KYC/KYB attempt.
	KYCStatus string `json:"kycStatus"`
	// HostedKYCURL is the URL for the user to complete the KYC/KYB process in a hosted flow, if applicable.
	HostedKYCURL string `json:"hostedKycUrl,omitempty"`
	// ProviderReferenceID is an identifier from the provider for this specific KYC/KYB session or attempt.
	// This ID should be stored and used for subsequent status checks.
	ProviderReferenceID string `json:"providerReferenceId,omitempty"`
	// NextSteps provides a human-readable description of what the user or system should do next, if any.
	NextSteps string `json:"nextSteps,omitempty"`
}

// KYCStatusRequest is used to request the current status of a previously initiated KYC/KYB process.
type KYCStatusRequest struct {
	// UserID is your internal unique identifier for the end-user (individual or business).
	UserID string `json:"userId"`
	// ProviderReferenceID is the identifier returned by the provider when KYC was initiated.
	ProviderReferenceID string `json:"providerReferenceId"`
}

// KYCStatusResponse contains the current status of an end-user's KYC/KYB verification.
type KYCStatusResponse struct {
	// KYCStatus represents the provider's assessment of the user's verification status
	// (e.g., "PENDING_REVIEW", "ACTIVE", "REJECTED", "ACTION_REQUIRED", "EXPIRED").
	// This status reflects the outcome for either an individual (KYC) or a business (KYB).
	KYCStatus string `json:"kycStatus"`
	// Reason provides additional details, especially if the status is "REJECTED" or "ACTION_REQUIRED".
	Reason string `json:"reason,omitempty"`
	// RequiredActions lists specific actions the user needs to take, if any.
	RequiredActions []string `json:"requiredActions,omitempty"`
}

// DepositAddressRequest is used to request a cryptocurrency deposit address for an end-user.
type DepositAddressRequest struct {
	// UserID is your internal unique identifier for the end-user for whom the address is being requested.
	UserID string `json:"userId"`
	// Network specifies the blockchain network (e.g., "Ethereum", "Polygon", "Solana").
	Network string `json:"network"`
	// CryptoCurrency specifies the ticker symbol of the cryptocurrency (e.g., "USDC", "ETH").
	CryptoCurrency string `json:"cryptoCurrency"`
}

// DepositAddressResponse contains the details of a cryptocurrency deposit address.
type DepositAddressResponse struct {
	// Address is the actual crypto deposit address.
	CryptoWalletAddress string `json:"cryptoWalletAddress"`
	// Network is the blockchain network to which this address belongs.
	Network string `json:"network"`
	// Memo is an optional memo or tag required for deposits on certain networks/exchanges (e.g., XRP, XLM, EOS).
	Memo string `json:"memo,omitempty"`
	// QRCodeURI is an optional URI that can be used to generate a QR code for the deposit address (and memo, if applicable).
	QRCodeURI string `json:"qrCodeUri,omitempty"`
	// ExpiresAt indicates if and when this deposit address might expire. Zero value means no expiration.
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
}

// OffRampTransactionRequest is used to initiate an off-ramp transaction (crypto to fiat).
type OffRampTransactionRequest struct {
	// UserID is your internal unique identifier for the end-user.
	UserID string `json:"userId"`
	// SourceCryptoAmount is the amount of cryptocurrency to be off-ramped.
	// Represented as a string to maintain decimal precision (e.g., "100.50").
	SourceCryptoAmount string `json:"sourceCryptoAmount"`
	// SourceCryptoCurrency is the ticker symbol of the crypto being off-ramped (e.g., "USDC").
	SourceCryptoCurrency string `json:"sourceCryptoCurrency"`
	// SourceNetwork is the blockchain network from which the crypto will be sent (e.g., "Ethereum").
	SourceNetwork string `json:"sourceNetwork"`
	// DestinationFiatCurrency is the desired fiat currency for payout (e.g., "USD", "EUR").
	DestinationFiatCurrency string `json:"destinationFiatCurrency"`
	// DestinationBankAccountID is your internal identifier for the pre-registered and verified bank account
	// where the fiat funds should be sent. This ID maps to an account approved by the provider.
	DestinationBankAccountID string `json:"destinationBankAccountId"`
	// IdempotencyKey is a client-generated key to ensure the request is processed only once,
	// preventing duplicate transactions in case of retries.
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
	// QuoteID is an optional identifier from a preceding quote step, if the provider supports it.
	// This can lock in an exchange rate or fee structure.
	QuoteID string `json:"quoteId,omitempty"`
}

// OffRampTransactionResponse contains the initial result of initiating an off-ramp transaction.
type OffRampTransactionResponse struct {
	// TransactionID is the unique identifier assigned by the provider to this off-ramp transaction.
	TransactionID string `json:"transactionId"`
	// Status indicates the initial status of the transaction (e.g., "PENDING_FUNDS", "PROCESSING", "AWAITING_CONFIRMATION").
	Status string `json:"status"`
	// EstimatedFiatAmount is the estimated amount of fiat currency the user will receive.
	// Represented as a string for decimal precision. This may change based on final FX rates and fees.
	EstimatedFiatAmount string `json:"estimatedFiatAmount,omitempty"`
	// DepositInstructions provides any specific instructions for the user to send their cryptocurrency,
	// if not already at a provider-managed deposit address. This could include a one-time address or memo.
	DepositInstructions string `json:"depositInstructions,omitempty"`
	// RequiredConfirmations is the number of blockchain confirmations needed if user is sending funds.
	RequiredConfirmations int `json:"requiredConfirmations,omitempty"`
}

// OffRampStatusRequest is used to request the current status of an ongoing off-ramp transaction.
type OffRampStatusRequest struct {
	// TransactionID is the provider's unique identifier for the transaction.
	TransactionID string `json:"transactionId"`
}

// OffRampStatusResponse contains the detailed current status of an off-ramp transaction.
type OffRampStatusResponse struct {
	// Status indicates the current state of the transaction (e.g., "PROCESSING", "COMPLETED", "FAILED", "RETURNED", "PENDING_SETTLEMENT").
	Status string `json:"status"`
	// ActualFiatAmount is the final amount of fiat currency credited or to be credited.
	// Represented as a string for decimal precision.
	ActualFiatAmount string `json:"actualFiatAmount,omitempty"`
	// PayoutDate is the date when the fiat funds were or are expected to be paid out.
	// Could be just a date "YYYY-MM-DD" or a full timestamp.
	PayoutDate string `json:"payoutDate,omitempty"`
	// ProviderFeeAmount is the fee charged by the off-ramp provider for this transaction.
	// Represented as a string for decimal precision.
	ProviderFeeAmount string `json:"providerFeeAmount,omitempty"`
	// FiatTransactionID is the provider's reference for the fiat payout (e.g., bank transfer ID).
	FiatTransactionID string `json:"fiatTransactionId,omitempty"`
	// ReasonForFailure provides details if the transaction status is "FAILED" or "RETURNED".
	ReasonForFailure string `json:"reasonForFailure,omitempty"`
	// LastUpdated is the timestamp of when this status was last updated by the provider.
	LastUpdated time.Time `json:"lastUpdated,omitempty"`
}

// --- Structs and Interface Method for Get All KYC Links ---

// RejectionReasonDetail provides structured information about why a KYC/KYB attempt might have issues.
type RejectionReasonDetail struct {
	DeveloperReason string `json:"developerReason,omitempty"` // Provider-specific internal reason/code
	Reason          string `json:"reason"`                    // User-facing reason
	CreatedAt       string `json:"createdAt,omitempty"`       // Timestamp of when the reason was recorded, ISO 8601 string. Consider time.Time if direct parsing is needed.
}

// KYCLinkDetail contains all relevant details for a single KYC link instance.
type KYCLinkDetail struct {
	ID                 string                  `json:"id"` // Provider's unique ID for the KYC link itself
	FullName           string                  `json:"fullName,omitempty"`
	Email              string                  `json:"email,omitempty"`
	Type               string                  `json:"type,omitempty"` // e.g., "INDIVIDUAL", "BUSINESS"
	KYCLinkURL         string                  `json:"kycLinkUrl,omitempty"`
	TOSLinkURL         string                  `json:"tosLinkUrl,omitempty"`
	KYCStatus          string                  `json:"kycStatus,omitempty"`          // Status of the KYC process itself (e.g., "NOT_STARTED", "PENDING", "APPROVED")
	TOSStatus          string                  `json:"tosStatus,omitempty"`          // Status of Terms of Service acceptance (e.g., "PENDING", "ACCEPTED")
	ProviderCustomerID string                  `json:"providerCustomerId,omitempty"` // The customer ID at the provider, if linked/created
	RejectionReasons   []RejectionReasonDetail `json:"rejectionReasons,omitempty"`
	// Add other common fields that might be relevant from various providers
}

// GetAllKYCLinksRequest contains parameters for requesting a list of KYC links.
type GetAllKYCLinksRequest struct {
	CustomerID string `json:"customerId,omitempty"` // Filter by provider's customer ID
	Email      string `json:"email,omitempty"`      // Filter by user's email
	// TODO: Consider adding pagination parameters if common (e.g., Page, PageSize, Offset, Limit)
}

// GetAllKYCLinksResponse is the structure for the response containing a list of KYC links.
type GetAllKYCLinksResponse struct {
	Count int             `json:"count"` // Total number of KYC links matching the query
	Data  []KYCLinkDetail `json:"data"`  // The list of KYC link details
}

// --- Structs and Interface Method for Create Wallet ---

// CreateWalletRequest contains parameters for requesting the creation of a new wallet by the provider.
type CreateWalletRequest struct {
	ProviderCustomerID string `json:"providerCustomerId"` // The customer ID at the provider for whom to create the wallet.
	Network            string `json:"network"`            // The blockchain network/chain for the wallet (e.g., "Solana", "Ethereum", "Base").
	// Potentially add other parameters like wallet type if providers support variations.
}

// CreateWalletResponse contains details of the newly created wallet.
type CreateWalletResponse struct {
	ProviderWalletID string `json:"providerWalletId"`    // The provider's unique ID for this wallet.
	Network          string `json:"network"`             // The blockchain network/chain of the wallet.
	WalletAddress    string `json:"walletAddress"`       // The actual public address of the wallet on the chain.
	CreatedAt        string `json:"createdAt,omitempty"` // ISO 8601 timestamp. Consider time.Time for direct parsing.
	UpdatedAt        string `json:"updatedAt,omitempty"` // ISO 8601 timestamp. Consider time.Time for direct parsing.
}

// --- Structs and Interface Method for Get Wallet ---

// WalletBalance represents the balance of a specific currency in a wallet.
type WalletBalance struct {
	Amount          string `json:"amount"`                    // The amount of the currency, e.g., "100.25"
	Currency        string `json:"currency"`                  // Ticker symbol of the currency, e.g., "USDB", "USDC"
	Network         string `json:"network"`                   // The network/chain this balance pertains to.
	ContractAddress string `json:"contractAddress,omitempty"` // Contract address of the token, if applicable.
}

// GetWalletRequest contains parameters for requesting details of a specific wallet.
type GetWalletRequest struct {
	ProviderCustomerID string `json:"providerCustomerId"` // The customer ID at the provider.
	ProviderWalletID   string `json:"providerWalletId"`   // The provider's unique ID for the wallet.
}

// GetWalletResponse contains detailed information about a specific wallet, including its balances.
type GetWalletResponse struct {
	ProviderWalletID string          `json:"providerWalletId"`
	Network          string          `json:"network"`
	WalletAddress    string          `json:"walletAddress"`
	CreatedAt        string          `json:"createdAt,omitempty"`
	UpdatedAt        string          `json:"updatedAt,omitempty"`
	Balances         []WalletBalance `json:"balances,omitempty"`
}

// --- Structs and Interface Method for Get All Wallets ---

// GetAllWalletsRequest contains parameters for requesting a list of wallets for a customer.
// Includes common pagination parameters.
type GetAllWalletsRequest struct {
	ProviderCustomerID string  `json:"providerCustomerId"`      // The customer ID at the provider.
	Limit              *int    `json:"limit,omitempty"`         // Optional: Number of items to return.
	StartingAfter      *string `json:"startingAfter,omitempty"` // Optional: Cursor for pagination (ID of item after which to start).
	EndingBefore       *string `json:"endingBefore,omitempty"`  // Optional: Cursor for pagination (ID of item before which to end).
}

// GetAllWalletsResponse is the structure for the response containing a list of wallets.
// It uses CreateWalletResponse for individual wallet items as they share common fields for a list view.
type GetAllWalletsResponse struct {
	Count int                    `json:"count"` // Total number of wallets matching the query (may differ from len(Data) due to pagination).
	Data  []CreateWalletResponse `json:"data"`  // The list of wallet details.
}
