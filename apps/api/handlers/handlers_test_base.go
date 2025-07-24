package handlers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/cyphera/cyphera-api/libs/go/client/coinmarketcap"
	dsClient "github.com/cyphera/cyphera-api/libs/go/client/delegation_server"
	"github.com/cyphera/cyphera-api/libs/go/db"
)

// Test helpers and fixtures

var (
	testWorkspaceID  = uuid.MustParse("01234567-89ab-cdef-0123-456789abcdef")
	testCustomerID   = uuid.MustParse("11234567-89ab-cdef-0123-456789abcdef")
	testProductID    = uuid.MustParse("21234567-89ab-cdef-0123-456789abcdef")
	testPriceID      = uuid.MustParse("31234567-89ab-cdef-0123-456789abcdef")
	testDelegationID = uuid.MustParse("41234567-89ab-cdef-0123-456789abcdef")
	testTokenID      = uuid.MustParse("51234567-89ab-cdef-0123-456789abcdef")
)

// createTestSubscription creates a test subscription with required fields
func createTestSubscription() db.Subscription {
	now := time.Now()
	return db.Subscription{
		ID:                 uuid.New(),
		CustomerID:         testCustomerID,
		ProductID:          testProductID,
		WorkspaceID:        testWorkspaceID,
		PriceID:            testPriceID,
		ProductTokenID:     testTokenID,
		TokenAmount:        1000,
		DelegationID:       testDelegationID,
		Status:             db.SubscriptionStatusActive,
		CurrentPeriodStart: pgtype.Timestamptz{Time: now, Valid: true},
		CurrentPeriodEnd:   pgtype.Timestamptz{Time: now.Add(30 * 24 * time.Hour), Valid: true},
		NextRedemptionDate: pgtype.Timestamptz{Time: now.Add(24 * time.Hour), Valid: true},
		TotalRedemptions:   0,
		TotalAmountInCents: 100000, // $1000.00
		CreatedAt:          pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:          pgtype.Timestamptz{Time: now, Valid: true},
	}
}

// createTestCommonServices creates a basic CommonServices for testing
func createTestCommonServices() *CommonServices {
	return &CommonServices{
		db:                        &db.Queries{}, // In real tests, this would be mocked
		cypheraSmartWalletAddress: "0xtest123",
		CMCClient:                 &coinmarketcap.Client{},
	}
}

// createTestDelegationClient creates a basic DelegationClient for testing
func createTestDelegationClient() *dsClient.DelegationClient {
	return &dsClient.DelegationClient{}
}

// createTestSubscriptionHandler creates a basic SubscriptionHandler for testing
func createTestSubscriptionHandler() *SubscriptionHandler {
	common := createTestCommonServices()
	delegationClient := createTestDelegationClient()
	return NewSubscriptionHandler(common, delegationClient)
}

// TestContext creates a test context with timeout
func createTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}