package params

import "github.com/google/uuid"

// CreateCustomerParams contains parameters for creating a customer
type CreateCustomerParams struct {
	Email                   string
	Name                    *string
	Description             *string
	Phone                   *string
	CompanyName             *string
	Web3AuthID              *string
	Web3AuthEmail           *string
	FinishedOnboarding      bool
	CustomerTier            *string
	PreferredFiatCurrencies []string
	TaxExempt               bool
	VATNumber               *string
	Metadata                map[string]interface{}
}

// UpdateCustomerParams contains parameters for updating a customer
type UpdateCustomerParams struct {
	ID                      uuid.UUID
	Email                   *string
	Name                    *string
	Description             *string
	Phone                   *string
	CompanyName             *string
	Web3AuthID              *string
	Web3AuthEmail           *string
	FinishedOnboarding      *bool
	CustomerTier            *string
	PreferredFiatCurrencies []string
	TaxExempt               *bool
	VATNumber               *string
	Metadata                map[string]interface{}
}

// ListCustomersParams contains parameters for listing customers
type ListCustomersParams struct {
	Limit  int32
	Offset int32
	Search *string
}

// ListWorkspaceCustomersParams contains parameters for listing workspace customers
type ListWorkspaceCustomersParams struct {
	WorkspaceID uuid.UUID
	Limit       int32
	Offset      int32
	Search      *string
}

// CreateCustomerWithWeb3AuthParams contains parameters for creating a customer with Web3Auth
type CreateCustomerWithWeb3AuthParams struct {
	Web3AuthID    string
	Email         string
	Name          *string
	WalletAddress *string
	NetworkType   *string
	Metadata      map[string]interface{}
}

// CreateCustomerWalletParams contains parameters for creating a customer wallet
type CreateCustomerWalletParams struct {
	CustomerID    uuid.UUID
	WalletType    string
	WalletAddress string
	NetworkType   string
	NetworkID     *uuid.UUID
	Nickname      *string
	ENS           *string
	IsPrimary     bool
	Verified      bool
	Metadata      map[string]interface{}
}