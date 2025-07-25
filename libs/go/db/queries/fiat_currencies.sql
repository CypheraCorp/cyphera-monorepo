-- name: GetFiatCurrency :one
SELECT * FROM fiat_currencies
WHERE code = $1 AND is_active = true
LIMIT 1;

-- name: GetFiatCurrencyByCode :one
SELECT * FROM fiat_currencies
WHERE code = $1
LIMIT 1;

-- name: ListActiveFiatCurrencies :many
SELECT * FROM fiat_currencies
WHERE is_active = true
ORDER BY code;

-- name: ListAllFiatCurrencies :many
SELECT * FROM fiat_currencies
ORDER BY code;

-- name: ListWorkspaceSupportedCurrencies :many
SELECT fc.* FROM fiat_currencies fc
INNER JOIN (
    SELECT jsonb_array_elements_text(supported_currencies) as currency_code
    FROM workspaces
    WHERE workspaces.id = $1
) w ON fc.code = w.currency_code
WHERE fc.is_active = true
ORDER BY fc.code;

-- name: UpdateFiatCurrency :one
UPDATE fiat_currencies
SET 
    name = $2,
    symbol = $3,
    decimal_places = $4,
    is_active = $5,
    symbol_position = $6,
    thousand_separator = $7,
    decimal_separator = $8,
    countries = $9,
    updated_at = CURRENT_TIMESTAMP
WHERE code = $1
RETURNING *;

-- name: ActivateFiatCurrency :exec
UPDATE fiat_currencies
SET is_active = true, updated_at = CURRENT_TIMESTAMP
WHERE code = $1;

-- name: DeactivateFiatCurrency :exec
UPDATE fiat_currencies
SET is_active = false, updated_at = CURRENT_TIMESTAMP
WHERE code = $1;

-- name: GetWorkspaceDefaultCurrency :one
SELECT fc.* FROM fiat_currencies fc
INNER JOIN workspaces w ON w.default_currency = fc.code
WHERE w.id = $1
LIMIT 1;

-- name: UpdateWorkspaceDefaultCurrency :exec
UPDATE workspaces
SET default_currency = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdateWorkspaceSupportedCurrencies :exec
UPDATE workspaces
SET supported_currencies = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: AddWorkspaceSupportedCurrency :exec
UPDATE workspaces
SET 
    supported_currencies = supported_currencies || $2::jsonb,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
AND NOT supported_currencies @> $2::jsonb;

-- name: RemoveWorkspaceSupportedCurrency :exec
UPDATE workspaces
SET 
    supported_currencies = supported_currencies - $2::text,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;