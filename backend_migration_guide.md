# Golang Backend Web3Auth Migration Guide

## Overview

This guide details the complete migration of the Golang backend authentication system from Supabase JWT validation to Web3Auth JWT validation. The migration maintains existing API key authentication while replacing Supabase-specific JWT handling with Web3Auth JWKS-based validation.

## Current Authentication Architecture

### Current Implementation
- **JWT Validation**: Uses shared secret (`SUPABASE_JWT_SECRET`) with HS256 algorithm
- **Claims Structure**: `SupabaseClaims` with Supabase-specific fields
- **User Lookup**: `queries.GetUserBySupabaseID(claims.Sub)`
- **Middleware**: `authClient.EnsureValidAPIKeyOrToken()` supports both API keys and JWT tokens
- **Database**: Users table has `supabase_id` column for user identification

### Authentication Flow
1. Frontend sends JWT in `Authorization: Bearer <token>` header
2. Golang middleware validates JWT using shared secret
3. Extracts user info from Supabase claims
4. Looks up user by `supabase_id`
5. Sets user context for protected routes

## Required Migration Changes

### 1. Database Schema Updates

**Users Table Additions:**
```sql
-- Add Web3Auth identification columns
ALTER TABLE users ADD COLUMN web3auth_id VARCHAR(255) UNIQUE;
ALTER TABLE users ADD COLUMN verifier VARCHAR(100);           -- Login method (google, discord, etc.)
ALTER TABLE users ADD COLUMN verifier_id VARCHAR(255);        -- ID from the verifier

-- Add indexes for performance
CREATE INDEX idx_users_web3auth_id ON users(web3auth_id);
CREATE INDEX idx_users_verifier ON users(verifier);
CREATE INDEX idx_users_verifier_id ON users(verifier_id);
```

**Wallets Table Additions:**
```sql
-- Add new wallet type for Web3Auth Smart Accounts
ALTER TYPE wallet_type ADD VALUE 'web3auth';

-- Add Web3Auth specific columns
ALTER TABLE wallets ADD COLUMN web3auth_user_id VARCHAR(255);
ALTER TABLE wallets ADD COLUMN smart_account_type VARCHAR(50) CHECK (smart_account_type IN ('web3auth_eoa', 'web3auth_smart_account'));
ALTER TABLE wallets ADD COLUMN deployment_status VARCHAR(50) CHECK (deployment_status IN ('pending', 'deployed', 'failed'));

-- Add indexes for Web3Auth wallet lookups
CREATE INDEX idx_wallets_web3auth_user_id ON wallets(web3auth_user_id);
CREATE INDEX idx_wallets_smart_account_type ON wallets(smart_account_type);
CREATE INDEX idx_wallets_deployment_status ON wallets(deployment_status);
```

### 2. Database Queries Updates

**New Queries Required:**
```sql
-- name: GetUserByWeb3AuthID :one
SELECT * FROM users WHERE web3auth_id = $1 LIMIT 1;

-- name: CreateUserWithWeb3Auth :one
INSERT INTO users (
    account_id, email, web3auth_id, verifier, verifier_id, role, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, NOW(), NOW()
) RETURNING *;

-- name: GetWalletsByWeb3AuthUserID :many
SELECT * FROM wallets WHERE web3auth_user_id = $1;

-- name: GetPrimaryWeb3AuthWallet :one
SELECT * FROM wallets
WHERE web3auth_user_id = $1
AND wallet_type = 'web3auth'
AND is_primary = true
LIMIT 1;

-- name: CreateWeb3AuthWallet :one
INSERT INTO wallets (
    workspace_id, wallet_type, wallet_address, network_type, nickname, 
    is_primary, verified, web3auth_user_id, smart_account_type, deployment_status, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;
```

### 3. Environment Variables Migration

**Remove (Supabase):**
```env
SUPABASE_JWT_SECRET=yLGVcL5UC+w3GYsjVXolyWMc3opy+DrXa3js3fXMcw1JFpUf1kVmKqxhntNHNqRWuqjRJ9ehUcLcfaSTFwLoTA==
```

**Add (Web3Auth):**
```env
# Web3Auth JWT Configuration
WEB3AUTH_CLIENT_ID=BO3nb2gu5FJjApgENutlzcQOy8ZS47QydOpfth-v8i9dM7yQINDd6dFQ
WEB3AUTH_JWKS_ENDPOINT=https://api-auth.web3auth.io/.well-known/jwks.json
WEB3AUTH_ISSUER=https://api-auth.web3auth.io
WEB3AUTH_AUDIENCE=BO3nb2gu5FJjApgENutlzcQOy8ZS47QydOpfth-v8i9dM7yQINDd6dFQ

# Development Configuration
WEB3AUTH_ENVIRONMENT=sapphire_devnet
WEB3AUTH_DEBUG=true

# Production Configuration (when ready)
# WEB3AUTH_ENVIRONMENT=sapphire_mainnet
# WEB3AUTH_DEBUG=false
```

### 4. Golang Dependencies

**Add Required Dependencies:**
```go
go get github.com/golang-jwt/jwt/v5
go get github.com/MicahParks/keyfunc/v2
go get github.com/patrickmn/go-cache
```

**Import Updates:**
```go
import (
    "github.com/golang-jwt/jwt/v5"
    "github.com/MicahParks/keyfunc/v2"
    "github.com/patrickmn/go-cache"
    "crypto/rsa"
    "encoding/json"
    "net/http"
    "time"
)
```

### 5. Code Changes Required

#### 5.1 Update AuthClient Structure

**Current:**
```go
type AuthClient struct {
    SupabaseJWTToken string
    UserMetadata     map[string]interface{} `json:"user_metadata"`
}

func NewAuthClient(supabaseJWTToken string) *AuthClient {
    return &AuthClient{
        SupabaseJWTToken: supabaseJWTToken,
    }
}
```

**New:**
```go
type AuthClient struct {
    Web3AuthClientID    string
    Web3AuthJWKSURL     string
    Web3AuthIssuer      string
    Web3AuthAudience    string
    jwksCache          *cache.Cache
    keyFunc            keyfunc.Keyfunc
}

func NewAuthClient() *AuthClient {
    client := &AuthClient{
        Web3AuthClientID: os.Getenv("WEB3AUTH_CLIENT_ID"),
        Web3AuthJWKSURL:  os.Getenv("WEB3AUTH_JWKS_ENDPOINT"),
        Web3AuthIssuer:   os.Getenv("WEB3AUTH_ISSUER"),
        Web3AuthAudience: os.Getenv("WEB3AUTH_AUDIENCE"),
        jwksCache:        cache.New(30*time.Minute, 1*time.Hour), // Cache JWKS for 30 minutes
    }
    
    // Initialize JWKS key function
    err := client.initializeJWKS()
    if err != nil {
        logger.Fatal("Failed to initialize Web3Auth JWKS", zap.Error(err))
    }
    
    return client
}
```

#### 5.2 Replace Claims Structure

**Current:**
```go
type SupabaseClaims struct {
    jwt.RegisteredClaims
    Email        string                 `json:"email"`
    Sub          string                 `json:"sub"` // This is the user ID
    Role         string                 `json:"role"`
    AppMetadata  map[string]interface{} `json:"app_metadata"`
    UserMetadata map[string]interface{} `json:"user_metadata"`
}
```

**New:**
```go
type Web3AuthClaims struct {
    jwt.RegisteredClaims
    Email           string                 `json:"email"`
    Sub             string                 `json:"sub"`             // User ID from Web3Auth
    Name            string                 `json:"name"`
    Picture         string                 `json:"picture"`
    Verifier        string                 `json:"verifier"`        // Login method (google, discord, etc.)
    VerifierId      string                 `json:"verifierId"`      // ID from the verifier
    WalletAddress   string                 `json:"walletAddress"`   // Smart Account address
    PublicKey       string                 `json:"publicKey"`       // User's public key
    AppMetadata     map[string]interface{} `json:"app_metadata"`
    UserMetadata    map[string]interface{} `json:"user_metadata"`
}
```

#### 5.3 Replace JWT Validation Method

**Current:**
```go
func (ac *AuthClient) validateSupabaseToken(tokenString string) (*SupabaseClaims, error) {
    // Remove "Bearer " prefix if present
    tokenString = strings.TrimPrefix(tokenString, "Bearer ")

    // Get JWT secret from environment
    if ac.SupabaseJWTToken == "" {
        return nil, fmt.Errorf("SUPABASE_JWT_SECRET not set")
    }

    // Parse the token
    token, err := jwt.ParseWithClaims(tokenString, &SupabaseClaims{}, func(token *jwt.Token) (interface{}, error) {
        // Validate signing method - Supabase uses HS256
        if token.Method.Alg() != "HS256" {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }

        return []byte(ac.SupabaseJWTToken), nil
    })

    if err != nil {
        return nil, fmt.Errorf("failed to parse token: %w", err)
    }

    claims, ok := token.Claims.(*SupabaseClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token claims")
    }

    // Check if token is expired
    if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
        return nil, fmt.Errorf("token is expired")
    }

    return claims, nil
}
```

**New:**
```go
func (ac *AuthClient) initializeJWKS() error {
    // Create JWKS key function
    keyFunc, err := keyfunc.Get(ac.Web3AuthJWKSURL, keyfunc.Options{
        RefreshInterval: 30 * time.Minute,
        RefreshTimeout:  10 * time.Second,
    })
    if err != nil {
        return fmt.Errorf("failed to get JWKS: %w", err)
    }
    
    ac.keyFunc = keyFunc
    return nil
}

func (ac *AuthClient) validateWeb3AuthToken(tokenString string) (*Web3AuthClaims, error) {
    // Remove "Bearer " prefix if present
    tokenString = strings.TrimPrefix(tokenString, "Bearer ")

    // Validate required environment variables
    if ac.Web3AuthClientID == "" {
        return nil, fmt.Errorf("WEB3AUTH_CLIENT_ID not set")
    }
    if ac.Web3AuthJWKSURL == "" {
        return nil, fmt.Errorf("WEB3AUTH_JWKS_ENDPOINT not set")
    }

    // Parse the token using JWKS
    token, err := jwt.ParseWithClaims(tokenString, &Web3AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
        // Validate signing method - Web3Auth uses RS256
        if token.Method.Alg() != "RS256" {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }

        // Get the key from JWKS
        key, err := ac.keyFunc.Keyfunc(token)
        if err != nil {
            return nil, fmt.Errorf("failed to get key from JWKS: %w", err)
        }

        return key, nil
    })

    if err != nil {
        return nil, fmt.Errorf("failed to parse token: %w", err)
    }

    claims, ok := token.Claims.(*Web3AuthClaims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token claims")
    }

    // Validate issuer
    if claims.Issuer != ac.Web3AuthIssuer {
        return nil, fmt.Errorf("invalid issuer: expected %s, got %s", ac.Web3AuthIssuer, claims.Issuer)
    }

    // Validate audience
    if !claims.VerifyAudience(ac.Web3AuthAudience, true) {
        return nil, fmt.Errorf("invalid audience: expected %s", ac.Web3AuthAudience)
    }

    // Check if token is expired
    if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
        return nil, fmt.Errorf("token is expired")
    }

    return claims, nil
}
```

#### 5.4 Update User Validation Logic

**Current:**
```go
func (ac *AuthClient) validateJWTToken(c *gin.Context, queries *db.Queries, authHeader string) (db.User, db.Account, error) {
    claims, err := ac.validateSupabaseToken(authHeader)
    if err != nil {
        log.Printf("Token validation failed: %v", err)
        return db.User{}, db.Account{}, ErrInvalidToken
    }

    // Get user by Supabase ID (sub claim)
    user, err := queries.GetUserBySupabaseID(c.Request.Context(), claims.Sub)
    if err != nil {
        return db.User{}, db.Account{}, fmt.Errorf("user not found")
    }

    // Get user's account
    account, err := queries.GetAccountByID(c.Request.Context(), user.AccountID)
    if err != nil {
        return db.User{}, db.Account{}, fmt.Errorf("failed to get user account")
    }

    return user, account, nil
}
```

**New:**
```go
func (ac *AuthClient) validateJWTToken(c *gin.Context, queries *db.Queries, authHeader string) (db.User, db.Account, error) {
    claims, err := ac.validateWeb3AuthToken(authHeader)
    if err != nil {
        logger.Log.Debug("Web3Auth token validation failed", zap.Error(err))
        return db.User{}, db.Account{}, ErrInvalidToken
    }

    // Try to get existing user by Web3Auth ID
    user, err := queries.GetUserByWeb3AuthID(c.Request.Context(), claims.Sub)
    if err != nil {
        // If user doesn't exist, create new user automatically
        if err == sql.ErrNoRows {
            logger.Log.Info("Creating new user from Web3Auth claims", 
                zap.String("web3auth_id", claims.Sub),
                zap.String("email", claims.Email),
                zap.String("verifier", claims.Verifier),
            )
            
            user, err = ac.createUserFromWeb3AuthClaims(c.Request.Context(), queries, claims)
            if err != nil {
                logger.Log.Error("Failed to create user from Web3Auth claims", zap.Error(err))
                return db.User{}, db.Account{}, fmt.Errorf("failed to create user: %w", err)
            }
        } else {
            logger.Log.Error("Failed to get user by Web3Auth ID", zap.Error(err))
            return db.User{}, db.Account{}, fmt.Errorf("user lookup failed: %w", err)
        }
    }

    // Get user's account
    account, err := queries.GetAccountByID(c.Request.Context(), user.AccountID)
    if err != nil {
        logger.Log.Error("Failed to get user account", 
            zap.String("user_id", user.ID.String()),
            zap.String("account_id", user.AccountID.String()),
            zap.Error(err),
        )
        return db.User{}, db.Account{}, fmt.Errorf("failed to get user account: %w", err)
    }

    return user, account, nil
}
```

#### 5.5 Add User Creation Logic

**New Method:**
```go
func (ac *AuthClient) createUserFromWeb3AuthClaims(ctx context.Context, queries *db.Queries, claims *Web3AuthClaims) (db.User, error) {
    // Create account first (assuming AccountType.MERCHANT for Web3Auth users)
    account, err := queries.CreateAccount(ctx, db.CreateAccountParams{
        AccountType: db.AccountTypeMerchant, // Web3Auth users default to merchant accounts
        // Add other required fields based on your schema
    })
    if err != nil {
        return db.User{}, fmt.Errorf("failed to create account: %w", err)
    }

    // Create workspace for the account
    workspace, err := queries.CreateWorkspace(ctx, db.CreateWorkspaceParams{
        AccountID: account.ID,
        Name:      fmt.Sprintf("%s's Workspace", claims.Name),
        // Add other required fields
    })
    if err != nil {
        return db.User{}, fmt.Errorf("failed to create workspace: %w", err)
    }

    // Create user with Web3Auth data
    user, err := queries.CreateUserWithWeb3Auth(ctx, db.CreateUserWithWeb3AuthParams{
        AccountID:   account.ID,
        WorkspaceID: workspace.ID,
        Email:       claims.Email,
        Web3AuthID:  claims.Sub,
        Verifier:    claims.Verifier,
        VerifierID:  claims.VerifierId,
        Role:        db.UserRoleUser, // Default role
    })
    if err != nil {
        return db.User{}, fmt.Errorf("failed to create user: %w", err)
    }

    // Create Smart Account wallet if wallet address is provided
    if claims.WalletAddress != "" {
        err = ac.createSmartAccountWallet(ctx, queries, user, claims.WalletAddress, workspace.ID)
        if err != nil {
            // Log error but don't fail user creation
            logger.Log.Error("Failed to create Smart Account wallet",
                zap.String("user_id", user.ID.String()),
                zap.String("wallet_address", claims.WalletAddress),
                zap.Error(err),
            )
        }
    }

    logger.Log.Info("Successfully created Web3Auth user",
        zap.String("user_id", user.ID.String()),
        zap.String("account_id", account.ID.String()),
        zap.String("workspace_id", workspace.ID.String()),
        zap.String("web3auth_id", claims.Sub),
        zap.String("email", claims.Email),
        zap.String("verifier", claims.Verifier),
    )

    return user, nil
}

func (ac *AuthClient) createSmartAccountWallet(ctx context.Context, queries *db.Queries, user db.User, smartAccountAddress string, workspaceID uuid.UUID) error {
    // Create Smart Account wallet using existing wallet creation logic
    wallet, err := queries.CreateWeb3AuthWallet(ctx, db.CreateWeb3AuthWalletParams{
        WorkspaceID:      workspaceID,
        WalletType:       db.WalletTypeWeb3auth,
        WalletAddress:    smartAccountAddress,
        NetworkType:      "ethereum", // Default network
        Nickname:         pgtype.Text{String: "Web3Auth Smart Account", Valid: true},
        IsPrimary:        pgtype.Bool{Bool: true, Valid: true},
        Verified:         pgtype.Bool{Bool: true, Valid: true},
        Web3AuthUserID:   pgtype.Text{String: user.Web3AuthID, Valid: true},
        SmartAccountType: pgtype.Text{String: "web3auth_smart_account", Valid: true},
        DeploymentStatus: pgtype.Text{String: "deployed", Valid: true},
        Metadata: json.RawMessage(`{
            "created_via": "web3auth_auto_creation",
            "verifier": "` + user.Verifier + `",
            "verifier_id": "` + user.VerifierID + `",
            "created_at": "` + time.Now().UTC().Format(time.RFC3339) + `"
        }`),
    })
    if err != nil {
        return fmt.Errorf("failed to create Smart Account wallet: %w", err)
    }

    logger.Log.Info("Created Smart Account wallet for Web3Auth user",
        zap.String("user_id", user.ID.String()),
        zap.String("wallet_id", wallet.ID.String()),
        zap.String("wallet_address", wallet.WalletAddress),
        zap.String("web3auth_id", user.Web3AuthID),
    )

    return nil
}
```

#### 5.6 Update Server Initialization

**Current (in server.go):**
```go
// --- Supabase JWT Secret ---
supabaseJwtSecret, err := secretsClient.GetSecretString(ctx, "SUPABASE_JWT_SECRET_ARN", "SUPABASE_JWT_SECRET")
if err != nil || supabaseJwtSecret == "" {
    logger.Fatal("Failed to get Supabase JWT Secret", zap.Error(err))
}

// --- Auth Client ---
authClient = auth.NewAuthClient(supabaseJwtSecret)
```

**New:**
```go
// --- Web3Auth Configuration ---
web3AuthClientID, err := secretsClient.GetSecretString(ctx, "WEB3AUTH_CLIENT_ID_ARN", "WEB3AUTH_CLIENT_ID")
if err != nil || web3AuthClientID == "" {
    logger.Fatal("Failed to get Web3Auth Client ID", zap.Error(err))
}

web3AuthJWKSEndpoint, err := secretsClient.GetSecretString(ctx, "WEB3AUTH_JWKS_ENDPOINT_ARN", "WEB3AUTH_JWKS_ENDPOINT")
if err != nil || web3AuthJWKSEndpoint == "" {
    logger.Fatal("Failed to get Web3Auth JWKS Endpoint", zap.Error(err))
}

web3AuthIssuer, err := secretsClient.GetSecretString(ctx, "WEB3AUTH_ISSUER_ARN", "WEB3AUTH_ISSUER")
if err != nil || web3AuthIssuer == "" {
    logger.Fatal("Failed to get Web3Auth Issuer", zap.Error(err))
}

web3AuthAudience, err := secretsClient.GetSecretString(ctx, "WEB3AUTH_AUDIENCE_ARN", "WEB3AUTH_AUDIENCE")
if err != nil || web3AuthAudience == "" {
    logger.Fatal("Failed to get Web3Auth Audience", zap.Error(err))
}

// Set environment variables for AuthClient
os.Setenv("WEB3AUTH_CLIENT_ID", web3AuthClientID)
os.Setenv("WEB3AUTH_JWKS_ENDPOINT", web3AuthJWKSEndpoint)
os.Setenv("WEB3AUTH_ISSUER", web3AuthIssuer)
os.Setenv("WEB3AUTH_AUDIENCE", web3AuthAudience)

// --- Auth Client ---
authClient = auth.NewAuthClient()
```

### 6. Testing Requirements

#### 6.1 Unit Tests
Create tests for:
- `validateWeb3AuthToken()` method
- `createUserFromWeb3AuthClaims()` method
- `createSmartAccountWallet()` method
- JWKS key fetching and caching

#### 6.2 Integration Tests
Test scenarios:
- New Web3Auth user authentication and auto-creation
- Existing Web3Auth user authentication
- Invalid/expired JWT token handling
- Smart Account wallet creation
- API key authentication (should remain unchanged)

#### 6.3 Migration Validation
- Verify all existing API endpoints work with new authentication
- Test role-based access control
- Verify workspace and account associations
- Test error handling and logging

### 7. Backward Compatibility

During migration period, consider supporting both authentication methods:
```go
func (ac *AuthClient) validateJWTToken(c *gin.Context, queries *db.Queries, authHeader string) (db.User, db.Account, error) {
    // Try Web3Auth first
    if claims, err := ac.validateWeb3AuthToken(authHeader); err == nil {
        return ac.handleWeb3AuthUser(c, queries, claims)
    }
    
    // Fallback to Supabase for existing users
    if claims, err := ac.validateSupabaseToken(authHeader); err == nil {
        return ac.handleSupabaseUser(c, queries, claims)
    }
    
    return db.User{}, db.Account{}, ErrInvalidToken
}
```

### 8. Monitoring and Logging

Add specific logging for Web3Auth operations:
```go
// Log successful Web3Auth authentication
logger.Log.Info("Web3Auth authentication successful",
    zap.String("web3auth_id", claims.Sub),
    zap.String("email", claims.Email),
    zap.String("verifier", claims.Verifier),
    zap.String("wallet_address", claims.WalletAddress),
)

// Log user creation
logger.Log.Info("Auto-created Web3Auth user",
    zap.String("user_id", user.ID.String()),
    zap.String("web3auth_id", claims.Sub),
    zap.String("verifier", claims.Verifier),
)

// Log authentication failures
logger.Log.Warn("Web3Auth authentication failed",
    zap.Error(err),
    zap.String("token_preview", tokenString[:20]+"..."),
)
```

### 9. Security Considerations

- **JWKS Caching**: Implement proper caching to avoid excessive JWKS requests
- **Token Validation**: Verify issuer, audience, and expiration
- **Rate Limiting**: Consider adding rate limiting for failed authentication attempts
- **Error Messages**: Don't expose sensitive information in error messages
- **Logging**: Log authentication attempts for security monitoring

### 10. Implementation Steps

1. **Database Migration**: Apply schema changes
2. **Update Dependencies**: Add required Go packages
3. **Environment Variables**: Update configuration management
4. **Code Changes**: Implement Web3Auth authentication logic
5. **Testing**: Comprehensive testing of all scenarios
6. **Deployment**: Staged deployment with monitoring
7. **Validation**: End-to-end testing in production environment

### 11. Rollback Plan

If issues arise during migration:
1. **Immediate**: Revert environment variables to Supabase
2. **Code Rollback**: Deploy previous version with Supabase authentication
3. **Database**: New columns are additive, so no rollback needed
4. **Monitoring**: Monitor for authentication failures and user creation issues

This migration guide provides the complete context needed to update the Golang backend for Web3Auth integration while maintaining existing functionality and security standards. 