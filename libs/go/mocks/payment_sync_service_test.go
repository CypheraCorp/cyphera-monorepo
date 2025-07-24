package mocks

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestMockPaymentSyncService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := NewMockPaymentSyncService(ctrl)

	// Test basic mock functionality
	mockService.EXPECT().
		GetServiceName().
		Return("test-service").
		Times(1)

	// Test the mock
	serviceName := mockService.GetServiceName()
	assert.Equal(t, "test-service", serviceName)
}

func TestMockPaymentSyncServiceWithHelper(t *testing.T) {
	mockService := NewMockPaymentSyncServiceForTest(t)

	// Test Configure method
	config := map[string]string{
		"api_key": "test-key",
	}
	
	mockService.EXPECT().
		Configure(gomock.Any(), config).
		Return(nil).
		Times(1)

	// Test CheckConnection method
	mockService.EXPECT().
		CheckConnection(gomock.Any()).
		Return(nil).
		Times(1)

	// Execute the mocked methods
	err := mockService.Configure(context.Background(), config)
	assert.NoError(t, err)

	err = mockService.CheckConnection(context.Background())
	assert.NoError(t, err)
}