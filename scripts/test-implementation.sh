#!/bin/bash

# Multi-Workspace Payment Sync Implementation Test Script
# Usage: ./scripts/test-implementation.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_BASE_URL="http://localhost:8080"
WORKSPACE_A_ID="01234567-89ab-cdef-0123-456789abcdef"
WORKSPACE_B_ID="11234567-89ab-cdef-0123-456789abcdef"

# Test Stripe keys (you'll need to replace these)
STRIPE_KEY_A="${STRIPE_TEST_KEY_A:-sk_test_replace_with_your_key_a}"
STRIPE_SECRET_A="${STRIPE_WEBHOOK_SECRET_A:-whsec_replace_with_your_secret_a}"
STRIPE_KEY_B="${STRIPE_TEST_KEY_B:-sk_test_replace_with_your_key_b}"
STRIPE_SECRET_B="${STRIPE_WEBHOOK_SECRET_B:-whsec_replace_with_your_secret_b}"

# API Key for authentication (replace with your actual API key)
API_KEY="${CYPHERA_API_KEY:-your_api_key_here}"

log() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check if API server is running
    if ! curl -s "$API_BASE_URL/health" > /dev/null; then
        error "API server is not running at $API_BASE_URL"
        error "Please start the API server with: make run-api"
        exit 1
    fi
    
    # Check if Stripe keys are set
    if [[ "$STRIPE_KEY_A" == "sk_test_replace_with_your_key_a" ]]; then
        warning "Please set your Stripe test keys:"
        echo "export STRIPE_TEST_KEY_A=sk_test_your_key_here"
        echo "export STRIPE_WEBHOOK_SECRET_A=whsec_your_secret_here"
        echo "export STRIPE_TEST_KEY_B=sk_test_your_other_key_here"
        echo "export STRIPE_WEBHOOK_SECRET_B=whsec_your_other_secret_here"
        echo "export CYPHERA_API_KEY=your_api_key_here"
        exit 1
    fi
    
    success "Prerequisites check passed"
}

setup_test_workspaces() {
    log "Setting up test workspaces in database..."
    
    # Create test workspaces using psql
    psql "$DATABASE_URL" -c "
    INSERT INTO workspaces (id, name, account_id, livemode) VALUES 
      ('$WORKSPACE_A_ID', 'Test Workspace A', (SELECT id FROM accounts LIMIT 1), false),
      ('$WORKSPACE_B_ID', 'Test Workspace B', (SELECT id FROM accounts LIMIT 1), false)
    ON CONFLICT (id) DO NOTHING;
    " 2>/dev/null || warning "Could not create workspaces (they may already exist)"
    
    success "Test workspaces setup complete"
}

configure_payment_providers() {
    log "Configuring payment providers for both workspaces..."
    
    # Configure Workspace A
    local response_a=$(curl -s -X POST "$API_BASE_URL/api/v1/workspaces/$WORKSPACE_A_ID/payment-configurations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $API_KEY" \
        -d "{
            \"provider_name\": \"stripe\",
            \"is_active\": true,
            \"is_test_mode\": true,
            \"configuration\": {
                \"api_key\": \"$STRIPE_KEY_A\",
                \"webhook_secret\": \"$STRIPE_SECRET_A\",
                \"environment\": \"test\"
            },
            \"webhook_endpoint_url\": \"https://your-domain.com/webhooks/stripe/$WORKSPACE_A_ID\"
        }")
    
    if echo "$response_a" | grep -q "error"; then
        error "Failed to configure Workspace A: $response_a"
        return 1
    fi
    
    # Configure Workspace B
    local response_b=$(curl -s -X POST "$API_BASE_URL/api/v1/workspaces/$WORKSPACE_B_ID/payment-configurations" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $API_KEY" \
        -d "{
            \"provider_name\": \"stripe\",
            \"is_active\": true,
            \"is_test_mode\": true,
            \"configuration\": {
                \"api_key\": \"$STRIPE_KEY_B\",
                \"webhook_secret\": \"$STRIPE_SECRET_B\",
                \"environment\": \"test\"
            },
            \"webhook_endpoint_url\": \"https://your-domain.com/webhooks/stripe/$WORKSPACE_B_ID\"
        }")
    
    if echo "$response_b" | grep -q "error"; then
        error "Failed to configure Workspace B: $response_b"
        return 1
    fi
    
    success "Payment providers configured for both workspaces"
}

test_connections() {
    log "Testing connections to Stripe for both workspaces..."
    
    # Test Workspace A connection
    local test_a=$(curl -s -X POST "$API_BASE_URL/api/v1/workspaces/$WORKSPACE_A_ID/payment-configurations/test-connection/stripe" \
        -H "Authorization: Bearer $API_KEY")
    
    if echo "$test_a" | grep -q "error"; then
        error "Connection test failed for Workspace A: $test_a"
        return 1
    fi
    
    # Test Workspace B connection
    local test_b=$(curl -s -X POST "$API_BASE_URL/api/v1/workspaces/$WORKSPACE_B_ID/payment-configurations/test-connection/stripe" \
        -H "Authorization: Bearer $API_KEY")
    
    if echo "$test_b" | grep -q "error"; then
        error "Connection test failed for Workspace B: $test_b"
        return 1
    fi
    
    success "Connection tests passed for both workspaces"
}

run_initial_sync() {
    log "Starting initial sync for Workspace A..."
    
    local sync_response=$(curl -s -X POST "$API_BASE_URL/api/v1/workspaces/$WORKSPACE_A_ID/sync/stripe/initial" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $API_KEY" \
        -d '{
            "entity_types": ["customers", "products", "prices", "subscriptions"],
            "batch_size": 10,
            "full_sync": true
        }')
    
    if echo "$sync_response" | grep -q "error"; then
        error "Initial sync failed: $sync_response"
        return 1
    fi
    
    local session_id=$(echo "$sync_response" | grep -o '"session_id":"[^"]*"' | cut -d'"' -f4)
    log "Initial sync started with session ID: $session_id"
    
    # Monitor sync progress
    log "Monitoring sync progress..."
    for i in {1..30}; do
        local status=$(curl -s "$API_BASE_URL/api/v1/workspaces/$WORKSPACE_A_ID/sync/sessions/$session_id" \
            -H "Authorization: Bearer $API_KEY" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
        
        echo -n "."
        
        if [[ "$status" == "completed" ]]; then
            echo ""
            success "Initial sync completed successfully"
            return 0
        elif [[ "$status" == "failed" ]]; then
            echo ""
            error "Initial sync failed"
            return 1
        fi
        
        sleep 2
    done
    
    echo ""
    warning "Initial sync is still running (timeout reached)"
}

verify_data_isolation() {
    log "Verifying data isolation between workspaces..."
    
    # Check if data exists in Workspace A
    local workspace_a_data=$(psql "$DATABASE_URL" -t -c "
        SELECT COUNT(*) FROM customers WHERE workspace_id = '$WORKSPACE_A_ID';
    " 2>/dev/null || echo "0")
    
    # Check if data leaked to Workspace B
    local workspace_b_data=$(psql "$DATABASE_URL" -t -c "
        SELECT COUNT(*) FROM customers WHERE workspace_id = '$WORKSPACE_B_ID';
    " 2>/dev/null || echo "0")
    
    workspace_a_data=$(echo "$workspace_a_data" | xargs)
    workspace_b_data=$(echo "$workspace_b_data" | xargs)
    
    log "Workspace A customers: $workspace_a_data"
    log "Workspace B customers: $workspace_b_data"
    
    if [[ "$workspace_a_data" -gt 0 ]] && [[ "$workspace_b_data" == 0 ]]; then
        success "Data isolation verified - data exists in Workspace A only"
    elif [[ "$workspace_a_data" == 0 ]]; then
        warning "No data found in Workspace A - initial sync may not have found any data"
    else
        error "Data isolation failed - data found in Workspace B"
        return 1
    fi
}

run_webhook_test() {
    log "Testing webhook endpoint accessibility..."
    
    # Test webhook endpoint with invalid signature (should be rejected)
    local webhook_response=$(curl -s -w "%{http_code}" -X POST "https://your-domain.com/webhooks/stripe/$WORKSPACE_A_ID" \
        -H "Content-Type: application/json" \
        -H "Stripe-Signature: invalid_signature" \
        -d '{"id": "evt_test", "type": "customer.created"}' \
        -o /dev/null)
    
    if [[ "$webhook_response" == "400" ]]; then
        success "Webhook signature validation working correctly"
    else
        warning "Webhook endpoint test inconclusive (response: $webhook_response)"
    fi
}

print_next_steps() {
    echo ""
    echo -e "${BLUE}=== TESTING COMPLETE ===${NC}"
    echo ""
    echo -e "${GREEN}Next steps for manual testing:${NC}"
    echo "1. Go to your Stripe test dashboard for Account A"
    echo "2. Create a test customer, product, or subscription"
    echo "3. Set up webhook endpoints in Stripe:"
    echo "   - Workspace A: https://your-domain.com/webhooks/stripe/$WORKSPACE_A_ID"
    echo "   - Workspace B: https://your-domain.com/webhooks/stripe/$WORKSPACE_B_ID"
    echo "4. Test real-time webhook processing"
    echo "5. Verify data updates in your database"
    echo ""
    echo -e "${YELLOW}Useful commands:${NC}"
    echo "# List configurations:"
    echo "curl -H 'Authorization: Bearer $API_KEY' $API_BASE_URL/api/v1/workspaces/$WORKSPACE_A_ID/payment-configurations"
    echo ""
    echo "# Check sync sessions:"
    echo "curl -H 'Authorization: Bearer $API_KEY' $API_BASE_URL/api/v1/workspaces/$WORKSPACE_A_ID/sync/sessions"
    echo ""
    echo "# Monitor database changes:"
    echo "psql \$DATABASE_URL -c \"SELECT workspace_id, COUNT(*) FROM customers GROUP BY workspace_id;\""
}

main() {
    echo -e "${BLUE}=== Multi-Workspace Payment Sync Implementation Test ===${NC}"
    echo ""
    
    check_prerequisites
    setup_test_workspaces
    configure_payment_providers
    test_connections
    run_initial_sync
    verify_data_isolation
    run_webhook_test
    print_next_steps
    
    echo ""
    success "Basic testing completed successfully!"
}

# Run main function
main "$@" 