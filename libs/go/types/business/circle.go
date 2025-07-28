package business

import "github.com/cyphera/cyphera-api/libs/go/db"

// CircleWalletData represents Circle-specific wallet data
type CircleWalletData struct {
	CircleWalletID string
	CircleUserID   string
	ChainID        int32
	State          string
}

// WalletWithCircleData represents a wallet with optional Circle data
type WalletWithCircleData struct {
	Wallet     db.Wallet
	CircleData *CircleWalletData
}
