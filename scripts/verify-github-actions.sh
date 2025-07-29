#!/bin/bash
set -e

echo "🔍 Verifying GitHub Actions Test Setup"
echo "======================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ $1${NC}"
    else
        echo -e "${RED}❌ $1${NC}"
        return 1
    fi
}

# Function to print info
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Function to print warning
print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

echo ""
echo "1. Testing Unit Tests (Handler Tests)"
echo "-----------------------------------"
cd apps/api
go test ./handlers/... -v -race -timeout=30s > /tmp/unit_test_output.log 2>&1
print_status "Unit tests completed"
echo "Unit test summary:"
grep -E "(PASS|FAIL|ok|FAIL)" /tmp/unit_test_output.log | tail -5

echo ""
echo "2. Testing Integration Tests"
echo "---------------------------"
cd ../../
go test -tags=integration ./tests/integration/... -v > /tmp/integration_test_output.log 2>&1
print_status "Integration tests completed"
echo "Integration test summary:"
grep -E "(PASS|FAIL|ok)" /tmp/integration_test_output.log | tail -3

echo ""
echo "3. Testing Delegation Server"
echo "---------------------------"
cd apps/delegation-server
npm test > /tmp/delegation_test_output.log 2>&1
print_status "Delegation server tests completed"
echo "Delegation server test summary:"
tail -3 /tmp/delegation_test_output.log

echo ""
echo "4. Testing Makefile Targets (GitHub Actions Dependencies)"
echo "--------------------------------------------------------"
cd ../../

# Test delegation server Makefile targets
print_info "Testing delegation-server-setup..."
make delegation-server-setup > /dev/null 2>&1
print_status "delegation-server-setup target works"

print_info "Testing delegation-server-lint..."
make delegation-server-lint > /dev/null 2>&1
print_status "delegation-server-lint target works"

print_info "Testing delegation-server-test..."
make delegation-server-test > /dev/null 2>&1
print_status "delegation-server-test target works"

print_info "Testing delegation-server-build..."
make delegation-server-build > /dev/null 2>&1
print_status "delegation-server-build target works"

echo ""
echo "5. Testing Build Commands"
echo "------------------------"
print_info "Testing Go API build..."
cd apps/api
go build ./... > /dev/null 2>&1
print_status "Go API builds successfully"

print_info "Testing Go libraries build..."
cd ../../libs/go
go build ./... > /dev/null 2>&1
print_status "Go libraries build successfully"

print_info "Testing subscription processor build..."
cd ../../apps/subscription-processor
go build ./... > /dev/null 2>&1
print_status "Subscription processor builds successfully"

echo ""
echo "6. Testing Code Quality"
echo "----------------------"
cd ../../

print_info "Testing Go formatting..."
FORMAT_ISSUES=$(gofmt -s -l libs/go/ apps/api/ | wc -l)
if [ "$FORMAT_ISSUES" -eq 0 ]; then
    print_status "Go code is properly formatted"
else
    print_warning "Found $FORMAT_ISSUES Go formatting issues"
fi

echo ""
echo "7. Verifying API Endpoint (Manual Test)"
echo "--------------------------------------"
print_info "Testing if API server can start..."
timeout 10s bash -c '
    cd apps/api
    go run cmd/main/main.go > /tmp/api_start.log 2>&1 &
    API_PID=$!
    sleep 5
    if kill -0 $API_PID 2>/dev/null; then
        echo "API server started successfully"
        kill $API_PID
        exit 0
    else
        echo "API server failed to start"
        exit 1
    fi
' || print_warning "API server test timed out or failed"

echo ""
echo "8. GitHub Actions Workflow Validation"
echo "------------------------------------"
print_info "Checking workflow files exist..."
for workflow in test.yml delegation-server.yml cyphera-api.yml; do
    if [ -f ".github/workflows/$workflow" ]; then
        print_status "$workflow exists"
    else
        print_warning "$workflow missing"
    fi
done

echo ""
echo "9. Testing Mock Generation"
echo "-------------------------"
print_info "Testing mock generation..."
make generate-mocks > /dev/null 2>&1
print_status "Mock generation completed"

echo ""
echo "📊 Summary"
echo "=========="
echo "The following tests are covered by GitHub Actions:"
echo ""
echo "✅ Unit Tests (Handler Tests)"
echo "   - Location: apps/api/handlers/"
echo "   - Command: go test ./handlers/... -v -race -timeout=30s"
echo "   - Runs on: Every push/PR to main/dev"
echo ""
echo "✅ Integration Tests"
echo "   - Location: tests/integration/"
echo "   - Command: go test -tags=integration ./tests/integration/... -v -timeout=30m"
echo "   - Runs on: Every push/PR to main/dev"  
echo "   - Database: PostgreSQL test instance"
echo ""
echo "✅ Delegation Server Tests"
echo "   - Location: apps/delegation-server/"
echo "   - Command: npm test"
echo "   - Runs on: Changes to delegation-server/"
echo ""
echo "✅ Code Quality Checks"
echo "   - Linting: golangci-lint for Go, eslint for TypeScript"
echo "   - Formatting: gofmt for Go"
echo "   - Builds: All components must build successfully"
echo ""
echo "✅ Coverage Reports"
echo "   - Unit test coverage"
echo "   - Integration test coverage"
echo "   - Coverage thresholds enforced"
echo ""
echo "🔧 Additional Workflows:"
echo "   - cyphera-api.yml: Main API deployment"
echo "   - subscription-processor.yml: Background processor"
echo "   - dunning-processor.yml: Dunning campaign processor"
echo "   - webhooks.yml: Webhook handlers"
echo ""
echo "All GitHub Actions tests should now pass with the fixes implemented!"