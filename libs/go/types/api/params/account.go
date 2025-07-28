package params

import "github.com/google/uuid"

// CreateAccountParams contains parameters for creating an account
type CreateAccountParams struct {
	Name               string
	AccountType        string
	Description        string
	BusinessName       string
	BusinessType       string
	WebsiteURL         string
	SupportEmail       string
	SupportPhone       string
	FinishedOnboarding bool
	Metadata           map[string]interface{}
}

// UpdateAccountParams contains parameters for updating an account
type UpdateAccountParams struct {
	ID                 uuid.UUID
	Name               *string
	Description        *string
	BusinessName       *string
	BusinessType       *string
	WebsiteURL         *string
	SupportEmail       *string
	SupportPhone       *string
	AccountType        *string
	FinishedOnboarding *bool
	Metadata           map[string]interface{}
}

// OnboardAccountParams contains parameters for onboarding an account
type OnboardAccountParams struct {
	AccountID     uuid.UUID
	UserID        uuid.UUID
	FirstName     string
	LastName      string
	AddressLine1  string
	AddressLine2  string
	City          string
	State         string
	PostalCode    string
	Country       string
	TaxID         string
	BusinessPhone string
	WalletAddress string
}
