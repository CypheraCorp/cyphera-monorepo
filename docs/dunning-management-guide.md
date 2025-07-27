# Dunning Management System Guide

The Cyphera API now includes a comprehensive dunning management system for handling failed recurring payments. This guide explains how to configure and use the dunning system.

## Overview

Dunning management automates the process of recovering failed payments through:
- Configurable retry schedules
- Automated email notifications
- Multiple retry attempts with different strategies
- Final actions for unrecoverable payments (cancel, pause, downgrade)
- Comprehensive analytics and reporting

## Database Schema

The dunning system uses the following tables:
- `dunning_configurations` - Workspace-specific retry configurations
- `dunning_campaigns` - Active dunning processes for failed payments
- `dunning_attempts` - History of retry attempts
- `dunning_email_templates` - Email templates for customer communication
- `dunning_analytics` - Aggregated performance metrics

## API Endpoints

### Configuration Management

**Create Dunning Configuration**
```bash
POST /api/v1/dunning/configurations
{
  "name": "Standard Retry Policy",
  "description": "Default retry policy for failed payments",
  "is_active": true,
  "is_default": true,
  "max_retry_attempts": 4,
  "retry_interval_days": [3, 7, 7, 7],
  "attempt_actions": [
    {
      "attempt": 1,
      "actions": ["email", "retry_payment"],
      "email_template_id": "uuid-here"
    },
    {
      "attempt": 2,
      "actions": ["email", "in_app", "retry_payment"]
    },
    {
      "attempt": 3,
      "actions": ["email", "retry_payment"]
    },
    {
      "attempt": 4,
      "actions": ["email", "retry_payment"]
    }
  ],
  "final_action": "cancel",
  "final_action_config": {},
  "send_pre_dunning_reminder": true,
  "pre_dunning_days": 3,
  "allow_customer_retry": true,
  "grace_period_hours": 24
}
```

**Get Configuration**
```bash
GET /api/v1/dunning/configurations/{id}
```

**List Configurations**
```bash
GET /api/v1/dunning/configurations
```

### Campaign Management

**List Dunning Campaigns**
```bash
GET /api/v1/dunning/campaigns?status=active&customer_id=uuid&limit=20&offset=0
```

**Get Campaign Details**
```bash
GET /api/v1/dunning/campaigns/{id}
```

**Pause Campaign**
```bash
POST /api/v1/dunning/campaigns/{id}/pause
```

**Resume Campaign**
```bash
POST /api/v1/dunning/campaigns/{id}/resume
```

### Email Template Management

**Create Email Template**
```bash
POST /api/v1/dunning/email-templates
{
  "name": "First Payment Retry",
  "template_type": "attempt_1",
  "subject": "Payment Failed - Action Required",
  "body_html": "<p>Hi {{customer_name}},</p><p>Your payment of {{amount}} failed...</p>",
  "body_text": "Hi {{customer_name}}, Your payment of {{amount}} failed...",
  "available_variables": ["customer_name", "amount", "retry_date", "product_name"],
  "is_active": true
}
```

**List Email Templates**
```bash
GET /api/v1/dunning/email-templates
```

### Analytics

**Get Campaign Statistics**
```bash
GET /api/v1/dunning/stats?start_date=2024-01-01&end_date=2024-12-31
```

## Integration with Payment Processing

When a payment fails, the system automatically:

1. Creates a dunning campaign
2. Waits for the configured grace period (default 24 hours)
3. Executes the first retry attempt
4. Continues with subsequent attempts based on the retry schedule
5. Takes final action if all retries fail

## Example Retry Schedule

A typical retry schedule might look like:
- **Attempt 1**: 3 days after failure - Send email + retry payment
- **Attempt 2**: 10 days after failure - Send email + in-app notification + retry payment
- **Attempt 3**: 17 days after failure - Send urgent email + retry payment
- **Attempt 4**: 24 days after failure - Final notice email + last retry
- **Final Action**: Cancel subscription if all attempts fail

## Email Template Variables

Available variables for email templates:
- `{{customer_name}}` - Customer's full name
- `{{customer_email}}` - Customer's email address
- `{{amount}}` - Failed payment amount
- `{{currency}}` - Payment currency
- `{{product_name}}` - Subscription product name
- `{{retry_date}}` - Next retry date
- `{{attempts_remaining}}` - Number of retry attempts left
- `{{payment_link}}` - Link for manual payment retry
- `{{support_email}}` - Merchant support email

## Best Practices

1. **Grace Period**: Allow 24-48 hours before first retry to handle temporary issues
2. **Retry Intervals**: Space retries appropriately (3, 7, 7, 7 days is common)
3. **Communication**: Send clear, helpful emails at each attempt
4. **Final Actions**: Consider downgrading to free plan instead of canceling
5. **Analytics**: Monitor recovery rates and adjust strategies accordingly

## Testing

To test the dunning system:

1. Create a test configuration
2. Simulate a failed payment
3. Monitor the campaign progress
4. Verify emails are sent
5. Check retry attempts are executed
6. Confirm final action is taken

## Integration Status

### ✅ Completed Integrations

1. **Email Service Integration (Resend)**
   - Integrated Resend for sending transactional emails
   - Template rendering with variable substitution
   - Error handling and retry logic
   - Configuration via environment variables:
     - `RESEND_API_KEY` or `RESEND_API_KEY_ARN` (AWS Secrets Manager)
     - `EMAIL_FROM_ADDRESS` (default: noreply@cypherapay.com)
     - `EMAIL_FROM_NAME` (default: Cyphera)

### TODO: Remaining Integrations

1. **Payment Processor Integration**
   - Hook into failed payment events
   - Implement actual payment retry via delegation server

3. **Subscription Management**
   - Implement subscription pause/cancel/downgrade actions
   - Update subscription status based on dunning results

4. **Webhook Notifications**
   - Send webhooks for dunning events
   - Allow merchants to track dunning progress

5. **Customer Portal**
   - Show dunning status to customers
   - Allow manual payment retry
   - Display payment history

## Monitoring

Key metrics to monitor:
- Recovery rate by attempt number
- Average time to recovery
- Lost revenue from failed campaigns
- Email engagement rates
- Customer response rates

Use the analytics endpoint to track these metrics and optimize your dunning strategy.

## Email Configuration with Resend

The dunning system uses Resend for sending transactional emails. To configure:

1. **Get Resend API Key**: Sign up at [resend.com](https://resend.com) and get your API key
2. **Set Environment Variables**:
   ```bash
   export RESEND_API_KEY="re_123456789"
   export EMAIL_FROM_ADDRESS="dunning@yourdomain.com"
   export EMAIL_FROM_NAME="Your Company"
   ```
3. **Verify Domain**: Add Resend's DNS records to your domain for better deliverability
4. **Test Email Sending**: Use the manual process endpoint to trigger test emails

### Email Template Variables

The following variables are available in email templates:
- `{{.CustomerName}}` - Customer's full name
- `{{.CustomerEmail}}` - Customer's email address  
- `{{.Amount}}` - Failed payment amount (formatted with currency)
- `{{.Currency}}` - Payment currency code
- `{{.ProductName}}` - Subscription product name
- `{{.RetryDate}}` - Next retry date (formatted)
- `{{.AttemptsRemaining}}` - Number of retry attempts left
- `{{.PaymentLink}}` - Link for manual payment retry
- `{{.SupportEmail}}` - Merchant support email
- `{{.MerchantName}}` - Merchant/company name
- `{{.UnsubscribeLink}}` - Unsubscribe link

### Default Email Templates

The system includes default templates for:
- `pre_dunning` - Sent before payment is due
- `attempt_1` - First retry attempt
- `attempt_2` - Second retry attempt  
- `final_notice` - Final attempt before action
- `recovery_success` - Sent when payment succeeds

You can customize these templates via the API or use the defaults.