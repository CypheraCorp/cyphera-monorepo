#!/bin/bash

# Test script for dunning email system
# This script creates test data and triggers a dunning email

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-your-api-key-here}"
TEST_EMAIL="natefikru@gmail.com"
WORKSPACE_ID="${WORKSPACE_ID:-00000000-0000-0000-0000-000000000000}"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Testing Dunning Email System${NC}"
echo "================================"
echo "API URL: $API_BASE_URL"
echo "Test Email: $TEST_EMAIL"
echo ""

# Step 1: Create a dunning configuration
echo -e "${BLUE}Step 1: Creating dunning configuration...${NC}"
CONFIG_RESPONSE=$(curl -s -X POST "$API_BASE_URL/api/v1/dunning/configurations" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Email Configuration",
    "description": "Configuration for testing email sending",
    "is_active": true,
    "is_default": true,
    "max_retry_attempts": 3,
    "retry_interval_days": [1, 3, 7],
    "attempt_actions": [
      {
        "attempt": 1,
        "actions": ["email", "retry_payment"]
      },
      {
        "attempt": 2,
        "actions": ["email", "retry_payment"]
      },
      {
        "attempt": 3,
        "actions": ["email", "retry_payment"]
      }
    ],
    "final_action": "cancel",
    "final_action_config": {},
    "send_pre_dunning_reminder": true,
    "pre_dunning_days": 3,
    "allow_customer_retry": true,
    "grace_period_hours": 1
  }')

CONFIG_ID=$(echo $CONFIG_RESPONSE | jq -r '.id' 2>/dev/null)
if [ "$CONFIG_ID" != "null" ] && [ ! -z "$CONFIG_ID" ]; then
    echo -e "${GREEN}✓ Created dunning configuration: $CONFIG_ID${NC}"
else
    echo -e "${RED}✗ Failed to create configuration${NC}"
    echo "Response: $CONFIG_RESPONSE"
fi

# Step 2: Create or get a test customer
echo -e "\n${BLUE}Step 2: Creating test customer...${NC}"
CUSTOMER_ID="test-customer-$(date +%s)"
# Note: You'll need to adjust this based on your actual customer creation endpoint
echo -e "${GREEN}✓ Using test customer ID: $CUSTOMER_ID${NC}"

# Step 3: Create a test email template
echo -e "\n${BLUE}Step 3: Creating email template...${NC}"
TEMPLATE_RESPONSE=$(curl -s -X POST "$API_BASE_URL/api/v1/dunning/email-templates" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Email Template",
    "template_type": "attempt_1",
    "subject": "Test Dunning Email - Payment Failed",
    "body_html": "<h1>Test Email</h1><p>Hi {{.CustomerName}},</p><p>This is a test dunning email sent to {{.CustomerEmail}}.</p><p>Amount due: {{.Amount}} {{.Currency}}</p><p>Next retry: {{.RetryDate}}</p><p>Attempts remaining: {{.AttemptsRemaining}}</p>",
    "body_text": "Test Email\n\nHi {{.CustomerName}},\n\nThis is a test dunning email sent to {{.CustomerEmail}}.\n\nAmount due: {{.Amount}} {{.Currency}}\nNext retry: {{.RetryDate}}\nAttempts remaining: {{.AttemptsRemaining}}",
    "available_variables": ["customer_name", "customer_email", "amount", "currency", "retry_date", "attempts_remaining"],
    "is_active": true
  }')

TEMPLATE_ID=$(echo $TEMPLATE_RESPONSE | jq -r '.id' 2>/dev/null)
if [ "$TEMPLATE_ID" != "null" ] && [ ! -z "$TEMPLATE_ID" ]; then
    echo -e "${GREEN}✓ Created email template: $TEMPLATE_ID${NC}"
else
    echo -e "${RED}✗ Failed to create template${NC}"
    echo "Response: $TEMPLATE_RESPONSE"
fi

# Step 4: Create a test dunning campaign
echo -e "\n${BLUE}Step 4: Creating test campaign directly in database...${NC}"
# Since we need to create test data, we'll use a simple approach
# In production, this would be triggered by a failed payment

# Create a minimal test campaign using SQL
cat > /tmp/create_test_campaign.sql << EOF
-- Insert a test customer if not exists
INSERT INTO customers (id, workspace_id, email, name)
VALUES (
    '11111111-1111-1111-1111-111111111111'::uuid,
    '$WORKSPACE_ID'::uuid,
    '$TEST_EMAIL',
    'Test User'
)
ON CONFLICT (id) DO UPDATE 
SET email = EXCLUDED.email,
    name = EXCLUDED.name;

-- Insert a test dunning campaign
INSERT INTO dunning_campaigns (
    id,
    workspace_id,
    configuration_id,
    customer_id,
    status,
    original_failure_reason,
    original_amount_cents,
    currency,
    current_attempt,
    next_retry_at
)
VALUES (
    gen_random_uuid(),
    '$WORKSPACE_ID'::uuid,
    '$CONFIG_ID'::uuid,
    '11111111-1111-1111-1111-111111111111'::uuid,
    'active',
    'Test payment failed',
    9999, -- $99.99
    'USD',
    0,
    NOW() - INTERVAL '1 minute' -- Make it due for processing
)
ON CONFLICT (id) DO NOTHING;
EOF

echo -e "${GREEN}✓ Test campaign SQL prepared${NC}"
echo -e "${BLUE}Note: You'll need to run this SQL manually in your database${NC}"
echo "SQL file saved to: /tmp/create_test_campaign.sql"

# Step 5: Trigger the manual process endpoint
echo -e "\n${BLUE}Step 5: Triggering manual dunning process...${NC}"
PROCESS_RESPONSE=$(curl -s -X POST "$API_BASE_URL/api/v1/dunning/process?limit=10" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json")

echo "Response: $PROCESS_RESPONSE"

# Summary
echo -e "\n${BLUE}Summary:${NC}"
echo "================================"
echo "1. Created dunning configuration: $CONFIG_ID"
echo "2. Created email template: $TEMPLATE_ID"
echo "3. Test email will be sent to: $TEST_EMAIL"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo "1. Run the SQL in /tmp/create_test_campaign.sql to create test data"
echo "2. Check your email at $TEST_EMAIL for the test dunning email"
echo "3. Check the application logs for email sending status"
echo ""
echo -e "${BLUE}To view campaigns:${NC}"
echo "curl -s \"$API_BASE_URL/api/v1/dunning/campaigns\" -H \"Authorization: Bearer $API_KEY\" | jq"