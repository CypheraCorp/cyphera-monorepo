package interfaces

import (
	"context"

	"github.com/google/uuid"
)

// DelegationClient handles delegation operations with the delegation server
type DelegationClient interface {
	ProcessPayment(ctx context.Context, params ProcessPaymentParams) (*ProcessPaymentResponse, error)
	CreateDelegation(ctx context.Context, params CreateDelegationParams) (*CreateDelegationResponse, error)
	RevokeDelegation(ctx context.Context, delegationID string) error
	GetDelegationStatus(ctx context.Context, delegationID string) (*DelegationStatus, error)
}

// ProcessPaymentParams contains parameters for processing a payment
type ProcessPaymentParams struct {
	DelegationID     string
	RecipientAddress string
	Amount           string
	TokenAddress     string
	NetworkID        uuid.UUID
}

// ProcessPaymentResponse contains the response from processing a payment
type ProcessPaymentResponse struct {
	TransactionHash string
	Status          string
	GasUsed         string
	BlockNumber     uint64
}

// CreateDelegationParams contains parameters for creating a delegation
type CreateDelegationParams struct {
	DelegatorAddress string
	NetworkID        uuid.UUID
	TokenAddress     string
	Amount           string
}

// CreateDelegationResponse contains the response from creating a delegation
type CreateDelegationResponse struct {
	DelegationID string
	Signature    string
	ExpiresAt    int64
}

// DelegationStatus represents the status of a delegation
type DelegationStatus struct {
	DelegationID    string
	Status          string
	RemainingAmount string
	UsedAmount      string
}

// CircleClient handles Circle API operations
type CircleClient interface {
	CreateWallet(ctx context.Context, params CreateWalletParams) (*CircleWallet, error)
	GetWallet(ctx context.Context, walletID string) (*CircleWallet, error)
	CreateUser(ctx context.Context, email string) (*CircleUser, error)
	GetUser(ctx context.Context, userID string) (*CircleUser, error)
	InitiateTransaction(ctx context.Context, params TransactionParams) (*CircleTransaction, error)
	GetTransaction(ctx context.Context, transactionID string) (*CircleTransaction, error)
}

// CreateWalletParams contains parameters for creating a Circle wallet
type CreateWalletParams struct {
	UserID      string
	NetworkType string
}

// CircleWallet represents a Circle wallet
type CircleWallet struct {
	ID          string
	Address     string
	NetworkType string
	Status      string
}

// CircleUser represents a Circle user
type CircleUser struct {
	ID    string
	Email string
	Token string
}

// TransactionParams contains parameters for a Circle transaction
type TransactionParams struct {
	FromWallet string
	ToAddress  string
	Amount     string
	TokenID    string
}

// CircleTransaction represents a Circle transaction
type CircleTransaction struct {
	ID              string
	TransactionHash string
	Status          string
	Amount          string
}

// StripeClient handles Stripe operations
type StripeClient interface {
	CreateCustomer(ctx context.Context, email string) (*StripeCustomer, error)
	CreateSubscription(ctx context.Context, customerID, priceID string) (*StripeSubscription, error)
	CancelSubscription(ctx context.Context, subscriptionID string) error
	CreatePaymentIntent(ctx context.Context, amount int64, currency string) (*StripePaymentIntent, error)
}

// StripeCustomer represents a Stripe customer
type StripeCustomer struct {
	ID    string
	Email string
}

// StripeSubscription represents a Stripe subscription
type StripeSubscription struct {
	ID         string
	CustomerID string
	Status     string
}

// StripePaymentIntent represents a Stripe payment intent
type StripePaymentIntent struct {
	ID     string
	Amount int64
	Status string
}

// ResendClient handles email operations via Resend
type ResendClient interface {
	SendEmail(ctx context.Context, params EmailParams) error
	SendBatch(ctx context.Context, emails []EmailParams) error
}

// EmailParams contains parameters for sending an email
type EmailParams struct {
	From    string
	To      []string
	Subject string
	HTML    string
	Text    string
	Tags    map[string]string
}

// Web3AuthClient handles Web3Auth operations
type Web3AuthClient interface {
	VerifyToken(ctx context.Context, token string) (*Web3AuthUser, error)
	GetUserInfo(ctx context.Context, userID string) (*Web3AuthUser, error)
}

// Web3AuthUser represents a Web3Auth user
type Web3AuthUser struct {
	ID            string
	Email         string
	WalletAddress string
	Provider      string
}
