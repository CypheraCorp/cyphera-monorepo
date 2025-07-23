# Subscription Processor

> **Navigation:** [← Root README](../../README.md) | [Main API →](../api/README.md) | [Architecture →](../../docs/architecture.md)

The subscription processor is a Go background service that handles automated billing for cryptocurrency subscriptions using stored delegation credentials.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Development](#development)
- [Processing Logic](#processing-logic)
- [Database Integration](#database-integration)
- [Error Handling](#error-handling)
- [Monitoring & Logging](#monitoring--logging)
- [Deployment](#deployment)

## Overview

The subscription processor runs as a scheduled background job that identifies due subscriptions, processes payments through the delegation server, and updates subscription records with comprehensive error handling and retry logic.

### Key Features
- **Automated Billing** - Processes recurring subscription payments
- **Delegation Management** - Uses stored delegation credentials for payments
- **Retry Logic** - Handles failed payments with exponential backoff
- **Event Logging** - Comprehensive audit trail for all operations
- **Dead Letter Queuing** - Manages permanently failed subscriptions
- **Multi-tenant Processing** - Workspace-aware subscription handling
- **Monitoring Integration** - CloudWatch metrics and alerts

### Processing Flow
1. **Discovery** - Find subscriptions due for payment
2. **Validation** - Verify delegation credentials and balances
3. **Payment** - Execute blockchain transactions via delegation server
4. **Recording** - Update subscription status and log events
5. **Retry Handling** - Queue failed payments for retry
6. **Notification** - Alert on payment success/failure

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Scheduler     │    │   Background    │    │   Dead Letter   │
│   (Cron/AWS)    │    │   Jobs Queue    │    │   Queue         │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │ Triggers
                    ┌────────────▼────────────┐
                    │  Subscription Processor │
                    │        (Go)             │
                    └────────────┬────────────┘
                                 │
         ┌───────────────────────┼───────────────────────┐
         │                       │                       │
┌────────▼─────────┐ ┌───────────▼────────┐ ┌───────────▼────────┐
│   Database       │ │   Delegation       │ │   Notification     │
│   (PostgreSQL)   │ │   Server (gRPC)    │ │   Service          │
│                  │ │                    │ │   (Email/Webhook)  │
└──────────────────┘ └────────────────────┘ └────────────────────┘
```

### Directory Structure

```
apps/subscription-processor/
├── cmd/
│   └── main.go                # Entry point
├── internal/
│   └── processor/             # Core processing logic
│       └── processor.go
├── go.mod                     # Go module dependencies
├── go.sum                     # Dependency checksums
└── README.md                  # This file
```

## Development

### Prerequisites
- Go 1.21 or later
- PostgreSQL database access
- Delegation server running
- Environment variables configured

### Installation
```bash
# From project root
npm run install:go

# Or directly with Go
cd apps/subscription-processor
go mod download
```

### Running Locally

#### Start Processor
```bash
# From project root
npm run dev:subscription

# Or directly with Go
cd apps/subscription-processor
go run cmd/main.go
```

#### Environment Variables
The processor uses the same `.env` file as other services:

```bash
# Database
DATABASE_URL="postgresql://postgres:postgres@localhost:5432/cyphera_dev"

# gRPC Services
DELEGATION_GRPC_ADDR="localhost:50051"

# Processing Configuration
SUBSCRIPTION_PROCESSOR_INTERVAL="5m"    # How often to run
SUBSCRIPTION_PROCESSOR_TIMEOUT="30s"    # Max processing time
MAX_RETRY_ATTEMPTS="3"                   # Failed payment retries
RETRY_BACKOFF_MULTIPLIER="2"            # Exponential backoff

# Logging
LOG_LEVEL="info"
NODE_ENV="development"
```

### Development Commands
```bash
# Run processor once
go run cmd/main.go

# Run with verbose logging
LOG_LEVEL=debug go run cmd/main.go

# Build binary
go build -o subscription-processor cmd/main.go

# Run tests
go test -v ./...

# Run with race detection
go test -race ./...
```

## Processing Logic

### Main Processing Loop
```go
// cmd/main.go
func main() {
    processor := internal.NewProcessor(db, delegationClient, logger)
    
    // Run continuously with interval
    ticker := time.NewTicker(processingInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            if err := processor.ProcessDueSubscriptions(); err != nil {
                logger.Error("Processing failed", "error", err)
            }
        case <-ctx.Done():
            logger.Info("Shutting down processor")
            return
        }
    }
}
```

### Subscription Discovery
```go
// internal/processor/processor.go
func (p *Processor) ProcessDueSubscriptions() error {
    // Find subscriptions due for payment
    dueSubscriptions, err := p.db.GetDueSubscriptions(ctx, time.Now())
    if err != nil {
        return fmt.Errorf("failed to get due subscriptions: %w", err)
    }
    
    logger.Info("Found due subscriptions", "count", len(dueSubscriptions))
    
    // Process each subscription
    for _, subscription := range dueSubscriptions {
        if err := p.processSubscription(ctx, subscription); err != nil {
            logger.Error("Failed to process subscription", 
                "subscription_id", subscription.ID,
                "error", err)
        }
    }
    
    return nil
}
```

### Individual Subscription Processing
```go
func (p *Processor) processSubscription(ctx context.Context, sub Subscription) error {
    // Start transaction for atomic updates
    tx, err := p.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Validate subscription and delegation
    if err := p.validateSubscription(ctx, sub); err != nil {
        return p.handleValidationError(ctx, sub, err)
    }
    
    // Execute payment via delegation server
    paymentResult, err := p.executePayment(ctx, sub)
    if err != nil {
        return p.handlePaymentError(ctx, sub, err)
    }
    
    // Update subscription record
    if err := p.updateSubscription(ctx, tx, sub, paymentResult); err != nil {
        return err
    }
    
    // Log successful payment event
    if err := p.logSubscriptionEvent(ctx, tx, sub, "payment_succeeded", paymentResult); err != nil {
        return err
    }
    
    return tx.Commit()
}
```

### Payment Execution
```go
func (p *Processor) executePayment(ctx context.Context, sub Subscription) (*PaymentResult, error) {
    // Get delegation credentials
    delegation, err := p.db.GetDelegationData(ctx, sub.DelegationID)
    if err != nil {
        return nil, fmt.Errorf("failed to get delegation: %w", err)
    }
    
    // Call delegation server via gRPC
    request := &delegation.RedeemDelegationRequest{
        DelegationHash:   delegation.Hash,
        NetworkId:        sub.Product.NetworkID,
        TokenAddress:     sub.Product.TokenAddress,
        Amount:           sub.Price.Amount,
        RecipientAddress: sub.Workspace.PaymentAddress,
    }
    
    response, err := p.delegationClient.RedeemDelegation(ctx, request)
    if err != nil {
        return nil, fmt.Errorf("delegation redemption failed: %w", err)
    }
    
    if response.Status != "success" {
        return nil, fmt.Errorf("payment failed: %s", response.Message)
    }
    
    return &PaymentResult{
        TransactionHash: response.TransactionHash,
        Amount:          sub.Price.Amount,
        Currency:        sub.Price.Currency,
        NetworkID:       sub.Product.NetworkID,
        Status:          "completed",
    }, nil
}
```

### Retry Logic
```go
func (p *Processor) handlePaymentError(ctx context.Context, sub Subscription, err error) error {
    // Increment retry count
    retryCount := sub.RetryCount + 1
    
    // Check if we should retry
    if retryCount <= p.maxRetryAttempts {
        // Calculate next retry time with exponential backoff
        backoffDuration := time.Duration(math.Pow(float64(p.retryBackoffMultiplier), float64(retryCount-1))) * time.Minute
        nextRetryAt := time.Now().Add(backoffDuration)
        
        // Update subscription for retry
        err := p.db.UpdateSubscriptionForRetry(ctx, sub.ID, retryCount, nextRetryAt)
        if err != nil {
            return fmt.Errorf("failed to schedule retry: %w", err)
        }
        
        // Log retry event
        p.logSubscriptionEvent(ctx, nil, sub, "payment_retry_scheduled", map[string]interface{}{
            "retry_count": retryCount,
            "next_retry_at": nextRetryAt,
            "error": err.Error(),
        })
        
        return nil
    }
    
    // Max retries exceeded - move to failed state
    return p.handlePermanentFailure(ctx, sub, err)
}
```

### Permanent Failure Handling
```go
func (p *Processor) handlePermanentFailure(ctx context.Context, sub Subscription, err error) error {
    // Update subscription status to failed
    updateErr := p.db.UpdateSubscriptionStatus(ctx, sub.ID, "payment_failed")
    if updateErr != nil {
        return fmt.Errorf("failed to update subscription status: %w", updateErr)
    }
    
    // Log permanent failure
    p.logSubscriptionEvent(ctx, nil, sub, "payment_permanently_failed", map[string]interface{}{
        "final_error": err.Error(),
        "retry_count": sub.RetryCount,
    })
    
    // Send notification to customer and merchant
    p.notificationService.SendPaymentFailedNotification(ctx, sub)
    
    // Add to dead letter queue for manual review
    return p.addToDeadLetterQueue(ctx, sub, err)
}
```

## Database Integration

### Key Queries
The processor uses SQLC-generated queries from the shared database package:

#### Get Due Subscriptions
```sql
-- name: GetDueSubscriptions :many
SELECT s.*, p.amount, p.currency, p.interval_count, p.interval_type,
       pr.name as product_name, pr.network_id, pr.token_address
FROM subscriptions s
JOIN prices p ON s.price_id = p.id
JOIN products pr ON p.product_id = pr.id
WHERE s.status = 'active'
  AND s.next_billing_date <= $1
  AND (s.retry_count < $2 OR s.next_retry_at <= $1)
ORDER BY s.next_billing_date ASC;
```

#### Update Subscription After Payment
```sql
-- name: UpdateSubscriptionAfterPayment :exec
UPDATE subscriptions 
SET 
    last_billing_date = $2,
    next_billing_date = $3,
    retry_count = 0,
    next_retry_at = NULL,
    updated_at = NOW()
WHERE id = $1;
```

#### Log Subscription Event
```sql
-- name: CreateSubscriptionEvent :one
INSERT INTO subscription_events (
    id, subscription_id, event_type, event_data, created_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, NOW()
) RETURNING *;
```

### Database Transactions
All subscription processing uses database transactions to ensure consistency:

```go
func (p *Processor) processSubscriptionWithTransaction(ctx context.Context, sub Subscription) error {
    // Begin transaction
    tx, err := p.db.BeginTx(ctx, &sql.TxOptions{
        Isolation: sql.LevelReadCommitted,
    })
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Create queries with transaction context
    queries := db.New(tx)
    
    // Perform all database operations within transaction
    if err := p.updateSubscriptionInTx(ctx, queries, sub); err != nil {
        return err
    }
    
    if err := p.logEventInTx(ctx, queries, sub, eventData); err != nil {
        return err
    }
    
    // Commit transaction
    return tx.Commit()
}
```

## Error Handling

### Error Categories
The processor handles different types of errors with specific strategies:

#### Temporary Errors (Retry)
- Network timeouts
- Blockchain congestion
- Insufficient gas
- Temporary service unavailability

#### Permanent Errors (Fail)
- Invalid delegation signatures
- Expired delegations
- Insufficient token balance
- Invalid subscription configuration

#### System Errors (Alert)
- Database connection failures
- gRPC service unavailability
- Configuration errors

### Error Handling Implementation
```go
func (p *Processor) categorizeError(err error) ErrorCategory {
    // Check for specific error types
    if isNetworkError(err) || isTimeoutError(err) {
        return TemporaryError
    }
    
    if isInvalidDelegationError(err) || isInsufficientBalanceError(err) {
        return PermanentError
    }
    
    if isDatabaseError(err) || isServiceUnavailableError(err) {
        return SystemError
    }
    
    // Default to temporary for unknown errors
    return TemporaryError
}

func (p *Processor) handleError(ctx context.Context, sub Subscription, err error) error {
    category := p.categorizeError(err)
    
    switch category {
    case TemporaryError:
        return p.scheduleRetry(ctx, sub, err)
    case PermanentError:
        return p.markAsFailed(ctx, sub, err)
    case SystemError:
        p.alertService.SendSystemAlert(ctx, err)
        return p.scheduleRetry(ctx, sub, err) // Retry system errors
    default:
        return p.scheduleRetry(ctx, sub, err)
    }
}
```

## Monitoring & Logging

### Structured Logging
The processor uses structured logging for observability:

```go
func (p *Processor) processSubscription(ctx context.Context, sub Subscription) error {
    logger := p.logger.WithFields(logrus.Fields{
        "subscription_id": sub.ID,
        "customer_id":     sub.CustomerID,
        "workspace_id":    sub.WorkspaceID,
        "amount":          sub.Price.Amount,
        "currency":        sub.Price.Currency,
    })
    
    logger.Info("Starting subscription processing")
    
    startTime := time.Now()
    defer func() {
        duration := time.Since(startTime)
        logger.WithField("duration_ms", duration.Milliseconds()).
               Info("Subscription processing completed")
    }()
    
    // Processing logic...
}
```

### Metrics Collection
```go
type ProcessorMetrics struct {
    SubscriptionsProcessed prometheus.Counter
    PaymentSuccesses      prometheus.Counter
    PaymentFailures       prometheus.Counter
    ProcessingDuration    prometheus.Histogram
    RetryCount           prometheus.Counter
}

func (p *Processor) recordMetrics(result ProcessingResult) {
    p.metrics.SubscriptionsProcessed.Inc()
    
    if result.Success {
        p.metrics.PaymentSuccesses.Inc()
    } else {
        p.metrics.PaymentFailures.Inc()
    }
    
    p.metrics.ProcessingDuration.Observe(result.Duration.Seconds())
    
    if result.RetryScheduled {
        p.metrics.RetryCount.Inc()
    }
}
```

### Health Checks
```go
func (p *Processor) HealthCheck(ctx context.Context) error {
    // Check database connectivity
    if err := p.db.Ping(ctx); err != nil {
        return fmt.Errorf("database unhealthy: %w", err)
    }
    
    // Check delegation server connectivity
    if err := p.delegationClient.HealthCheck(ctx); err != nil {
        return fmt.Errorf("delegation server unhealthy: %w", err)
    }
    
    return nil
}
```

## Deployment

### AWS Lambda Deployment
The processor can be deployed as a scheduled AWS Lambda function:

```yaml
# serverless.yml
functions:
  subscriptionProcessor:
    handler: bootstrap
    runtime: provided.al2
    timeout: 300
    memorySize: 512
    events:
      - schedule: rate(5 minutes)
    environment:
      DATABASE_URL: ${ssm:/cyphera/database-url}
      DELEGATION_GRPC_ADDR: ${ssm:/cyphera/delegation-grpc-addr}
```

#### Build for Lambda
```bash
# Build for Lambda
GOOS=linux GOARCH=amd64 go build -o bootstrap cmd/main.go
zip lambda-deployment.zip bootstrap
```

### Container Deployment
For container orchestration platforms:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o subscription-processor cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/subscription-processor .
CMD ["./subscription-processor"]
```

### Kubernetes Deployment
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: subscription-processor
spec:
  schedule: "*/5 * * * *"  # Every 5 minutes
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: processor
            image: cyphera/subscription-processor:latest
            env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: cyphera-secrets
                  key: database-url
          restartPolicy: OnFailure
```

### Environment Configuration
Production environment variables should be managed via:

```bash
# AWS Systems Manager Parameter Store
aws ssm put-parameter \
  --name "/cyphera/subscription-processor/database-url" \
  --value "postgresql://..." \
  --type "SecureString"

# Kubernetes Secrets
kubectl create secret generic subscription-processor-secrets \
  --from-literal=database-url="postgresql://..."
```

## Performance Considerations

### Batch Processing
For high-volume processing, implement batching:

```go
func (p *Processor) ProcessInBatches(ctx context.Context, batchSize int) error {
    offset := 0
    
    for {
        subscriptions, err := p.db.GetDueSubscriptionsBatch(ctx, batchSize, offset)
        if err != nil {
            return err
        }
        
        if len(subscriptions) == 0 {
            break // No more subscriptions
        }
        
        // Process batch concurrently
        if err := p.processBatch(ctx, subscriptions); err != nil {
            return err
        }
        
        offset += batchSize
    }
    
    return nil
}
```

### Concurrent Processing
```go
func (p *Processor) processBatch(ctx context.Context, subscriptions []Subscription) error {
    semaphore := make(chan struct{}, p.maxConcurrency)
    var wg sync.WaitGroup
    
    for _, sub := range subscriptions {
        wg.Add(1)
        go func(subscription Subscription) {
            defer wg.Done()
            
            semaphore <- struct{}{} // Acquire
            defer func() { <-semaphore }() // Release
            
            if err := p.processSubscription(ctx, subscription); err != nil {
                p.logger.Error("Failed to process subscription", 
                    "subscription_id", subscription.ID,
                    "error", err)
            }
        }(sub)
    }
    
    wg.Wait()
    return nil
}
```

---

## Related Documentation

- **[Architecture Guide](../../docs/architecture.md)** - System overview
- **[Main API Documentation](../api/README.md)** - API service integration
- **[Delegation Server](../delegation-server/README.md)** - Blockchain operations
- **[Database Schema](../../libs/go/db/README.md)** - Database documentation

## Need Help?

- **[Troubleshooting](../../docs/troubleshooting.md)** - Common issues
- **[Contributing](../../docs/contributing.md)** - Development workflow
- **[Monitoring Guide](../../docs/monitoring.md)** - Observability setup
- **GitHub Issues** - Bug reports and feature requests

---

*Last updated: $(date '+%Y-%m-%d')*
*Service Version: 2.0.0*