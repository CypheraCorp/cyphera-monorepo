package params

import "github.com/google/uuid"

// CreateWalletParams contains parameters for creating a wallet
type CreateWalletParams struct {
	WorkspaceID   uuid.UUID
	WalletType    string
	WalletAddress string
	NetworkType   string
	NetworkID     *uuid.UUID
	Nickname      string
	ENS           string
	IsPrimary     bool
	Verified      bool
	Metadata      map[string]interface{}
}

// UpdateWalletParams contains parameters for updating a wallet
type UpdateWalletParams struct {
	ID            uuid.UUID
	Nickname      *string
	ENS           *string
	IsPrimary     *bool
	Verified      *bool
	Metadata      map[string]interface{}
}
