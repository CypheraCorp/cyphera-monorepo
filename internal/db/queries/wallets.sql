-- name: CreateWallet :one
INSERT INTO wallets (
    account_id,
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

-- name: GetWalletByID :one
SELECT * FROM wallets
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetWalletByAddress :one
SELECT * FROM wallets
WHERE wallet_address = $1 AND network_type = $2 AND deleted_at IS NULL;

-- name: ListWalletsByAccountID :many
SELECT * FROM wallets
WHERE account_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListPrimaryWalletsByAccountID :many
SELECT * FROM wallets
WHERE account_id = $1 AND is_primary = true AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListWalletsByNetworkType :many
SELECT * FROM wallets
WHERE account_id = $1 AND network_type = $2 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateWallet :one
UPDATE wallets
SET 
    nickname = COALESCE($1, nickname),
    ens = COALESCE($2, ens),
    is_primary = COALESCE($3, is_primary),
    verified = COALESCE($4, verified),
    metadata = COALESCE($5, metadata),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $6 AND deleted_at IS NULL
RETURNING *;

-- name: UpdateWalletVerificationStatus :one
UPDATE wallets
SET 
    verified = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: SetWalletAsPrimary :execrows
WITH updated_wallets AS (
    UPDATE wallets w
    SET is_primary = false
    WHERE w.account_id = $1 
    AND w.network_type = $2 
    AND w.is_primary = true 
    AND w.deleted_at IS NULL
)
UPDATE wallets w
SET 
    is_primary = true,
    updated_at = CURRENT_TIMESTAMP
WHERE w.id = $3 AND w.deleted_at IS NULL;

-- name: UpdateWalletLastUsed :exec
UPDATE wallets
SET 
    last_used_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: SoftDeleteWallet :exec
UPDATE wallets
SET 
    deleted_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetWalletStats :one
SELECT 
    COUNT(*) as total_wallets,
    COUNT(*) FILTER (WHERE verified = true) as verified_wallets,
    COUNT(*) FILTER (WHERE is_primary = true) as primary_wallets,
    COUNT(DISTINCT network_type) as network_types_count
FROM wallets
WHERE account_id = $1 AND deleted_at IS NULL;

-- name: SearchWallets :many
SELECT * FROM wallets
WHERE account_id = $1 
AND deleted_at IS NULL
AND (
    wallet_address ILIKE $2 
    OR nickname ILIKE $2 
    OR ens ILIKE $2
)
ORDER BY created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetWalletsByENS :many
SELECT * FROM wallets
WHERE account_id = $1 
AND ens IS NOT NULL 
AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetRecentlyUsedWallets :many
SELECT * FROM wallets
WHERE account_id = $1 
AND last_used_at IS NOT NULL 
AND deleted_at IS NULL
ORDER BY last_used_at DESC
LIMIT $2;
