# Testing Dunning Email System with Resend

This guide walks you through testing the dunning email integration with Resend.

## Prerequisites

1. Resend API Key from [resend.com](https://resend.com)
2. Environment variables set:
   ```bash
   export RESEND_API_KEY="re_your_api_key_here"
   export EMAIL_FROM_ADDRESS="noreply@yourdomain.com"  # Optional
   export EMAIL_FROM_NAME="Your Company"               # Optional
   ```

## Step 1: Quick Email Test with Curl

First, verify that your Resend API key works:

```bash
# Run the curl test script
RESEND_API_KEY=re_your_api_key ./scripts/test-resend-curl.sh
```

This will send a test email to natefikru@gmail.com (or specify TEST_EMAIL=your@email.com).

## Step 2: Test Direct Go Integration

If you want to test the Go integration directly:

```bash
# Set your API key
export RESEND_API_KEY="re_your_api_key"

# Run the direct test
go run scripts/test-dunning-direct.go
```

## Step 3: Test Full Dunning System

### 3.1 Start the API Server

```bash
# Set all required environment variables
export RESEND_API_KEY="re_your_api_key"
export DATABASE_URL="postgresql://user:pass@localhost/cyphera"
# ... other required vars ...

# Start the server
make dev
```

### 3.2 Create Test Data

```bash
# First, get your workspace ID
WORKSPACE_ID="your-workspace-id"
API_KEY="your-api-key"

# Create a dunning configuration
curl -X POST http://localhost:8080/api/v1/dunning/configurations \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Email Config",
    "is_active": true,
    "is_default": true,
    "max_retry_attempts": 3,
    "retry_interval_days": [0, 1, 2],
    "attempt_actions": [
      {"attempt": 1, "actions": ["email", "retry_payment"]},
      {"attempt": 2, "actions": ["email", "retry_payment"]},
      {"attempt": 3, "actions": ["email", "retry_payment"]}
    ],
    "final_action": "cancel",
    "grace_period_hours": 0
  }'

# Save the configuration ID from the response
CONFIG_ID="..."
```

### 3.3 Create Test Campaign in Database

Connect to your database and run:

```sql
-- Insert test customer
INSERT INTO customers (id, workspace_id, email, name)
VALUES (
    gen_random_uuid(),
    'your-workspace-id'::uuid,
    'natefikru@gmail.com',
    'Test User'
);

-- Get the customer ID
SELECT id FROM customers WHERE email = 'natefikru@gmail.com';

-- Insert test dunning campaign (replace IDs)
INSERT INTO dunning_campaigns (
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
    'your-workspace-id'::uuid,
    'config-id-from-above'::uuid,
    'customer-id-from-above'::uuid,
    'active',
    'Test payment failed',
    9999, -- $99.99
    'USD',
    0,
    NOW() - INTERVAL '1 minute' -- Due immediately
);
```

### 3.4 Trigger Processing

```bash
# Process dunning campaigns
curl -X POST http://localhost:8080/api/v1/dunning/process?limit=10 \
  -H "Authorization: Bearer $API_KEY"
```

### 3.5 Check Results

1. Check your email inbox at natefikru@gmail.com
2. Check the API server logs for email sending status
3. Verify campaign status:

```bash
# List campaigns
curl http://localhost:8080/api/v1/dunning/campaigns \
  -H "Authorization: Bearer $API_KEY" | jq
```

## Troubleshooting

### Email Not Sending?

1. **Check API Key**: Ensure RESEND_API_KEY is set correctly
2. **Check Logs**: Look for errors in the API server logs
3. **Verify Domain**: For custom domains, ensure DNS records are configured
4. **Test with curl**: Use the curl script to isolate API issues

### Common Issues

1. **"From" address rejected**: Use `onboarding@resend.dev` for testing or verify your domain
2. **Rate limits**: Resend has rate limits; wait between tests
3. **Invalid API key**: Double-check your key starts with `re_`

## Using Custom Email Templates

Create custom templates via API:

```bash
curl -X POST http://localhost:8080/api/v1/dunning/email-templates \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Custom First Attempt",
    "template_type": "attempt_1",
    "subject": "Payment Failed - {{.CustomerName}}",
    "body_html": "<p>Custom HTML template...</p>",
    "is_active": true
  }'
```

## Production Setup

1. **Verify Your Domain**: Add Resend's DNS records
2. **Set From Address**: Use your verified domain
3. **Configure Templates**: Create professional templates
4. **Monitor Metrics**: Track open rates and bounces in Resend dashboard
5. **Set Up Webhooks**: Configure Resend webhooks for delivery status

## Email Template Variables

Available in all templates:
- `{{.CustomerName}}`
- `{{.CustomerEmail}}`
- `{{.Amount}}`
- `{{.Currency}}`
- `{{.ProductName}}`
- `{{.RetryDate}}`
- `{{.AttemptsRemaining}}`
- `{{.PaymentLink}}`
- `{{.SupportEmail}}`
- `{{.MerchantName}}`
- `{{.UnsubscribeLink}}`