# Multi-Workspace Payment Sync Testing Guide

**Version:** 1.0  
**Date:** December 2024  
**Purpose:** Comprehensive testing plan for validating the multi-workspace payment sync implementation

## Phase 1: Environment Setup

### 1.1 Test Stripe Accounts Setup
Create two separate Stripe test accounts to validate workspace isolation:

**Account A (Primary Test):**
- Create at: https://dashboard.stripe.com/register
- Name: "Cyphera Test Workspace A"
- Enable test mode
- Note down: `sk_test_...` and webhook endpoint secret

**Account B (Isolation Test):**
- Create at: https://dashboard.stripe.com/register  
- Name: "Cyphera Test Workspace B"
- Enable test mode
- Note down: `sk_test_...` and webhook endpoint secret

### 1.2 Local Development Environment
```bash
# 1. Ensure database is running and migrated
make db-up
make db-migrate

# 2. Generate latest database code
sqlc generate

# 3. Build all components
make build

# 4. Start API server locally
make run-api

# 5. In separate terminals, test Lambda functions locally
make test-webhook-receiver
make test-webhook-processor
```

### 1.3 Test Workspaces in Database
```sql
-- Create test workspaces in your database
INSERT INTO workspaces (id, name, account_id, livemode) VALUES 
  ('01234567-89ab-cdef-0123-456789abcdef', 'Test Workspace A', (SELECT id FROM accounts LIMIT 1), false),
  ('11234567-89ab-cdef-0123-456789abcdef', 'Test Workspace B', (SELECT id FROM accounts LIMIT 1), false);
```

## Phase 2: API Endpoint Testing

### 2.1 Configure Payment Provider for Workspace A
```bash
# Test workspace A configuration
curl -X POST http://localhost:8080/api/v1/workspaces/01234567-89ab-cdef-0123-456789abcdef/payment-configurations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "provider_name": "stripe",
    "is_active": true,
    "is_test_mode": true,
    "configuration": {
      "api_key": "sk_test_YOUR_STRIPE_KEY_A",
      "webhook_secret": "whsec_YOUR_WEBHOOK_SECRET_A",
      "environment": "test"
    },
    "webhook_endpoint_url": "https://your-domain.com/webhooks/stripe/01234567-89ab-cdef-0123-456789abcdef"
  }'
```

### 2.2 Configure Payment Provider for Workspace B
```bash
# Test workspace B configuration (for isolation testing)
curl -X POST http://localhost:8080/api/v1/workspaces/11234567-89ab-cdef-0123-456789abcdef/payment-configurations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "provider_name": "stripe",
    "is_active": true,
    "is_test_mode": true,
    "configuration": {
      "api_key": "sk_test_YOUR_STRIPE_KEY_B",
      "webhook_secret": "whsec_YOUR_WEBHOOK_SECRET_B",
      "environment": "test"
    },
    "webhook_endpoint_url": "https://your-domain.com/webhooks/stripe/11234567-89ab-cdef-0123-456789abcdef"
  }'
```

### 2.3 Verify Configuration
```bash
# List configurations for workspace A
curl -X GET http://localhost:8080/api/v1/workspaces/01234567-89ab-cdef-0123-456789abcdef/payment-configurations \
  -H "Authorization: Bearer YOUR_API_KEY"

# Test connection for workspace A
curl -X POST http://localhost:8080/api/v1/workspaces/01234567-89ab-cdef-0123-456789abcdef/payment-configurations/test-connection/stripe \
  -H "Authorization: Bearer YOUR_API_KEY"
```

## Phase 3: Initial Sync Testing

### 3.1 Prepare Test Data in Stripe
**In Stripe Account A Dashboard:**
1. Create 2-3 test customers
2. Create 1-2 test products
3. Create 2-3 prices for the products
4. Create 1-2 test subscriptions
5. Generate 1-2 test invoices

**In Stripe Account B Dashboard:**
1. Create different test customers (to verify isolation)
2. Create different products/prices

### 3.2 Test Initial Sync for Workspace A
```bash
# Start initial sync
curl -X POST http://localhost:8080/api/v1/workspaces/01234567-89ab-cdef-0123-456789abcdef/sync/stripe/initial \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "entity_types": ["customers", "products", "prices", "subscriptions", "invoices"],
    "batch_size": 10,
    "full_sync": true
  }'

# Monitor sync progress
curl -X GET http://localhost:8080/api/v1/workspaces/01234567-89ab-cdef-0123-456789abcdef/sync/sessions \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### 3.3 Verify Data Sync
```sql
-- Check that data was synced to workspace A only
SELECT 'customers' as entity_type, count(*) as count, workspace_id 
FROM customers 
WHERE workspace_id = '01234567-89ab-cdef-0123-456789abcdef'
UNION ALL
SELECT 'products', count(*), workspace_id 
FROM products 
WHERE workspace_id = '01234567-89ab-cdef-0123-456789abcdef'
UNION ALL
SELECT 'prices', count(*), workspace_id 
FROM prices p 
JOIN products pr ON p.product_id = pr.id 
WHERE pr.workspace_id = '01234567-89ab-cdef-0123-456789abcdef'
UNION ALL
SELECT 'subscriptions', count(*), workspace_id 
FROM subscriptions s 
JOIN customers c ON s.customer_id = c.id 
WHERE c.workspace_id = '01234567-89ab-cdef-0123-456789abcdef';

-- Verify no data leaked to workspace B
SELECT count(*) as leaked_customers FROM customers 
WHERE workspace_id = '11234567-89ab-cdef-0123-456789abcdef';
```

## Phase 4: Webhook Testing

### 4.1 Setup Webhook Endpoints in Stripe
**For Stripe Account A:**
1. Go to Stripe Dashboard > Developers > Webhooks
2. Add endpoint: `https://your-domain.com/webhooks/stripe/01234567-89ab-cdef-0123-456789abcdef`
3. Select events:
   - `customer.created`, `customer.updated`, `customer.deleted`
   - `product.created`, `product.updated`
   - `price.created`, `price.updated`
   - `invoice.created`, `invoice.updated`, `invoice.paid`
   - `customer.subscription.created`, `customer.subscription.updated`, `customer.subscription.deleted`

**For Stripe Account B:**
1. Add endpoint: `https://your-domain.com/webhooks/stripe/11234567-89ab-cdef-0123-456789abcdef`
2. Select same events

### 4.2 Test Webhook Processing
```bash
# Monitor webhook processing logs
tail -f /var/log/webhook-receiver.log &
tail -f /var/log/webhook-processor.log &

# In Stripe Dashboard A: Create a new customer
# This should trigger a webhook

# Verify webhook was processed
curl -X GET http://localhost:8080/api/v1/workspaces/01234567-89ab-cdef-0123-456789abcdef/sync/sessions?session_type=webhook_sync \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### 4.3 Test Real-time Updates
**Test Customer Update:**
1. In Stripe Dashboard A: Update a customer's email
2. Verify the update appears in your database:
```sql
SELECT * FROM customers 
WHERE workspace_id = '01234567-89ab-cdef-0123-456789abcdef' 
ORDER BY updated_at DESC LIMIT 5;
```

**Test Invoice Payment:**
1. In Stripe Dashboard A: Mark an invoice as paid
2. Verify the status update:
```sql
SELECT * FROM invoices 
WHERE workspace_id = '01234567-89ab-cdef-0123-456789abcdef' 
AND status = 'paid' 
ORDER BY updated_at DESC;
```

## Phase 5: Multi-Workspace Isolation Testing

### 5.1 Cross-Workspace Data Verification
```sql
-- Verify workspace A and B have separate data
SELECT 
  workspace_id,
  COUNT(*) as customer_count
FROM customers 
GROUP BY workspace_id;

-- Verify no cross-contamination
SELECT 
  c.workspace_id as customer_workspace,
  s.workspace_id as subscription_workspace_via_customer
FROM customers c
LEFT JOIN subscriptions s ON c.id = s.customer_id
WHERE c.workspace_id != (
  SELECT workspace_id FROM customers c2 WHERE c2.id = s.customer_id
);
-- Should return 0 rows
```

### 5.2 Test Webhook Routing
1. **Trigger webhook from Stripe Account A**
   - Should only update workspace A data
2. **Trigger webhook from Stripe Account B**
   - Should only update workspace B data
3. **Verify isolation:**
```sql
-- Check recent webhook events by workspace
SELECT 
  workspace_id,
  provider_name,
  event_type,
  COUNT(*) as event_count,
  MAX(occurred_at) as last_event
FROM payment_sync_events 
WHERE webhook_event_id IS NOT NULL
GROUP BY workspace_id, provider_name, event_type
ORDER BY workspace_id, last_event DESC;
```

## Phase 6: Error Handling Testing

### 6.1 Test Invalid API Key
```bash
# Test with invalid API key
curl -X POST http://localhost:8080/api/v1/workspaces/01234567-89ab-cdef-0123-456789abcdef/payment-configurations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "provider_name": "stripe",
    "is_active": true,
    "is_test_mode": true,
    "configuration": {
      "api_key": "sk_test_invalid_key",
      "webhook_secret": "whsec_invalid",
      "environment": "test"
    }
  }'
# Should return error about invalid credentials
```

### 6.2 Test Webhook Signature Validation
```bash
# Send webhook with invalid signature (should be rejected)
curl -X POST https://your-domain.com/webhooks/stripe/01234567-89ab-cdef-0123-456789abcdef \
  -H "Content-Type: application/json" \
  -H "Stripe-Signature: invalid_signature" \
  -d '{"id": "evt_test", "type": "customer.created"}'
# Should return 400 Bad Request
```

## Phase 7: Performance Testing

### 7.1 Batch Processing Test
```bash
# Test large initial sync
curl -X POST http://localhost:8080/api/v1/workspaces/01234567-89ab-cdef-0123-456789abcdef/sync/stripe/initial \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "entity_types": ["customers", "products", "prices", "subscriptions"],
    "batch_size": 100,
    "full_sync": true
  }'

# Monitor memory/CPU usage during sync
top -p $(pgrep -f webhook-processor)
```

### 7.2 Concurrent Webhook Test
Use a tool like `vegeta` or create a simple script to send multiple webhooks simultaneously and verify they're all processed correctly.

## Phase 8: Production Readiness Checklist

### 8.1 Security Verification
- [ ] API keys are encrypted in database
- [ ] Webhook signatures are validated
- [ ] No sensitive data in logs
- [ ] HTTPS endpoints only
- [ ] Rate limiting works

### 8.2 Monitoring Setup
- [ ] CloudWatch alarms firing correctly
- [ ] SQS queue depth monitoring
- [ ] Lambda error rates monitoring
- [ ] Database connection monitoring

### 8.3 Backup & Recovery
- [ ] Database backups configured
- [ ] Failed webhook replay mechanism tested
- [ ] DLQ processing tested

## Expected Results

**✅ Success Criteria:**
- Both workspaces can configure Stripe independently
- Initial sync completes without errors
- Webhooks route to correct workspace
- Real-time updates work
- No data cross-contamination
- Error handling works as expected

**🚨 Red Flags:**
- Data appearing in wrong workspace
- Webhook signature validation failures
- Initial sync timeouts or errors
- API configuration errors
- Cross-workspace data leakage

## Troubleshooting Common Issues

### Issue: Webhook not received
**Check:**
1. Webhook URL is correct in Stripe dashboard
2. SSL certificate is valid
3. API Gateway is deployed correctly
4. Lambda functions have proper permissions

### Issue: Initial sync fails
**Check:**
1. API key is valid and has correct permissions
2. Rate limiting isn't exceeded
3. Database connections are working
4. Memory limits in Lambda functions

### Issue: Data in wrong workspace
**Check:**
1. Webhook routing logic in webhook-receiver
2. Workspace resolution in webhook-processor
3. Database queries include workspace_id filters

Remember to test both success and failure scenarios to ensure your system is robust! 