package mocks

import (
	"context"
	"testing"

	"github.com/cyphera/cyphera-api/libs/go/client/circle"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestMockCircleClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := NewMockCircleClientInterface(ctrl)

	// Mock user creation with actual Circle response structure
	expectedUserResponse := &circle.UserResponse{}

	mockClient.EXPECT().
		CreateUserWithPinAuth(gomock.Any(), "external-user-123").
		Return(expectedUserResponse, nil).
		Times(1)

	// Test the mock
	ctx := context.Background()
	userResponse, err := mockClient.CreateUserWithPinAuth(ctx, "external-user-123")
	
	assert.NoError(t, err)
	assert.NotNil(t, userResponse)
}

func TestMockCircleClientWithHelper(t *testing.T) {
	mockClient := NewMockCircleClientForTest(t)

	// Mock wallet creation with simplified request/response
	createWalletsRequest := circle.CreateWalletsRequest{
		AccountType:    "SCA",
		Blockchains:    []string{"ETH", "MATIC"},
		IdempotencyKey: "wallet-create-123",
	}

	expectedWalletResponse := &circle.CreateWalletsResponse{}

	mockClient.EXPECT().
		CreateWallets(gomock.Any(), createWalletsRequest, "user-token-123").
		Return(expectedWalletResponse, nil).
		Times(1)

	// Test wallet balance query with actual Circle param structure
	balanceParams := &circle.GetWalletBalanceParams{}
	expectedBalance := &circle.WalletBalanceResponse{}

	mockClient.EXPECT().
		GetWalletBalance(gomock.Any(), "wallet-123", "user-token-123", balanceParams).
		Return(expectedBalance, nil).
		Times(1)

	// Execute the mocked methods
	ctx := context.Background()
	
	walletResponse, err := mockClient.CreateWallets(ctx, createWalletsRequest, "user-token-123")
	assert.NoError(t, err)
	assert.NotNil(t, walletResponse)

	balanceResponse, err := mockClient.GetWalletBalance(ctx, "wallet-123", "user-token-123", balanceParams)
	assert.NoError(t, err)
	assert.NotNil(t, balanceResponse)
}