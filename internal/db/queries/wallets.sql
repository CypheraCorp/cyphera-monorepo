-- name: CreateWallet :one
INSERT INTO wallets (
    account_id,
    wallet_type,
    wallet_address,
    network_type,
    network_id,
    nickname,
    ens,
    is_primary,
    verified,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetWalletByID :one
SELECT * FROM wallets
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetWalletWithCircleDataByID :one
SELECT 
    w.*,
    cw.id as circle_wallet_id,
    cw.circle_user_id,
    cw.circle_wallet_id as circle_id,
    cw.chain_id,
    cw.state as circle_state
FROM wallets w
LEFT JOIN circle_wallets cw ON w.id = cw.wallet_id AND w.wallet_type = 'circle_wallet'
WHERE w.id = $1 AND w.deleted_at IS NULL;

-- name: GetWalletByAddress :one
SELECT * FROM wallets
WHERE wallet_address = $1 AND network_type = $2 AND deleted_at IS NULL;

-- name: GetWalletWithCircleDataByAddress :one
SELECT 
    w.*,
    cw.id as circle_wallet_id,
    cw.circle_user_id,
    cw.circle_wallet_id as circle_id,
    cw.chain_id,
    cw.state as circle_state
FROM wallets w
LEFT JOIN circle_wallets cw ON w.id = cw.wallet_id AND w.wallet_type = 'circle_wallet'
WHERE w.wallet_address = $1 AND w.network_type = $2 AND w.deleted_at IS NULL;

-- name: ListWalletsByAccountID :many
SELECT * FROM wallets
WHERE account_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListWalletsWithCircleDataByAccountID :many
SELECT 
    w.*,
    cw.id as circle_wallet_id,
    cw.circle_user_id,
    cw.circle_wallet_id as circle_id,
    cw.chain_id,
    cw.state as circle_state
FROM wallets w
LEFT JOIN circle_wallets cw ON w.id = cw.wallet_id AND w.wallet_type = 'circle_wallet'
WHERE w.account_id = $1 AND w.deleted_at IS NULL
ORDER BY w.created_at DESC;

-- name: ListCircleWalletsByAccountID :many
SELECT 
    w.*,
    cw.id as circle_wallet_id,
    cw.circle_user_id,
    cw.circle_wallet_id as circle_id,
    cw.chain_id,
    cw.state as circle_state
FROM wallets w
JOIN circle_wallets cw ON w.id = cw.wallet_id
WHERE w.account_id = $1 AND w.wallet_type = 'circle_wallet' AND w.deleted_at IS NULL
ORDER BY w.created_at DESC;

-- name: ListCircleWalletsByCircleUserID :many
SELECT 
    w.*,
    cw.id as circle_wallet_id,
    cw.circle_user_id,
    cw.circle_wallet_id as circle_id,
    cw.chain_id,
    cw.state as circle_state
FROM wallets w
JOIN circle_wallets cw ON w.id = cw.wallet_id
WHERE cw.circle_user_id = $1 AND w.wallet_type = 'circle_wallet' AND w.deleted_at IS NULL
ORDER BY w.created_at DESC;

-- name: ListPrimaryWalletsByAccountID :many
SELECT * FROM wallets
WHERE account_id = $1 AND is_primary = true AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListPrimaryWalletsWithCircleDataByAccountID :many
SELECT 
    w.*,
    cw.id as circle_wallet_id,
    cw.circle_user_id,
    cw.circle_wallet_id as circle_id,
    cw.chain_id,
    cw.state as circle_state
FROM wallets w
LEFT JOIN circle_wallets cw ON w.id = cw.wallet_id AND w.wallet_type = 'circle_wallet'
WHERE w.account_id = $1 AND w.is_primary = true AND w.deleted_at IS NULL
ORDER BY w.created_at DESC;

-- name: ListWalletsByNetworkType :many
SELECT * FROM wallets
WHERE account_id = $1 AND network_type = $2 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListWalletsWithCircleDataByNetworkType :many
SELECT 
    w.*,
    cw.id as circle_wallet_id,
    cw.circle_user_id,
    cw.circle_wallet_id as circle_id,
    cw.chain_id,
    cw.state as circle_state
FROM wallets w
LEFT JOIN circle_wallets cw ON w.id = cw.wallet_id AND w.wallet_type = 'circle_wallet'
WHERE w.account_id = $1 AND w.network_type = $2 AND w.deleted_at IS NULL
ORDER BY w.created_at DESC;

-- name: ListWalletsByWalletType :many
SELECT * FROM wallets
WHERE account_id = $1 AND wallet_type = $2 AND deleted_at IS NULL
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
    COUNT(DISTINCT network_type) as network_types_count,
    COUNT(*) FILTER (WHERE wallet_type = 'wallet') as standard_wallets_count,
    COUNT(*) FILTER (WHERE wallet_type = 'circle_wallet') as circle_wallets_count
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

-- name: SearchWalletsWithCircleData :many
SELECT 
    w.*,
    cw.id as circle_wallet_id,
    cw.circle_user_id,
    cw.circle_wallet_id as circle_id,
    cw.chain_id,
    cw.state as circle_state
FROM wallets w
LEFT JOIN circle_wallets cw ON w.id = cw.wallet_id AND w.wallet_type = 'circle_wallet'
WHERE w.account_id = $1 
AND w.deleted_at IS NULL
AND (
    w.wallet_address ILIKE $2 
    OR w.nickname ILIKE $2 
    OR w.ens ILIKE $2
    OR cw.circle_wallet_id ILIKE $2
)
ORDER BY w.created_at DESC
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

-- name: GetRecentlyUsedWalletsWithCircleData :many
SELECT 
    w.*,
    cw.id as circle_wallet_id,
    cw.circle_user_id,
    cw.circle_wallet_id as circle_id,
    cw.chain_id,
    cw.state as circle_state
FROM wallets w
LEFT JOIN circle_wallets cw ON w.id = cw.wallet_id AND w.wallet_type = 'circle_wallet'
WHERE w.account_id = $1 
AND w.last_used_at IS NOT NULL 
AND w.deleted_at IS NULL
ORDER BY w.last_used_at DESC
LIMIT $2;

-- name: CreateCircleWalletEntry :one
INSERT INTO circle_wallets (
    wallet_id,
    circle_user_id,
    circle_wallet_id,
    chain_id,
    state
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: UpdateCircleWalletState :one
UPDATE circle_wallets
SET 
    state = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE wallet_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: GetCircleWalletByCircleWalletID :one
SELECT 
    w.*,
    cw.id as circle_wallet_id,
    cw.circle_user_id,
    cw.circle_wallet_id as circle_id,
    cw.chain_id,
    cw.state as circle_state
FROM wallets w
JOIN circle_wallets cw ON w.id = cw.wallet_id
WHERE cw.circle_wallet_id = $1 AND w.deleted_at IS NULL;
