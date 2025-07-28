#!/bin/bash

# Mock generation script for Cyphera API
# This script generates Go mocks using gomock for testing

set -e

echo "🔧 Generating Go mocks..."

# Root directory of the project
ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT_DIR"

# Create mocks directory if it doesn't exist
MOCKS_DIR="libs/go/mocks"
mkdir -p "$MOCKS_DIR"

# Generate mock for PaymentSyncService interface
echo "📦 Generating PaymentSyncService mock..."
mockgen -source=libs/go/client/payment_sync/interface.go \
    -destination=libs/go/mocks/payment_sync_service.go \
    -package=mocks \
    PaymentSyncService

# Generate mock for HTTP MetricsCollector interface  
echo "📦 Generating MetricsCollector mock..."
mockgen -source=libs/go/client/http/client.go \
    -destination=libs/go/mocks/metrics_collector.go \
    -package=mocks \
    MetricsCollector

# Generate mock for Circle client interface
echo "📦 Generating CircleClientInterface mock..."
mockgen -source=libs/go/client/circle/interface.go \
    -destination=libs/go/mocks/circle_client.go \
    -package=mocks \
    CircleClientInterface

# Look for other interfaces in common client packages
echo "📦 Scanning for additional interfaces..."

# Generate mocks for any database interfaces if they exist
if find libs/go/db -name "*.go" -exec grep -l "type.*interface" {} \; | head -1 >/dev/null 2>&1; then
    echo "📦 Found database interfaces, generating mocks..."
    for file in $(find libs/go/db -name "*.go" -exec grep -l "type.*interface" {} \;); do
        basename=$(basename "$file" .go)
        mockgen -source="$file" \
            -destination="libs/go/mocks/db_${basename}.go" \
            -package=mocks \
            $(grep "^type.*interface" "$file" | sed 's/type \([^ ]*\).*/\1/' | tr '\n' ' ')
    done
fi

# Generate convenience mock creation functions
echo "📦 Creating mock helpers..."
cat > "libs/go/mocks/helpers.go" << 'EOF'
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
EOF

echo "✅ Mock generation completed!"
echo "📁 Mocks generated in: $MOCKS_DIR"
echo ""
echo "Usage in tests:"
echo "  import \"github.com/cyphera/cyphera-api/libs/go/mocks\""
echo "  mockService := mocks.NewMockPaymentSyncService(t)"
echo "  mockService.EXPECT().GetServiceName().Return(\"test-service\")"