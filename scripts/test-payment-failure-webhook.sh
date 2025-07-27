#!/bin/bash

# Test script for payment failure webhook

API_URL="http://localhost:8080/api/v1"
API_KEY="${API_KEY:-YOUR_API_KEY_HERE}"

# Function to print colored output
print_status() {
    if [ "$2" = "success" ]; then
        echo -e "\033[32m✓ $1\033[0m"
    elif [ "$2" = "error" ]; then
        echo -e "\033[31m✗ $1\033[0m"
    else
        echo -e "\033[33m→ $1\033[0m"
    fi
}

print_status "Testing Payment Failure Webhook"

# Check if API_KEY is set
if [ "$API_KEY" = "YOUR_API_KEY_HERE" ]; then
    print_status "Please set API_KEY environment variable" "error"
    exit 1
fi

# Test data
SUBSCRIPTION_ID="YOUR_SUBSCRIPTION_ID_HERE"
CUSTOMER_ID="YOUR_CUSTOMER_ID_HERE"

# Single payment failure webhook
print_status "Sending single payment failure webhook..."
RESPONSE=$(curl -s -X POST "$API_URL/webhooks/payment-failure" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d @- <<JSON
{
  "provider": "stripe",
  "subscription_id": "$SUBSCRIPTION_ID",
  "customer_id": "$CUSTOMER_ID",
  "amount_cents": 9999,
  "currency": "USD",
  "failure_reason": "insufficient_funds",
  "failed_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "metadata": {
    "stripe_charge_id": "ch_test_12345",
    "stripe_payment_intent": "pi_test_67890",
    "error_code": "card_declined",
    "decline_code": "insufficient_funds"
  }
}
JSON
)

echo "Response: $RESPONSE"

# Check if successful
if echo "$RESPONSE" | grep -q "processed"; then
    print_status "Payment failure webhook processed successfully" "success"
else
    print_status "Payment failure webhook failed" "error"
    echo "$RESPONSE"
fi

echo ""
print_status "Testing Batch Payment Failures..."

# Batch payment failures
BATCH_RESPONSE=$(curl -s -X POST "$API_URL/webhooks/payment-failures/batch" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d @- <<JSON
[
  {
    "provider": "stripe",
    "subscription_id": "$SUBSCRIPTION_ID",
    "customer_id": "$CUSTOMER_ID",
    "amount_cents": 4999,
    "currency": "USD",
    "failure_reason": "card_expired",
    "failed_at": "$(date -u -d '1 hour ago' +"%Y-%m-%dT%H:%M:%SZ")",
    "metadata": {
      "stripe_charge_id": "ch_test_expired_1"
    }
  },
  {
    "provider": "circle",
    "subscription_id": "$SUBSCRIPTION_ID",
    "customer_id": "$CUSTOMER_ID",
    "amount_cents": 19999,
    "currency": "USD",
    "failure_reason": "blockchain_congestion",
    "failed_at": "$(date -u -d '30 minutes ago' +"%Y-%m-%dT%H:%M:%SZ")",
    "metadata": {
      "transaction_hash": "0x123...",
      "gas_price": "100 gwei",
      "error": "replacement transaction underpriced"
    }
  }
]
JSON
)

echo "Batch Response: $BATCH_RESPONSE"

if echo "$BATCH_RESPONSE" | grep -q "completed"; then
    print_status "Batch payment failures processed" "success"
    
    # Extract results
    SUCCESS=$(echo "$BATCH_RESPONSE" | grep -o '"success":[0-9]*' | cut -d: -f2)
    FAILED=$(echo "$BATCH_RESPONSE" | grep -o '"failed":[0-9]*' | cut -d: -f2)
    
    print_status "Successful: $SUCCESS, Failed: $FAILED"
else
    print_status "Batch processing failed" "error"
fi

echo ""
print_status "Checking created dunning campaigns..."

# List dunning campaigns
CAMPAIGNS=$(curl -s -X GET "$API_URL/dunning/campaigns" \
  -H "Authorization: Bearer $API_KEY")

if echo "$CAMPAIGNS" | grep -q "$SUBSCRIPTION_ID"; then
    print_status "Dunning campaign created for subscription" "success"
    
    # Extract campaign details
    CAMPAIGN_ID=$(echo "$CAMPAIGNS" | grep -o '"id":"[^"]*' | head -1 | cut -d'"' -f4)
    echo "Campaign ID: $CAMPAIGN_ID"
    
    # Get campaign details
    CAMPAIGN_DETAILS=$(curl -s -X GET "$API_URL/dunning/campaigns/$CAMPAIGN_ID" \
      -H "Authorization: Bearer $API_KEY")
    
    echo ""
    echo "Campaign Details:"
    echo "$CAMPAIGN_DETAILS" | jq '.'
else
    print_status "No dunning campaign found" "error"
fi

echo ""
print_status "Test completed!"
echo ""
echo "📝 Notes:"
echo "- Replace SUBSCRIPTION_ID and CUSTOMER_ID with valid IDs from your database"
echo "- The webhook will create a failed payment event and trigger dunning campaign creation"
echo "- Check the dunning campaigns list to see the created campaign"
echo "- Use the manual process endpoint to test email sending: POST /api/v1/dunning/process"