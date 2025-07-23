-- name: CreateCustomerWallet :one
INSERT INTO customer_wallets (
    customer_id,
    wallet_address,
    network_type,
    nickname,
    ens,
    is_primary,
    verified,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetCustomerWallet :one
SELECT * FROM customer_wallets
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetCustomerWalletByAddress :one
SELECT * FROM customer_wallets
WHERE customer_id = $1 AND wallet_address = $2 AND network_type = $3 AND deleted_at IS NULL;

-- name: ListCustomerWallets :many
SELECT * FROM customer_wallets
WHERE customer_id = $1 AND deleted_at IS NULL
ORDER BY is_primary DESC, created_at DESC;

-- name: ListPrimaryCustomerWallets :many
SELECT * FROM customer_wallets
WHERE is_primary = true AND deleted_at IS NULL
ORDER BY customer_id;

-- name: GetPrimaryCustomerWallet :one
SELECT * FROM customer_wallets
WHERE customer_id = $1 AND is_primary = true AND deleted_at IS NULL
LIMIT 1;

-- name: GetCustomersByWalletAddress :many
SELECT c.* FROM customers c
JOIN customer_wallets cw ON c.id = cw.customer_id
WHERE cw.wallet_address = $1 AND c.deleted_at IS NULL AND cw.deleted_at IS NULL;

-- name: UpdateCustomerWallet :one
UPDATE customer_wallets SET
    nickname = $2,
    ens = $3,
    is_primary = $4,
    verified = $5,
    metadata = $6,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: GetCustomerIdForWallet :one
-- Get the customer_id for a wallet
SELECT customer_id FROM customer_wallets
WHERE id = $1 AND deleted_at IS NULL;

-- name: UnsetPrimaryForCustomerWallets :exec
-- Unset primary flag for all wallets of a customer except the specified wallet
UPDATE customer_wallets
SET is_primary = false, updated_at = CURRENT_TIMESTAMP
WHERE customer_id = $1
AND id != $2
AND deleted_at IS NULL;

-- name: MarkCustomerWalletAsPrimary :one
-- Set a specific customer wallet as primary
UPDATE customer_wallets
SET is_primary = true, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: VerifyCustomerWallet :one
UPDATE customer_wallets
SET verified = true, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteCustomerWallet :exec
UPDATE customer_wallets
SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteCustomerWalletsByCustomer :exec
UPDATE customer_wallets
SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE customer_id = $1;

-- name: CountCustomerWallets :one
SELECT COUNT(*) FROM customer_wallets
WHERE customer_id = $1 AND deleted_at IS NULL;

-- name: CountVerifiedCustomerWallets :one
SELECT COUNT(*) FROM customer_wallets
WHERE customer_id = $1 AND verified = true AND deleted_at IS NULL;

-- name: GetWalletsWithSimilarAddress :many
SELECT * FROM customer_wallets
WHERE wallet_address ILIKE $1 AND deleted_at IS NULL
ORDER BY customer_id, is_primary DESC
LIMIT $2;

-- name: UpdateCustomerWalletUsageTime :one
UPDATE customer_wallets
SET last_used_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL
RETURNING *; 