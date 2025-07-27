-- name: CreateDunningAttempt :one
INSERT INTO dunning_attempts (
    campaign_id,
    attempt_number,
    attempt_type,
    status,
    payment_id,
    communication_type,
    email_template_id,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetDunningAttempt :one
SELECT * FROM dunning_attempts
WHERE id = $1;

-- name: ListDunningAttempts :many
SELECT * FROM dunning_attempts
WHERE campaign_id = $1
ORDER BY attempt_number DESC;

-- name: UpdateDunningAttempt :one
UPDATE dunning_attempts
SET 
    status = COALESCE($2, status),
    completed_at = $3,
    payment_status = COALESCE($4, payment_status),
    payment_error = COALESCE($5, payment_error),
    communication_sent = COALESCE($6, communication_sent),
    communication_error = COALESCE($7, communication_error),
    customer_response = COALESCE($8, customer_response),
    customer_response_at = $9,
    metadata = COALESCE($10, metadata)
WHERE id = $1
RETURNING *;

-- name: GetLatestAttemptForCampaign :one
SELECT * FROM dunning_attempts
WHERE campaign_id = $1
ORDER BY attempt_number DESC
LIMIT 1;

-- name: CountSuccessfulAttempts :one
SELECT COUNT(*) as successful_attempts
FROM dunning_attempts
WHERE campaign_id = $1
    AND status = 'success';

-- name: GetAttemptsByType :many
SELECT 
    attempt_type,
    COUNT(*) as total_attempts,
    COUNT(*) FILTER (WHERE status = 'success') as successful_attempts,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_attempts
FROM dunning_attempts
WHERE campaign_id = $1
GROUP BY attempt_type;