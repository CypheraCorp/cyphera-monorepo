package params

import "github.com/google/uuid"

// CreateUserParams contains parameters for creating a user
type CreateUserParams struct {
	Web3AuthID     string
	Verifier       string
	VerifierID     string
	Email          string
	AccountID      uuid.UUID
	Role           string
	IsAccountOwner bool
	FirstName      string
	LastName       string
	AddressLine1   string
	AddressLine2   string
	City           string
	StateRegion    string
	PostalCode     string
	Country        string
	DisplayName    string
	PictureURL     string
	Phone          string
	Timezone       string
	Locale         string
	EmailVerified  bool
	Metadata       map[string]interface{}
}

// UpdateUserParams contains parameters for updating a user
type UpdateUserParams struct {
	ID               uuid.UUID
	Email            string
	FirstName        string
	LastName         string
	AddressLine1     string
	AddressLine2     string
	City             string
	StateRegion      string
	PostalCode       string
	Country          string
	DisplayName      string
	PictureURL       string
	Phone            string
	Timezone         string
	Locale           string
	EmailVerified    *bool
	TwoFactorEnabled *bool
	Status           string
	Metadata         map[string]interface{}
}
