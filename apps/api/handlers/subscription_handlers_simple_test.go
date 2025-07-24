package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/cyphera/cyphera-api/libs/go/client/coinmarketcap"
	dsClient "github.com/cyphera/cyphera-api/libs/go/client/delegation_server"
	"github.com/cyphera/cyphera-api/libs/go/db"
)

func TestNewSubscriptionHandler_Simple(t *testing.T) {
	// Create basic dependencies
	common := &CommonServices{
		db:                        &db.Queries{},
		cypheraSmartWalletAddress: "0xtest",
		CMCClient:                 &coinmarketcap.Client{},
	}
	
	delegationClient := &dsClient.DelegationClient{}
	
	// Create handler
	handler := NewSubscriptionHandler(common, delegationClient)
	
	// Verify initialization
	assert.NotNil(t, handler)
	assert.Equal(t, common, handler.common)
	assert.Equal(t, delegationClient, handler.delegationClient)
	assert.Equal(t, "", handler.lastRedemptionTxHash)
}

func TestSubscriptionExistsError_Simple(t *testing.T) {
	subscription := &db.Subscription{
		ID: testUUID,
	}
	
	err := &SubscriptionExistsError{Subscription: subscription}
	
	expectedMessage := "subscription already exists with ID: " + testUUID.String()
	assert.Equal(t, expectedMessage, err.Error())
}

// Test constants and helpers
var testUUID = uuid.MustParse("01234567-89ab-cdef-0123-456789abcdef")