-- name: CreateFailedSubscriptionAttempt :one
INSERT INTO failed_subscription_attempts (
    customer_id,
    product_id,
    product_token_id,
    customer_wallet_id,
    wallet_address,
    error_type,
    error_message,
    error_details,
    delegation_signature,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetFailedSubscriptionAttempt :one
SELECT * FROM failed_subscription_attempts
WHERE id = $1;

-- name: ListFailedSubscriptionAttempts :many
SELECT * FROM failed_subscription_attempts
ORDER BY occurred_at DESC;

-- name: ListFailedSubscriptionAttemptsByCustomer :many
SELECT * FROM failed_subscription_attempts
WHERE customer_id = $1
ORDER BY occurred_at DESC;

-- name: ListFailedSubscriptionAttemptsByProduct :many
SELECT * FROM failed_subscription_attempts
WHERE product_id = $1
ORDER BY occurred_at DESC;

-- name: ListFailedSubscriptionAttemptsByWalletAddress :many
SELECT * FROM failed_subscription_attempts
WHERE wallet_address = $1
ORDER BY occurred_at DESC;

-- name: ListFailedSubscriptionAttemptsByErrorType :many
SELECT * FROM failed_subscription_attempts
WHERE error_type = $1
ORDER BY occurred_at DESC;

-- name: ListRecentFailedSubscriptionAttempts :many
SELECT * FROM failed_subscription_attempts
WHERE occurred_at >= $1
ORDER BY occurred_at DESC;

-- name: CountFailedSubscriptionAttempts :one
SELECT COUNT(*) FROM failed_subscription_attempts;

-- name: CountFailedSubscriptionAttemptsByErrorType :one
SELECT COUNT(*) FROM failed_subscription_attempts
WHERE error_type = $1;

-- name: DeleteFailedSubscriptionAttempt :exec
DELETE FROM failed_subscription_attempts
WHERE id = $1;

-- name: ListFailedSubscriptionAttemptsWithPagination :many
SELECT * FROM failed_subscription_attempts
ORDER BY occurred_at DESC
LIMIT $1 OFFSET $2; 