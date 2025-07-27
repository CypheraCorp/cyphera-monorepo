# Automated Dunning Campaign Creation

This document describes how the automated payment failure detection and dunning campaign creation system works in Cyphera.

## Overview

The system automatically detects failed payments and creates dunning campaigns to recover revenue. This happens in two ways:

1. **Subscription Processor Integration**: During scheduled subscription processing, failed payments are detected and campaigns are created
2. **Webhook Integration**: Payment providers can send failure notifications via webhook to trigger immediate campaign creation

## Architecture

### Components

1. **PaymentFailureDetector Service** (`/libs/go/services/payment_failure_detector.go`)
   - Detects failed payment events
   - Creates dunning campaigns automatically
   - Implements campaign strategies based on customer history

2. **Subscription Processor Enhancement** (`/apps/subscription-processor/`)
   - Runs on AWS Lambda with EventBridge scheduling
   - Processes subscriptions and detects failures
   - Triggers the PaymentFailureDetector

3. **Payment Failure Webhook Handler** (`/apps/api/handlers/payment_failure_webhook_handler.go`)
   - Receives failure notifications from payment providers
   - Creates failed payment events
   - Triggers campaign creation

## Campaign Creation Logic

### Detection Process

1. **Failed Event Detection**
   ```go
   // Look for recent failed events
   failedEvents := ListRecentSubscriptionEventsByType("failed", lookbackTime)
   ```

2. **Duplicate Prevention**
   - Checks if an active campaign already exists for the subscription
   - Prevents multiple campaigns for the same failure

3. **Configuration Selection**
   - Uses workspace's active dunning configuration
   - Creates default configuration if none exists

### Campaign Strategies

The system determines campaign strategy based on customer payment history:

| Strategy | Criteria | Description |
|----------|----------|-------------|
| `new_customer` | < 3 total payments | First-time or very new customers |
| `premium` | ≥90% success rate & amount ≥$100 | High-value reliable customers |
| `standard` | ≥80% success rate | Good payment history |
| `cautious` | ≥50% success rate | Mixed payment history |
| `high_risk` | <50% success rate | Poor payment history |

### Default Configuration

If no dunning configuration exists, the system creates one with:
- **Max Attempts**: 4
- **Retry Schedule**: [3, 7, 7, 7] days
- **Grace Period**: 24 hours

## Integration Points

### 1. Subscription Processor

```go
// In HandleRequest
results, err := app.subscriptionProcessor.ProcessDueSubscriptions(ctx)
if results.Failed > 0 {
    detectionResult, err := app.failureDetector.DetectAndCreateCampaigns(ctx, 10)
}
```

### 2. Webhook Endpoints

#### Single Payment Failure
```bash
POST /api/v1/webhooks/payment-failure
{
  "provider": "stripe",
  "subscription_id": "uuid",
  "customer_id": "uuid",
  "amount_cents": 9999,
  "currency": "USD",
  "failure_reason": "insufficient_funds",
  "failed_at": "2024-01-20T10:00:00Z",
  "metadata": {...}
}
```

#### Batch Payment Failures
```bash
POST /api/v1/webhooks/payment-failures/batch
[
  {...},
  {...}
]
```

## Database Schema

### New Queries
- `ListRecentFailedPayments`: Find failed payments without campaigns
- `GetFailedPaymentCount`: Count failures for a subscription
- `CheckExistingDunningCampaign`: Check for active campaigns
- `GetSubscriptionPaymentHistory`: Get payment history for strategy

## Configuration

### Environment Variables
- No additional environment variables required
- Uses existing database and service configurations

### Lambda Schedule
- Subscription processor runs every 1-5 minutes (configurable)
- Failed payment detection happens after each run
- Lookback window: 10 minutes (adjustable)

## Monitoring

### Logs
```
INFO: Found failed subscription events count=3 since=2024-01-20T09:50:00Z
INFO: Created dunning campaign for failed payment campaign_id=xxx subscription_id=xxx strategy=standard
INFO: Payment failure detection completed failed_events=3 campaigns_created=2 campaigns_skipped=1
```

### Metrics
- Failed events detected
- Campaigns created vs skipped
- Strategy distribution
- Processing errors

## Testing

### Manual Testing

1. **Simulate Failed Payment**
   ```bash
   # Create a failed subscription event
   ./scripts/test-payment-failure-webhook.sh
   ```

2. **Trigger Detection Manually**
   ```bash
   # Run subscription processor locally
   cd apps/subscription-processor
   go run cmd/main.go
   ```

3. **Check Results**
   ```bash
   # List dunning campaigns
   curl -X GET http://localhost:8080/api/v1/dunning/campaigns \
     -H "Authorization: Bearer $API_KEY"
   ```

### Integration Testing

1. Create test subscription
2. Simulate payment failure via webhook
3. Verify campaign creation
4. Check email notifications
5. Verify duplicate prevention

## Best Practices

1. **Lookback Window**: Set based on Lambda execution frequency
   - 5-minute schedule → 10-minute lookback
   - 1-minute schedule → 5-minute lookback

2. **Strategy Tuning**: Adjust strategy thresholds based on business needs

3. **Configuration Management**: Create workspace-specific configurations for different retry strategies

4. **Monitoring**: Set up CloudWatch alarms for:
   - High failure rates
   - Campaign creation failures
   - Processing errors

## Troubleshooting

### Common Issues

1. **No Campaigns Created**
   - Check if subscription events exist
   - Verify workspace has active configuration
   - Check for existing active campaigns

2. **Duplicate Campaigns**
   - Verify duplicate prevention logic
   - Check lookback window vs execution frequency

3. **Wrong Strategy Applied**
   - Review payment history calculation
   - Check strategy threshold configuration

### Debug Commands

```sql
-- Check recent failed events
SELECT * FROM subscription_events 
WHERE event_type IN ('failed', 'failed_redemption')
AND occurred_at > NOW() - INTERVAL '1 hour';

-- Check active campaigns
SELECT * FROM dunning_campaigns
WHERE status IN ('active', 'paused')
AND workspace_id = 'xxx';

-- Check payment history
SELECT event_type, COUNT(*) 
FROM subscription_events
WHERE subscription_id = 'xxx'
GROUP BY event_type;
```

## Future Enhancements

1. **Machine Learning Strategy**: Use ML to predict optimal retry schedules
2. **Real-time Processing**: Process failures immediately via SQS/SNS
3. **Custom Strategies**: Allow workspace-specific strategy definitions
4. **A/B Testing**: Test different retry schedules and measure effectiveness