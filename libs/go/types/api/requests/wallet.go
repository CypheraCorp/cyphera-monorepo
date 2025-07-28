package requests

// CreateWalletRequest represents the request body for creating a wallet
type CreateWalletRequest struct {
	WalletType    string                 `json:"wallet_type" binding:"required"` // 'wallet' or 'circle_wallet'
	WalletAddress string                 `json:"wallet_address" binding:"required"`
	NetworkType   string                 `json:"network_type" binding:"required"` // 'evm' or 'solana'
	Nickname      string                 `json:"nickname,omitempty"`
	ENS           string                 `json:"ens,omitempty"`
	IsPrimary     bool                   `json:"is_primary"`
	Verified      bool                   `json:"verified"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	// Circle wallet specific fields
	CircleUserID   string `json:"circle_user_id,omitempty"`   // Only for circle wallets
	CircleWalletID string `json:"circle_wallet_id,omitempty"` // Only for circle wallets
	ChainID        int32  `json:"chain_id,omitempty"`         // Only for circle wallets
	State          string `json:"state,omitempty"`            // Only for circle wallets
}

// UpdateWalletRequest represents the request body for updating a wallet
type UpdateWalletRequest struct {
	Nickname  string                 `json:"nickname,omitempty"`
	ENS       string                 `json:"ens,omitempty"`
	IsPrimary bool                   `json:"is_primary,omitempty"`
	Verified  bool                   `json:"verified,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	// Circle wallet specific fields
	State string `json:"state,omitempty"` // Only for circle wallets
}
