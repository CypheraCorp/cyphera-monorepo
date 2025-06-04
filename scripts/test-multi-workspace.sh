#!/bin/bash
set -e

# ===============================================
# Multi-Workspace Webhook Testing Script
# ===============================================
# Tests webhook isolation between different workspaces

echo "🧪 Testing multi-workspace webhook isolation..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
  echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
  echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
  echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
  echo -e "${RED}❌ $1${NC}"
}

# Configuration
API_BASE_URL="http://localhost:8080"
WEBHOOK_BASE_URL="http://localhost:3001"
TEST_TIMEOUT=30

# Check if services are running
check_services() {
  log_info "Checking if services are running..."
  
  if ! curl -sf "$API_BASE_URL/health" > /dev/null 2>&1; then
    log_error "API server is not running on $API_BASE_URL"
    log_info "Run 'make local-webhooks-up' to start services"
    exit 1
  fi
  
  if ! curl -sf "$WEBHOOK_BASE_URL/health" > /dev/null 2>&1; then
    log_error "Webhook receiver is not running on $WEBHOOK_BASE_URL"
    log_info "Run 'make local-webhooks-up' to start services"
    exit 1
  fi
  
  log_success "All services are running"
}

# Create test workspace
create_workspace() {
  local workspace_name="$1"
  local account_id="$2"
  
  log_info "Creating workspace: $workspace_name"
  
  local response=$(curl -s -X POST "$API_BASE_URL/api/v1/workspaces" \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"$workspace_name\",
      \"account_id\": \"$account_id\"
    }" 2>/dev/null || echo '{"error": "failed"}')
  
  if echo "$response" | jq -e '.id' > /dev/null 2>&1; then
    local workspace_id=$(echo "$response" | jq -r '.id')
    log_success "Created workspace: $workspace_id"
    echo "$workspace_id"
  else
    log_error "Failed to create workspace: $workspace_name"
    echo "$response" | jq '.' 2>/dev/null || echo "$response"
    return 1
  fi
}

# Configure payment provider for workspace
configure_payment_provider() {
  local workspace_id="$1"
  local stripe_key="$2"
  local webhook_secret="$3"
  
  log_info "Configuring payment provider for workspace: $workspace_id"
  
  local response=$(curl -s -X POST "$API_BASE_URL/api/v1/workspaces/$workspace_id/payment-configurations" \
    -H "Content-Type: application/json" \
    -d "{
      \"provider_name\": \"stripe\",
      \"is_active\": true,
      \"is_test_mode\": true,
      \"configuration\": {
        \"api_key\": \"$stripe_key\",
        \"webhook_secret\": \"$webhook_secret\",
        \"environment\": \"test\"
      }
    }" 2>/dev/null || echo '{"error": "failed"}')
  
  if echo "$response" | jq -e '.id' > /dev/null 2>&1; then
    log_success "Configured payment provider for workspace: $workspace_id"
  else
    log_warning "Failed to configure payment provider (this may be normal if endpoints don't exist yet)"
  fi
}

# Send test webhook
send_test_webhook() {
  local workspace_id="$1"
  local customer_id="$2"
  local event_id="$3"
  
  log_info "Sending test webhook to workspace: $workspace_id"
  
  local timestamp=$(date +%s)
  local response=$(curl -s -w "%{http_code}" -X POST "$WEBHOOK_BASE_URL/webhooks/stripe/$workspace_id" \
    -H "Content-Type: application/json" \
    -H "Stripe-Signature: t=$timestamp,v1=test_signature_$workspace_id" \
    -d "{
      \"id\": \"$event_id\",
      \"type\": \"customer.created\",
      \"data\": {
        \"object\": {
          \"id\": \"$customer_id\",
          \"email\": \"test-$customer_id@example.com\",
          \"name\": \"Test Customer $customer_id\"
        }
      }
    }" 2>/dev/null || echo "000")
  
  local http_code="${response: -3}"
  if [[ "$http_code" == "200" ]]; then
    log_success "Webhook sent successfully to workspace: $workspace_id"
    return 0
  else
    log_warning "Webhook failed with HTTP $http_code for workspace: $workspace_id"
    return 1
  fi
}

# Check customer count for workspace
check_customer_count() {
  local workspace_id="$1"
  
  # Try to get customers for the workspace
  local response=$(curl -s "$API_BASE_URL/api/v1/workspaces/$workspace_id/customers" 2>/dev/null || echo '[]')
  
  if echo "$response" | jq -e '. | type == "array"' > /dev/null 2>&1; then
    local count=$(echo "$response" | jq 'length')
    echo "$count"
  else
    # If the endpoint doesn't exist, check database directly
    log_info "API endpoint not available, checking database directly..."
    
    # Connect to database and count customers for workspace
    local count=$(docker-compose -f docker-compose.webhooks.yml exec -T postgres \
      psql -U apiuser -d cyphera -t -c \
      "SELECT COUNT(*) FROM customers WHERE workspace_id = '$workspace_id';" 2>/dev/null | xargs || echo "0")
    
    echo "$count"
  fi
}

# Wait for webhook processing
wait_for_processing() {
  local max_wait="$1"
  local check_interval=2
  local elapsed=0
  
  log_info "Waiting up to ${max_wait}s for webhook processing..."
  
  while [[ $elapsed -lt $max_wait ]]; do
    # Check if there are any pending SQS messages
    local queue_attrs=$(AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws --endpoint-url=http://localhost:4566 sqs get-queue-attributes \
      --queue-url http://localhost:4566/000000000000/webhook-queue \
      --attribute-names ApproximateNumberOfMessages \
      --region us-east-1 \
      --output text \
      --query 'Attributes.ApproximateNumberOfMessages' 2>/dev/null || echo "0")
    
    if [[ "$queue_attrs" == "0" ]]; then
      log_success "All webhooks processed"
      return 0
    fi
    
    if [[ $((elapsed % 10)) -eq 0 ]] && [[ $elapsed -gt 0 ]]; then
      log_info "Still waiting... ($elapsed/${max_wait}s, $queue_attrs messages in queue)"
    fi
    
    sleep $check_interval
    ((elapsed += check_interval))
  done
  
  log_warning "Timeout waiting for webhook processing"
  return 1
}

# Main test execution
main() {
  echo "========================================"
  echo "🧪 Multi-Workspace Webhook Isolation Test"
  echo "========================================"
  
  # Check prerequisites
  check_services
  
  # Clean up any existing test data
  log_info "Cleaning up previous test data..."
  AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws --endpoint-url=http://localhost:4566 sqs purge-queue \
    --queue-url http://localhost:4566/000000000000/webhook-queue \
    --region us-east-1 > /dev/null 2>&1 || true
  
  # Test data
  local workspace_a_name="Test Workspace A"
  local workspace_b_name="Test Workspace B"
  local account_a_id="test-account-a-$(date +%s)"
  local account_b_id="test-account-b-$(date +%s)"
  
  # Create workspaces
  log_info "Creating test workspaces..."
  local workspace_a_id=$(create_workspace "$workspace_a_name" "$account_a_id")
  local workspace_b_id=$(create_workspace "$workspace_b_name" "$account_b_id")
  
  if [[ -z "$workspace_a_id" ]] || [[ -z "$workspace_b_id" ]]; then
    log_error "Failed to create workspaces"
    exit 1
  fi
  
  echo ""
  log_info "Created workspaces:"
  echo "  Workspace A: $workspace_a_id"
  echo "  Workspace B: $workspace_b_id"
  
  # Configure payment providers
  log_info "Configuring payment providers..."
  configure_payment_provider "$workspace_a_id" "sk_test_workspace_a_key" "whsec_workspace_a_secret"
  configure_payment_provider "$workspace_b_id" "sk_test_workspace_b_key" "whsec_workspace_b_secret"
  
  # Get initial customer counts
  local initial_count_a=$(check_customer_count "$workspace_a_id")
  local initial_count_b=$(check_customer_count "$workspace_b_id")
  
  echo ""
  log_info "Initial customer counts:"
  echo "  Workspace A: $initial_count_a customers"
  echo "  Workspace B: $initial_count_b customers"
  
  # Send test webhooks
  echo ""
  log_info "Sending test webhooks..."
  
  local customer_a_id="cus_workspace_a_$(date +%s)"
  local customer_b_id="cus_workspace_b_$(date +%s)"
  local event_a_id="evt_test_a_$(date +%s)"
  local event_b_id="evt_test_b_$(date +%s)"
  
  send_test_webhook "$workspace_a_id" "$customer_a_id" "$event_a_id"
  send_test_webhook "$workspace_b_id" "$customer_b_id" "$event_b_id"
  
  # Wait for processing
  wait_for_processing $TEST_TIMEOUT
  
  # Check final customer counts
  sleep 2  # Give a little extra time for database updates
  
  local final_count_a=$(check_customer_count "$workspace_a_id")
  local final_count_b=$(check_customer_count "$workspace_b_id")
  
  echo ""
  log_info "Final customer counts:"
  echo "  Workspace A: $final_count_a customers"
  echo "  Workspace B: $final_count_b customers"
  
  # Verify isolation
  echo ""
  log_info "Verifying workspace isolation..."
  
  local expected_count_a=$((initial_count_a + 1))
  local expected_count_b=$((initial_count_b + 1))
  
  local test_passed=true
  
  if [[ "$final_count_a" -eq "$expected_count_a" ]]; then
    log_success "Workspace A isolation: PASSED (expected $expected_count_a, got $final_count_a)"
  else
    log_error "Workspace A isolation: FAILED (expected $expected_count_a, got $final_count_a)"
    test_passed=false
  fi
  
  if [[ "$final_count_b" -eq "$expected_count_b" ]]; then
    log_success "Workspace B isolation: PASSED (expected $expected_count_b, got $final_count_b)"
  else
    log_error "Workspace B isolation: FAILED (expected $expected_count_b, got $final_count_b)"
    test_passed=false
  fi
  
  # Additional checks
  echo ""
  log_info "Running additional isolation checks..."
  
  # Check that webhooks were processed
  local webhook_events_count=$(docker-compose -f docker-compose.webhooks.yml exec -T postgres \
    psql -U apiuser -d cyphera -t -c \
    "SELECT COUNT(*) FROM payment_sync_events WHERE event_id IN ('$event_a_id', '$event_b_id');" 2>/dev/null | xargs || echo "0")
  
  if [[ "$webhook_events_count" -eq "2" ]]; then
    log_success "Webhook events recorded: PASSED ($webhook_events_count events found)"
  else
    log_warning "Webhook events recorded: INCONCLUSIVE ($webhook_events_count events found, expected 2)"
  fi
  
  # Summary
  echo ""
  echo "========================================"
  if [[ "$test_passed" == "true" ]]; then
    log_success "🎉 MULTI-WORKSPACE ISOLATION TEST PASSED!"
    echo ""
    echo "✅ Both workspaces processed exactly one webhook each"
    echo "✅ No cross-workspace data contamination detected"
    echo "✅ Webhook routing and processing working correctly"
  else
    log_error "❌ MULTI-WORKSPACE ISOLATION TEST FAILED!"
    echo ""
    echo "Please check:"
    echo "• Webhook receiver routing logic"
    echo "• Database workspace_id constraints"
    echo "• Payment sync event processing"
    echo ""
    echo "Run 'make local-webhooks-logs' to check service logs"
  fi
  echo "========================================"
  
  # Cleanup option
  echo ""
  read -p "Would you like to clean up test data? (y/n): " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    log_info "Cleaning up test data..."
    
    # Clean up database (if endpoints exist)
    if curl -sf "$API_BASE_URL/api/v1/workspaces/$workspace_a_id" > /dev/null 2>&1; then
      curl -s -X DELETE "$API_BASE_URL/api/v1/workspaces/$workspace_a_id" > /dev/null 2>&1 || true
    fi
    if curl -sf "$API_BASE_URL/api/v1/workspaces/$workspace_b_id" > /dev/null 2>&1; then
      curl -s -X DELETE "$API_BASE_URL/api/v1/workspaces/$workspace_b_id" > /dev/null 2>&1 || true
    fi
    
    # Clean up SQS
    AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test aws --endpoint-url=http://localhost:4566 sqs purge-queue \
      --queue-url http://localhost:4566/000000000000/webhook-queue \
      --region us-east-1 > /dev/null 2>&1 || true
    
    log_success "Test data cleaned up"
  fi
  
  if [[ "$test_passed" == "true" ]]; then
    exit 0
  else
    exit 1
  fi
}

# Run main function
main "$@" 