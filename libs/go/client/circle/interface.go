package circle

import "context"

// CircleClientInterface defines the interface for Circle API operations
type CircleClientInterface interface {
	// User management
	GetUserByToken(ctx context.Context, userToken string) (*UserResponse, error)
	GetUserByID(ctx context.Context, userID string) (*UserByIDResponse, error)
	InitializeUser(ctx context.Context, request InitializeUserRequest, userToken string) (*InitializeUserResponse, error)
	CreateUserToken(ctx context.Context, userID string) (*UserTokenResponse, error)
	CreateUserWithPinAuth(ctx context.Context, externalUserID string) (*UserResponse, error)
	GetChallenge(ctx context.Context, challengeID string, userToken string) (*ChallengeResponse, error)

	// PIN management
	CreatePinChallenge(ctx context.Context, idempotencyKey string, userToken string) (*PinChallengeResponse, error)
	UpdatePinChallenge(ctx context.Context, idempotencyKey string, userToken string) (*PinChallengeResponse, error)
	CreatePinRestoreChallenge(ctx context.Context, idempotencyKey string, userToken string) (*PinChallengeResponse, error)

	// Wallet management
	CreateWallets(ctx context.Context, request CreateWalletsRequest, userToken string) (*CreateWalletsResponse, error)
	GetWalletBalance(ctx context.Context, walletID string, userToken string, params *GetWalletBalanceParams) (*WalletBalanceResponse, error)
	GetWallet(ctx context.Context, walletID string, userToken string) (*WalletResponse, error)
	ListWallets(ctx context.Context, userToken string, params *ListWalletsParams) (*ListWalletsResponse, error)

	// Transaction management
	CreateTransferChallenge(ctx context.Context, request TransferChallengeRequest, userToken string) (*TransferChallengeResponse, error)
	ListTransactions(ctx context.Context, userToken string, params *ListTransactionsParams) (*TransactionListResponse, error)
	GetTransaction(ctx context.Context, transactionID string, userToken string) (*TransactionResponse, error)
	EstimateTransferFee(ctx context.Context, request EstimateTransferFeeRequest, userToken string) (*EstimateTransferFeeResponse, error)
	ValidateAddress(ctx context.Context, request ValidateAddressRequest) (*ValidateAddressResponse, error)
}

// Ensure CircleClient implements the interface
var _ CircleClientInterface = (*CircleClient)(nil)