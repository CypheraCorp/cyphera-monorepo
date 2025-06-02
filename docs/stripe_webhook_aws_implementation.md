# Multi-Provider Webhook AWS Infrastructure Implementation

**Version:** 3.0  
**Date:** December 2024  
**Status:** Implementation Ready - Multi-Provider Multi-Workspace Support

## Overview

This document outlines the implementation plan for processing **multi-provider payment webhooks** using AWS serverless architecture with **full multi-workspace support**. The system handles webhook events from multiple payment providers (Stripe, Chargebee, PayPal, etc.) across multiple accounts asynchronously, ensuring reliable processing, proper workspace isolation, and database synchronization across different customer workspaces.

## Multi-Provider Multi-Workspace Architecture

```
┌─────────────┐    ┌──────────────┐    ┌─────────────────┐    ┌─────┐    ┌───────────────────┐    ┌──────────┐
│ Stripe      │───▶│ API Gateway  │───▶│ Lambda          │───▶│ SQS │───▶│ Lambda            │───▶│ Database │
│ Account A   │    │ /webhooks/   │    │ Receiver        │    │     │    │ Processor         │    │          │
│ (Workspace) │    │ providers    │    │ - Route by      │    │     │    │ - Workspace       │    │          │
├─────────────┤    │              │    │   Provider &    │    │     │    │   Context         │    │          │
│ Chargebee   │───▶│              │───▶│   Account ID    │───▶│     │───▶│ - Dynamic         │───▶│          │
│ Site B      │    │              │    │ - Validate      │    │     │    │   Config Load     │    │          │
│ (Workspace) │    │              │    │   Signature     │    │     │    │ - Event           │    │          │
├─────────────┤    │              │    │ - Workspace     │    │     │    │   Processing      │    │          │
│ PayPal      │───▶│              │───▶│   Lookup        │───▶│     │───▶│ - Idempotency     │───▶│          │
│ Account C   │    │              │    │ - Queue Event   │    │     │    │                   │    │          │
│ (Workspace) │    │              │    │                 │    │     │    │                   │    │          │
└─────────────┘    └──────────────┘    └─────────────────┘    └─────┘    └───────────────────┘    └──────────┘
                                               │                                      │
                                               ▼                                      ▼
                                         ┌──────────┐                          ┌──────────┐
                                         │CloudWatch│                          │   DLQ    │
                                         │   Logs   │                          │          │
                                         └──────────┘                          └──────────┘
```

## Multi-Provider Webhook Strategy

### 1. Provider-Agnostic Workspace Identification

Based on webhook documentation from major providers, we can identify workspaces using:

1. **Provider Account ID** (Primary): 
   - **Stripe**: `Stripe-Account` header (`acct_xxx`)
   - **Chargebee**: Site ID in webhook URL or event data 
   - **PayPal**: Webhook ID or merchant ID
   - **Square**: Application ID or location ID

2. **Generic Mapping Table**: Map any provider account ID to workspace IDs
3. **Webhook Endpoint Parameters**: Optional query parameters for workspace routing  
4. **Custom Metadata**: Use provider-specific metadata fields

### 2. Generic Webhook Configuration Patterns

#### Pattern A: Single Endpoint + Provider/Account Routing (Recommended)
```
API Endpoint: https://api.cyphera.com/webhooks/providers
- All providers and workspaces use the same endpoint
- Provider identified by User-Agent or webhook structure
- Account identified by provider-specific headers/data
- Dynamic webhook secret lookup per workspace + provider
```

#### Pattern B: Provider-Specific Endpoints (Alternative)
```
API Endpoints: 
- https://api.cyphera.com/webhooks/stripe
- https://api.cyphera.com/webhooks/chargebee  
- https://api.cyphera.com/webhooks/paypal
- Workspace ID from provider account mapping
- Provider-specific webhook secret per workspace
```

## Enhanced Architecture Components

### 1.1 Generic Database Schema (Already Implemented)

#### Provider-Agnostic Account Mapping
```sql
-- Generic table for any payment provider account mapping
CREATE TABLE IF NOT EXISTS workspace_provider_accounts (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id),
    provider_name VARCHAR(50) NOT NULL, -- 'stripe', 'chargebee', 'paypal', etc.
    provider_account_id VARCHAR(255) NOT NULL, -- Provider Account ID (acct_xxx, site_xxx, etc.)
    account_type VARCHAR(50) NOT NULL, -- 'standard', 'express', 'custom', 'platform', 'site', etc.
    is_active BOOLEAN NOT NULL DEFAULT true,
    environment VARCHAR(20) NOT NULL DEFAULT 'live', -- 'live' or 'test'
    display_name VARCHAR(255), -- Human readable name
    metadata JSONB DEFAULT '{}', -- Provider-specific metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(provider_name, provider_account_id, environment)
);
```

#### Enhanced Generic Webhook Event Tracking
```sql
-- Generic webhook event tracking for all providers
CREATE TABLE IF NOT EXISTS payment_sync_events (
    -- ... existing fields ...
    webhook_event_id VARCHAR(255), -- Provider event ID (evt_xxx, webhook_xxx, etc.)
    provider_account_id VARCHAR(255), -- Provider Account ID for workspace routing
    idempotency_key VARCHAR(255), -- workspace_id + provider + event_id
    processing_attempts INTEGER DEFAULT 0,
    signature_valid BOOLEAN,
    -- ... other fields ...
);
```

### 1.2 Infrastructure Components

#### Enhanced SQS Configuration
```yaml
Main Queue: "{env}-provider-webhook-events"
DLQ: "{env}-provider-webhook-events-dlq"
Visibility Timeout: 300 seconds (5 minutes)
Message Retention: 14 days
Max Retry Attempts: 3
Batch Size: 10 messages per Lambda invoke
```

#### Enhanced IAM Roles
```yaml
Receiver Lambda Role:
  - API Gateway invoke permissions
  - SQS send message permissions
  - Secrets Manager read (all workspace configs)
  - CloudWatch logs write
  - Database VPC access (for workspace lookups)

Processor Lambda Role:
  - SQS receive/delete message permissions
  - Database VPC access
  - Secrets Manager read (workspace-specific configs)
  - CloudWatch logs write
```

## Phase 1: Multi-Provider Infrastructure

### Terraform Configuration

#### 1.1 Enhanced SQS Infrastructure (`terraform/provider_webhooks_sqs.tf`)
```hcl
# Enhanced SQS with multi-provider support
resource "aws_sqs_queue" "provider_webhook_events" {
  name = "${var.service_prefix}-provider-webhook-events-${var.stage}"
  
  # Enhanced configuration for multi-provider multi-workspace
  visibility_timeout_seconds = 300
  message_retention_seconds = 1209600 # 14 days
  receive_wait_time_seconds = 20 # Long polling
  
  # Enhanced redrive policy
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.provider_webhook_events_dlq.arn
    maxReceiveCount     = 3
  })

  tags = merge(local.common_tags, {
    Name        = "${var.service_prefix}-provider-webhook-events-${var.stage}"
    Purpose     = "Multi-provider multi-workspace webhook event processing"
    Component   = "provider-webhooks"
  })
}
```

#### 1.2 Enhanced Lambda Configuration
```hcl
# Enhanced environment variables for multi-provider multi-workspace support
resource "aws_lambda_function" "provider_webhook_receiver" {
  function_name = "${var.service_prefix}-provider-webhook-receiver-${var.stage}"
  
  environment {
    variables = {
      SQS_QUEUE_URL = aws_sqs_queue.provider_webhook_events.url
      LOG_LEVEL = "INFO"
      STAGE = var.stage
      # Database connection for workspace lookups
      DB_HOST = var.db_host
      DB_NAME = var.db_name
      RDS_SECRET_ARN = var.rds_secret_arn
      # Payment sync encryption key for config decryption
      PAYMENT_SYNC_ENCRYPTION_KEY_ARN = aws_secretsmanager_secret.payment_sync_encryption_key.arn
    }
  }
}
```

## Phase 2: Multi-Provider Lambda Functions

### 2.1 Enhanced Generic Webhook Receiver Lambda

**Location**: `cmd/provider-webhook-receiver/main.go`

**Enhanced Responsibilities:**
1. **Accept webhook from any payment provider**
2. **Identify provider type (Stripe, Chargebee, PayPal, etc.)**
3. **Extract provider account ID from headers/payload**
4. **Lookup workspace ID from provider account mapping**
5. **Dynamically fetch workspace-specific webhook secret**
6. **Validate webhook signature with provider-specific method**
7. **Enrich event with workspace and provider context**
8. **Queue to SQS with workspace metadata**

**Key Implementation Details:**

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    
    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "go.uber.org/zap"
)

type WebhookReceiver struct {
    logger            *zap.Logger
    sqsClient         *sqs.Client
    secretsClient     *secretsmanager.Client
    dbQueries         *db.Queries
    queueURL          string
    encryptionKey     string
}

type EnrichedWebhookEvent struct {
    ProviderEventID   string                 `json:"provider_event_id"`
    ProviderName      string                 `json:"provider_name"`
    ProviderAccountID string                 `json:"provider_account_id"`
    WorkspaceID       string                 `json:"workspace_id"`
    EventType         string                 `json:"event_type"`
    RawEventData      json.RawMessage        `json:"raw_event_data"`
    ReceivedAt        int64                  `json:"received_at"`
    IdempotencyKey    string                 `json:"idempotency_key"`
    Headers           map[string]string      `json:"headers"`
}

func (w *WebhookReceiver) HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    // 1. Identify provider and extract account ID
    providerInfo, err := w.identifyProvider(ctx, request)
    if err != nil {
        w.logger.Error("Failed to identify provider", zap.Error(err))
        return events.APIGatewayProxyResponse{
            StatusCode: 400,
            Body:       "Unable to identify payment provider",
        }, nil
    }
    
    // 2. Lookup workspace by provider account ID
    workspaceID, err := w.lookupWorkspaceByProviderAccount(ctx, 
        providerInfo.ProviderName, providerInfo.AccountID)
    if err != nil {
        w.logger.Error("Failed to lookup workspace", 
            zap.String("provider", providerInfo.ProviderName),
            zap.String("account_id", providerInfo.AccountID),
            zap.Error(err))
        return events.APIGatewayProxyResponse{
            StatusCode: 400,
            Body:       "Invalid provider account",
        }, nil
    }
    
    // 3. Get workspace-specific webhook secret
    webhookSecret, err := w.getWorkspaceWebhookSecret(ctx, workspaceID, providerInfo.ProviderName)
    if err != nil {
        w.logger.Error("Failed to get webhook secret",
            zap.String("workspace_id", workspaceID),
            zap.String("provider", providerInfo.ProviderName),
            zap.Error(err))
        return events.APIGatewayProxyResponse{
            StatusCode: 500,
            Body:       "Configuration error",
        }, nil
    }
    
    // 4. Validate webhook signature (provider-specific)
    if err := w.validateWebhookSignature(request, providerInfo.ProviderName, webhookSecret); err != nil {
        w.logger.Error("Webhook signature validation failed",
            zap.String("workspace_id", workspaceID),
            zap.String("provider", providerInfo.ProviderName),
            zap.Error(err))
        return events.APIGatewayProxyResponse{
            StatusCode: 400,
            Body:       "Invalid signature",
        }, nil
    }
    
    // 5. Parse provider event
    eventInfo, err := w.parseProviderEvent(request, providerInfo.ProviderName)
    if err != nil {
        w.logger.Error("Failed to parse provider event", 
            zap.String("provider", providerInfo.ProviderName),
            zap.Error(err))
        return events.APIGatewayProxyResponse{
            StatusCode: 400,
            Body:       "Invalid event format",
        }, nil
    }
    
    // 6. Create enriched event for processing
    enrichedEvent := EnrichedWebhookEvent{
        ProviderEventID:   eventInfo.EventID,
        ProviderName:      providerInfo.ProviderName,
        ProviderAccountID: providerInfo.AccountID,
        WorkspaceID:       workspaceID,
        EventType:         eventInfo.EventType,
        RawEventData:      json.RawMessage(request.Body),
        ReceivedAt:        time.Now().Unix(),
        IdempotencyKey:    fmt.Sprintf("%s-%s-%s", workspaceID, providerInfo.ProviderName, eventInfo.EventID),
        Headers:           request.Headers,
    }
    
    // 7. Queue to SQS
    if err := w.queueWebhookEvent(ctx, enrichedEvent); err != nil {
        w.logger.Error("Failed to queue webhook event", zap.Error(err))
        return events.APIGatewayProxyResponse{
            StatusCode: 500,
            Body:       "Processing error",
        }, nil
    }
    
    w.logger.Info("Successfully queued webhook event",
        zap.String("provider_event_id", eventInfo.EventID),
        zap.String("provider", providerInfo.ProviderName),
        zap.String("workspace_id", workspaceID),
        zap.String("event_type", eventInfo.EventType))
    
    return events.APIGatewayProxyResponse{
        StatusCode: 200,
        Body:       "OK",
    }, nil
}

type ProviderInfo struct {
    ProviderName string
    AccountID    string
}

type EventInfo struct {
    EventID   string
    EventType string
}

func (w *WebhookReceiver) identifyProvider(ctx context.Context, request events.APIGatewayProxyRequest) (*ProviderInfo, error) {
    // Check User-Agent for provider identification
    userAgent := request.Headers["User-Agent"]
    
    // Stripe identification
    if strings.Contains(userAgent, "Stripe") || request.Headers["Stripe-Account"] != "" {
        accountID := request.Headers["Stripe-Account"]
        if accountID == "" {
            accountID = request.Headers["stripe-account"] // Case insensitive
        }
        return &ProviderInfo{
            ProviderName: "stripe",
            AccountID:    accountID,
        }, nil
    }
    
    // Chargebee identification
    if strings.Contains(userAgent, "ChargeBee") {
        // Parse Chargebee site from webhook URL or event data
        var event map[string]interface{}
        if err := json.Unmarshal([]byte(request.Body), &event); err == nil {
            if site, ok := event["site"].(string); ok {
                return &ProviderInfo{
                    ProviderName: "chargebee",
                    AccountID:    site,
                }, nil
            }
        }
    }
    
    // PayPal identification
    if strings.Contains(userAgent, "PayPal") {
        // PayPal webhook verification header contains merchant ID
        if authHeader := request.Headers["PAYPAL-AUTH-ALGO"]; authHeader != "" {
            // Extract merchant ID from PayPal headers or event data
            // Implementation specific to PayPal webhook format
        }
    }
    
    return nil, fmt.Errorf("unable to identify payment provider from webhook")
}

func (w *WebhookReceiver) lookupWorkspaceByProviderAccount(ctx context.Context, providerName, accountID string) (string, error) {
    // Query workspace_provider_accounts table using our new generic queries
    mapping, err := w.dbQueries.GetWorkspaceByProviderAccount(ctx, db.GetWorkspaceByProviderAccountParams{
        ProviderName:      providerName,
        ProviderAccountID: accountID,
        Environment:       "live", // or determine from context
    })
    if err != nil {
        return "", fmt.Errorf("workspace not found for %s account %s: %w", providerName, accountID, err)
    }
    return mapping.WorkspaceID.String(), nil
}

func (w *WebhookReceiver) getWorkspaceWebhookSecret(ctx context.Context, workspaceID, providerName string) (string, error) {
    // Get workspace payment configuration using our existing query
    config, err := w.dbQueries.GetPaymentConfigurationByProvider(ctx, db.GetPaymentConfigurationByProviderParams{
        WorkspaceID:  uuid.MustParse(workspaceID),
        ProviderName: providerName,
    })
    if err != nil {
        return "", fmt.Errorf("configuration not found: %w", err)
    }
    
    // Decrypt webhook secret from configuration
    var providerConfig map[string]interface{}
    if err := w.decryptConfiguration(config.EncryptedConfiguration, &providerConfig); err != nil {
        return "", fmt.Errorf("failed to decrypt configuration: %w", err)
    }
    
    webhookSecret, ok := providerConfig["webhook_secret"].(string)
    if !ok {
        return "", fmt.Errorf("webhook secret not found in configuration")
    }
    
    return webhookSecret, nil
}

func (w *WebhookReceiver) validateWebhookSignature(request events.APIGatewayProxyRequest, providerName, secret string) error {
    switch providerName {
    case "stripe":
        return w.validateStripeSignature(request, secret)
    case "chargebee":
        return w.validateChargebeeSignature(request, secret)
    case "paypal":
        return w.validatePayPalSignature(request, secret)
    default:
        return fmt.Errorf("signature validation not implemented for provider: %s", providerName)
    }
}

func (w *WebhookReceiver) parseProviderEvent(request events.APIGatewayProxyRequest, providerName string) (*EventInfo, error) {
    var event map[string]interface{}
    if err := json.Unmarshal([]byte(request.Body), &event); err != nil {
        return nil, err
    }
    
    switch providerName {
    case "stripe":
        return &EventInfo{
            EventID:   event["id"].(string),
            EventType: event["type"].(string),
        }, nil
    case "chargebee":
        return &EventInfo{
            EventID:   event["id"].(string),
            EventType: event["event_type"].(string),
        }, nil
    case "paypal":
        return &EventInfo{
            EventID:   event["id"].(string),
            EventType: event["event_type"].(string),
        }, nil
    default:
        return nil, fmt.Errorf("event parsing not implemented for provider: %s", providerName)
    }
}

## Phase 3: Multi-Workspace Database Queries

### Enhanced SQLC Queries

#### Workspace-Stripe Account Mapping
```sql
-- name: CreateWorkspaceStripeAccount :one
INSERT INTO workspace_stripe_accounts (
    workspace_id, stripe_account_id, account_type, environment
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetWorkspaceByStripeAccount :one
SELECT workspace_id, account_type FROM workspace_stripe_accounts
WHERE stripe_account_id = $1 AND is_active = true;

-- name: ListStripeAccountsByWorkspace :many
SELECT * FROM workspace_stripe_accounts
WHERE workspace_id = $1 AND is_active = true;
```

#### Enhanced Idempotency Tracking
```sql
-- name: CheckWebhookIdempotency :one
SELECT COUNT(*) FROM payment_sync_events
WHERE idempotency_key = $1;

-- name: CreateWebhookEvent :one
INSERT INTO payment_sync_events (
    workspace_id, provider_name, webhook_event_id, 
    idempotency_key, event_type, stripe_account_id
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;
```

## Phase 4: Enhanced Monitoring & Operations

### Multi-Workspace Monitoring

#### CloudWatch Metrics
```yaml
Custom Metrics:
  - WebhooksByWorkspace: Count by workspace_id
  - ProcessingLatencyByWorkspace: Latency per workspace
  - ErrorRateByWorkspace: Error rate per workspace
  - ActiveWorkspaces: Number of active workspaces
  - StripeAccountsPerWorkspace: Distribution metrics
```

#### Enhanced Alarms
```yaml
Workspace-Specific Alarms:
  - High error rate for specific workspace
  - Webhook processing delays per workspace
  - Configuration issues per workspace
  - Unusual webhook volume per workspace
```

### Operational Dashboards

#### Multi-Workspace Dashboard
- **Workspace Overview**: Active workspaces and health status
- **Webhook Volume**: Per-workspace webhook processing rates
- **Error Analysis**: Workspace-specific error breakdown
- **Performance Metrics**: Latency and throughput by workspace

## Multi-Workspace Configuration Management

### Workspace Onboarding Process

1. **Create Workspace Payment Configuration**
   ```bash
   POST /api/v1/sync/config
   {
     "provider_name": "stripe",
     "configuration": {
       "api_key": "sk_test_...",
       "webhook_secret": "whsec_..."
     }
   }
   ```

2. **Register Stripe Account Mapping**
   ```sql
   INSERT INTO workspace_stripe_accounts 
   (workspace_id, stripe_account_id, account_type)
   VALUES ('workspace-uuid', 'acct_stripe123', 'standard');
   ```

3. **Configure Stripe Webhook Endpoint**
   ```bash
   # Stripe CLI or Dashboard
   Webhook URL: https://api.cyphera.com/webhooks/stripe
   Events: customer.*, product.*, price.*, subscription.*, invoice.*
   ```

### Workspace Migration Strategy

#### Existing Single-Workspace to Multi-Workspace
1. **Create workspace_stripe_accounts entries** for existing configurations
2. **Update webhook URLs** to use new multi-workspace endpoint
3. **Test webhook routing** with existing workspaces
4. **Gradually migrate** additional workspaces

## Security Considerations

### Multi-Workspace Security

1. **Workspace Isolation**: Complete data isolation between workspaces
2. **Configuration Encryption**: All webhook secrets encrypted per workspace
3. **Access Control**: Workspace-scoped access to webhook data
4. **Audit Logging**: Comprehensive audit trail per workspace
5. **Rate Limiting**: Per-workspace rate limiting to prevent abuse

### Enhanced Security Measures

```yaml
Security Features:
  - Workspace-specific webhook secret rotation
  - Cross-workspace event validation
  - Encrypted configuration storage
  - Audit logging for all webhook events
  - Rate limiting per Stripe account
```

## Testing Strategy

### Multi-Workspace Testing

#### Unit Tests
- Workspace lookup functionality
- Dynamic configuration loading
- Event processing per workspace
- Idempotency handling per workspace

#### Integration Tests
- Multiple workspace webhook processing
- Concurrent workspace events
- Workspace configuration changes
- Error handling per workspace

#### Load Testing
- High-volume multi-workspace scenarios
- Concurrent webhook processing
- Database performance under multi-workspace load
- SQS handling of workspace-distributed events

## Success Criteria

✅ **Multi-Workspace Support**: Handle unlimited workspaces  
✅ **Workspace Isolation**: Complete data and processing isolation  
✅ **Dynamic Configuration**: Runtime configuration loading per workspace  
✅ **Reliability**: 99.9% webhook processing success rate per workspace  
✅ **Performance**: < 2 second average processing time regardless of workspace count  
✅ **Scalability**: Handle 1000+ webhooks per minute across all workspaces  
✅ **Security**: Zero cross-workspace data leakage  
✅ **Monitoring**: Complete observability per workspace  

## Implementation Roadmap

### Phase 1: Foundation (Week 1)
- [ ] Enhanced database schema with workspace mapping
- [ ] Updated Terraform infrastructure
- [ ] Basic multi-workspace webhook receiver

### Phase 2: Core Processing (Week 1-2)
- [ ] Enhanced event processor with workspace context
- [ ] Dynamic configuration loading
- [ ] Workspace-scoped idempotency

### Phase 3: Production Hardening (Week 2)
- [ ] Comprehensive testing suite
- [ ] Multi-workspace monitoring
- [ ] Performance optimization

### Phase 4: Operations (Week 3)
- [ ] Operational runbooks
- [ ] Workspace onboarding automation
- [ ] Cost optimization

---

**Next Steps:** Begin Phase 1 implementation with enhanced Terraform infrastructure and database schema updates.

---

**Dependencies:**
- Existing payment sync interface (`