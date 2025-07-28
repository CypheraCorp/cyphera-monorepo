package mocks

import (
	"testing"
	
	"go.uber.org/mock/gomock"
)

// NewMockPaymentSyncServiceForTest creates a new mock PaymentSyncService for testing
func NewMockPaymentSyncServiceForTest(t *testing.T) *MockPaymentSyncService {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	return NewMockPaymentSyncService(ctrl)
}

// NewMockMetricsCollectorForTest creates a new mock MetricsCollector for testing  
func NewMockMetricsCollectorForTest(t *testing.T) *MockMetricsCollector {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	return NewMockMetricsCollector(ctrl)
}

// NewMockCircleClientForTest creates a new mock CircleClientInterface for testing
func NewMockCircleClientForTest(t *testing.T) *MockCircleClientInterface {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	return NewMockCircleClientInterface(ctrl)
}
