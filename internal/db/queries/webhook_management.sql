-- webhook_management.sql
-- Specialized queries for multi-provider webhook processing in AWS Lambda

-- name: GetWorkspaceConfigForWebhook :one
-- Combined query to get workspace and configuration for webhook processing
SELECT 
    w.id as workspace_id,
    w.name as workspace_name,
    w.livemode as workspace_livemode,
    wpc.id as config_id,
    wpc.provider_name,
    wpc.is_active as config_active,
    wpc.is_test_mode,
    wpc.configuration,
    wpc.webhook_secret_key,
    wpc.connected_account_id,
    wpa.provider_account_id,
    wpa.account_type,
    wpa.environment
FROM workspaces w
JOIN workspace_provider_accounts wpa ON w.id = wpa.workspace_id
JOIN workspace_payment_configurations wpc ON w.id = wpc.workspace_id AND wpa.provider_name = wpc.provider_name
WHERE wpa.provider_name = $1 
  AND wpa.provider_account_id = $2 
  AND wpa.environment = $3
  AND wpa.is_active = true
  AND wpc.is_active = true
  AND w.deleted_at IS NULL
  AND wpa.deleted_at IS NULL
  AND wpc.deleted_at IS NULL;

-- name: LogWebhookReceived :one
-- Log incoming webhook before processing
INSERT INTO payment_sync_events (
    workspace_id,
    provider_name,
    entity_type,
    event_type,
    event_message,
    event_details,
    webhook_event_id,
    provider_account_id,
    idempotency_key,
    signature_valid
) VALUES (
    $1, $2, 'webhook', 'webhook_received', $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: UpdateWebhookProcessingStatus :one
-- Update webhook processing status (success/failure)
UPDATE payment_sync_events
SET 
    event_type = $2,
    event_message = $3,
    event_details = COALESCE(event_details, '{}'::jsonb) || $4,
    processing_attempts = processing_attempts + 1
WHERE id = $1
RETURNING *;

-- name: GetRecentWebhookErrors :many
-- Get recent webhook processing errors for monitoring
SELECT 
    webhook_event_id,
    provider_name,
    provider_account_id,
    event_message,
    event_details,
    processing_attempts,
    occurred_at
FROM payment_sync_events
WHERE workspace_id = $1
  AND provider_name = $2
  AND event_type = 'webhook_processing_failed'
  AND occurred_at >= NOW() - INTERVAL '24 hours'
ORDER BY occurred_at DESC
LIMIT $3;

-- name: GetWebhookProcessingStats :one
-- Get webhook processing statistics for monitoring
SELECT 
    COUNT(*) as total_webhooks,
    COUNT(CASE WHEN event_type = 'webhook_processed_successfully' THEN 1 END) as successful_webhooks,
    COUNT(CASE WHEN event_type = 'webhook_processing_failed' THEN 1 END) as failed_webhooks,
    COUNT(CASE WHEN signature_valid = false THEN 1 END) as invalid_signatures,
    AVG(processing_attempts) as avg_processing_attempts,
    MAX(occurred_at) as last_webhook_time
FROM payment_sync_events
WHERE workspace_id = $1
  AND provider_name = $2
  AND webhook_event_id IS NOT NULL
  AND occurred_at >= $3;

-- name: GetDuplicateWebhookEvents :many
-- Find duplicate webhook events for debugging
SELECT 
    webhook_event_id,
    COUNT(*) as duplicate_count,
    array_agg(id ORDER BY occurred_at) as event_ids,
    MIN(occurred_at) as first_received,
    MAX(occurred_at) as last_received
FROM payment_sync_events
WHERE workspace_id = $1
  AND provider_name = $2
  AND webhook_event_id IS NOT NULL
  AND occurred_at >= NOW() - INTERVAL '7 days'
GROUP BY webhook_event_id
HAVING COUNT(*) > 1
ORDER BY duplicate_count DESC
LIMIT $3;

-- name: GetWebhookEventsByTimeRange :many
-- Get webhook events within a specific time range for analysis
SELECT 
    webhook_event_id,
    entity_type,
    event_type,
    signature_valid,
    processing_attempts,
    occurred_at,
    event_details->'event_data'->'type' as webhook_type
FROM payment_sync_events
WHERE workspace_id = $1
  AND provider_name = $2
  AND webhook_event_id IS NOT NULL
  AND occurred_at BETWEEN $3 AND $4
ORDER BY occurred_at DESC
LIMIT $5 OFFSET $6;

-- name: UpdateLastWebhookTime :exec
-- Update the last webhook received time for a workspace configuration
UPDATE workspace_payment_configurations
SET 
    last_webhook_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE workspace_id = $1 
  AND provider_name = $2
  AND deleted_at IS NULL;

-- name: GetWebhookHealthStatus :one
-- Get overall webhook health for a workspace and provider
SELECT 
    wpc.provider_name,
    wpc.last_webhook_at,
    wpc.is_active as config_active,
    COUNT(pse.id) as recent_events,
    COUNT(CASE WHEN pse.event_type = 'webhook_processing_failed' THEN 1 END) as recent_failures,
    MAX(pse.occurred_at) as last_event_time
FROM workspace_payment_configurations wpc
LEFT JOIN payment_sync_events pse ON wpc.workspace_id = pse.workspace_id 
    AND wpc.provider_name = pse.provider_name
    AND pse.webhook_event_id IS NOT NULL
    AND pse.occurred_at >= NOW() - INTERVAL '1 hour'
WHERE wpc.workspace_id = $1
  AND wpc.provider_name = $2
  AND wpc.deleted_at IS NULL
GROUP BY wpc.provider_name, wpc.last_webhook_at, wpc.is_active;

-- name: GetFailedWebhooksForRetry :many
-- Get failed webhook events that are eligible for retry
SELECT 
    id,
    webhook_event_id,
    provider_account_id,
    event_details,
    processing_attempts,
    occurred_at
FROM payment_sync_events
WHERE workspace_id = $1
  AND provider_name = $2
  AND event_type = 'webhook_processing_failed'
  AND processing_attempts < $3
  AND occurred_at >= NOW() - INTERVAL '24 hours'
ORDER BY occurred_at ASC
LIMIT $4;

-- name: MarkWebhookForRetry :one
-- Mark a webhook event for retry processing
UPDATE payment_sync_events
SET 
    event_type = 'webhook_retry_queued',
    event_details = COALESCE(event_details, '{}'::jsonb) || jsonb_build_object(
        'retry_queued_at', EXTRACT(epoch FROM CURRENT_TIMESTAMP),
        'retry_attempt', processing_attempts + 1
    )
WHERE id = $1
RETURNING *;

-- name: GetWebhookEventForReplay :one
-- Get full webhook event details for replay functionality
SELECT 
    id,
    workspace_id,
    provider_name,
    webhook_event_id,
    provider_account_id,
    event_details,
    processing_attempts,
    signature_valid,
    occurred_at
FROM payment_sync_events
WHERE workspace_id = $1
  AND provider_name = $2
  AND webhook_event_id = $3
  AND webhook_event_id IS NOT NULL
ORDER BY occurred_at DESC
LIMIT 1;

-- name: ReplayWebhookEvent :one
-- Create a new event record for webhook replay
INSERT INTO payment_sync_events (
    workspace_id,
    provider_name,
    entity_type,
    event_type,
    event_message,
    event_details,
    webhook_event_id,
    provider_account_id,
    idempotency_key,
    signature_valid,
    processing_attempts
) VALUES (
    $1, $2, 'webhook', 'webhook_replayed', $3, $4, $5, $6, $7, $8, 0
) RETURNING *;

-- name: GetFailedSyncSessionsForRecovery :many
-- Get failed or incomplete sync sessions that can be resumed
SELECT 
    id,
    workspace_id,
    provider_name,
    session_type,
    status,
    entity_types,
    config,
    progress,
    error_summary,
    started_at,
    created_at
FROM payment_sync_sessions
WHERE workspace_id = $1
  AND (status = 'failed' OR status = 'running')
  AND created_at >= NOW() - INTERVAL '7 days'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ResumeSyncSession :one
-- Resume a failed sync session by updating its status
UPDATE payment_sync_sessions
SET 
    status = 'running',
    progress = COALESCE(progress, '{}'::jsonb) || jsonb_build_object(
        'resumed_at', EXTRACT(epoch FROM CURRENT_TIMESTAMP),
        'resume_count', COALESCE((progress->>'resume_count')::int, 0) + 1
    ),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
  AND workspace_id = $2
  AND status IN ('failed', 'running')
RETURNING *;

-- name: GetSyncProgressByEntityType :many
-- Get detailed progress for a sync session by entity type
SELECT 
    entity_type,
    COUNT(*) as total_events,
    COUNT(CASE WHEN event_type LIKE '%success%' OR event_type LIKE '%completed%' THEN 1 END) as successful_events,
    COUNT(CASE WHEN event_type LIKE '%failed%' OR event_type LIKE '%error%' THEN 1 END) as failed_events,
    MAX(occurred_at) as last_processed_at
FROM payment_sync_events
WHERE session_id = $1
  AND entity_type != 'webhook'
GROUP BY entity_type
ORDER BY entity_type;

-- name: GetDLQProcessingStats :one
-- Get statistics about DLQ processing for monitoring
SELECT 
    COUNT(*) as total_dlq_messages,
    COUNT(CASE WHEN event_type = 'dlq_processing_success' THEN 1 END) as successfully_processed,
    COUNT(CASE WHEN event_type = 'dlq_processing_failed' THEN 1 END) as processing_failed,
    COUNT(CASE WHEN processing_attempts >= 5 THEN 1 END) as max_retries_exceeded,
    MAX(occurred_at) as last_processed_at
FROM payment_sync_events
WHERE event_type LIKE 'dlq_%'
  AND workspace_id = $1
  AND provider_name = $2
  AND occurred_at >= $3;

-- name: LogDLQProcessingAttempt :one
-- Log DLQ processing attempt
INSERT INTO payment_sync_events (
    workspace_id,
    provider_name,
    entity_type,
    event_type,
    event_message,
    event_details,
    webhook_event_id,
    provider_account_id,
    processing_attempts
) VALUES (
    $1, $2, 'dlq_processing', $3, $4, $5, $6, $7, $8
) RETURNING *; 