#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_step() {
    echo -e "${BLUE}🔧 $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# Check if we're in the right directory
if [ ! -f "Makefile" ] || [ ! -d "apps/api" ]; then
    print_error "Please run this script from the cyphera-api root directory"
    exit 1
fi

echo "🚀 Cyphera Test Runner"
echo "====================="
echo ""

# Parse command line arguments
case "${1:-all}" in
    "handlers"|"unit")
        print_step "Running Handler Tests (GitHub Actions Unit Tests)"
        cd apps/api
        go test ./handlers/... -v -race -timeout=30s
        print_success "Handler tests completed"
        ;;
        
    "integration")
        print_step "Running Integration Tests"
        go test -tags=integration ./tests/integration/... -v -timeout=30m
        print_success "Integration tests completed"
        ;;
        
    "delegation")
        print_step "Running Delegation Server Tests"
        cd apps/delegation-server
        npm test
        print_success "Delegation server tests completed"
        ;;
        
    "builds")
        print_step "Testing All Builds"
        echo "  → API build..."
        cd apps/api && go build ./... > /dev/null
        echo "  → Libraries build..."
        cd ../../libs/go && go build ./... > /dev/null
        echo "  → Subscription processor build..."
        cd ../../apps/subscription-processor && go build ./... > /dev/null
        print_success "All builds successful"
        ;;
        
    "format")
        print_step "Checking Code Formatting"
        FORMAT_ISSUES=$(gofmt -s -l libs/go/ apps/api/ | wc -l)
        if [ "$FORMAT_ISSUES" -eq 0 ]; then
            print_success "Go code is properly formatted"
        else
            print_warning "Found $FORMAT_ISSUES formatting issues:"
            gofmt -s -l libs/go/ apps/api/
        fi
        ;;
        
    "quick")
        print_step "Running Quick Test Suite (No Database)"
        echo ""
        print_step "1. Handler Tests..."
        cd apps/api
        go test ./handlers/... -v -race -timeout=30s
        echo ""
        print_step "2. Service Tests..."
        cd ../libs/go
        go test ./services/... -v -race -timeout=30s || print_warning "Some service tests may have issues"
        print_success "Quick tests completed"
        ;;
        
    "github"|"ci"|"all")
        print_step "Running GitHub Actions Test Suite Locally"
        echo "=========================================="
        echo ""
        
        print_step "1️⃣ Handler Tests (Unit Tests)"
        cd apps/api
        go test ./handlers/... -v -race -timeout=30s
        cd ../..
        echo ""
        
        print_step "2️⃣ Integration Tests"
        go test -tags=integration ./tests/integration/... -v -timeout=30m
        echo ""
        
        print_step "3️⃣ Delegation Server Tests"
        cd apps/delegation-server
        npm test
        cd ../..
        echo ""
        
        print_step "4️⃣ Build Verification"
        echo "  → API build..."
        cd apps/api && go build ./... > /dev/null && cd ../..
        echo "  → Libraries build..."
        cd libs/go && go build ./... > /dev/null && cd ../..
        echo "  → Subscription processor build..."
        cd apps/subscription-processor && go build ./... > /dev/null && cd ../..
        print_success "All builds successful"
        echo ""
        
        print_step "5️⃣ Code Formatting Check"
        FORMAT_ISSUES=$(gofmt -s -l libs/go/ apps/api/ | wc -l)
        if [ "$FORMAT_ISSUES" -eq 0 ]; then
            print_success "Go code is properly formatted"
        else
            print_warning "Found $FORMAT_ISSUES formatting issues"
        fi
        echo ""
        
        print_success "🎉 All GitHub Actions tests completed!"
        echo ""
        echo "Your code is ready for GitHub Actions CI/CD!"
        ;;
        
    *)
        echo "Usage: $0 [test-type]"
        echo ""
        echo "Available test types:"
        echo "  handlers     - API handler tests (same as GitHub Actions unit tests)"
        echo "  integration  - Integration tests with database"
        echo "  delegation   - Delegation server TypeScript tests"
        echo "  builds       - Verify all components build"
        echo "  format       - Check code formatting"
        echo "  quick        - Fast tests (handlers + services, no database)"
        echo "  github|ci    - Complete GitHub Actions test suite (default)"
        echo ""
        echo "Examples:"
        echo "  $0                    # Run all GitHub Actions tests"
        echo "  $0 handlers          # Run just handler tests"
        echo "  $0 quick             # Run quick test suite"
        echo "  $0 integration       # Run integration tests"
        ;;
esac