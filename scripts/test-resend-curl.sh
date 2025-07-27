#!/bin/bash

# Simple Resend API test using curl
# Usage: RESEND_API_KEY=re_xxx ./test-resend-curl.sh

RESEND_API_KEY="${RESEND_API_KEY}"
TEST_EMAIL="${TEST_EMAIL:-natefikru@gmail.com}"

if [ -z "$RESEND_API_KEY" ]; then
    echo "❌ Error: RESEND_API_KEY environment variable is required"
    echo "Usage: RESEND_API_KEY=re_xxx ./test-resend-curl.sh"
    exit 1
fi

echo "📧 Testing Resend API..."
echo "Sending test email to: $TEST_EMAIL"
echo ""

# Send test email using Resend API
RESPONSE=$(curl -s -X POST https://api.resend.com/emails \
  -H "Authorization: Bearer $RESEND_API_KEY" \
  -H "Content-Type: application/json" \
  -d @- <<EOF
{
  "from": "Cyphera <onboarding@resend.dev>",
  "to": ["$TEST_EMAIL"],
  "subject": "Test Dunning Email from Cyphera",
  "html": "<h1>Test Dunning Email</h1><p>Hello from Cyphera!</p><p>This is a test email to verify that the Resend integration is working correctly.</p><p><strong>Test Details:</strong></p><ul><li>Amount Due: \$99.99</li><li>Next Retry: Tomorrow</li><li>Payment Method: Update Required</li></ul><p><a href='https://app.cyphera.com' style='background-color:#28a745;color:white;padding:10px 20px;text-decoration:none;border-radius:5px;display:inline-block;'>Update Payment Method</a></p>",
  "text": "Test Dunning Email\n\nHello from Cyphera!\n\nThis is a test email to verify that the Resend integration is working correctly.\n\nTest Details:\n- Amount Due: \$99.99\n- Next Retry: Tomorrow\n- Payment Method: Update Required\n\nUpdate Payment Method: https://app.cyphera.com"
}
EOF
)

# Check if email was sent successfully
if echo "$RESPONSE" | grep -q '"id"'; then
    EMAIL_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*' | cut -d'"' -f4)
    echo "✅ Email sent successfully!"
    echo "Email ID: $EMAIL_ID"
    echo ""
    echo "📬 Check your inbox at: $TEST_EMAIL"
else
    echo "❌ Failed to send email"
    echo "Response: $RESPONSE"
    exit 1
fi

echo ""
echo "Next steps to test the full dunning system:"
echo "1. Set these environment variables:"
echo "   export RESEND_API_KEY=$RESEND_API_KEY"
echo "   export EMAIL_FROM_ADDRESS=noreply@yourdomain.com"
echo "   export EMAIL_FROM_NAME=\"Your Company\""
echo ""
echo "2. Start your API server"
echo "3. Create test data and trigger dunning process"