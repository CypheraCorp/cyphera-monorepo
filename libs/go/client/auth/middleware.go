package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/constants"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/logger"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

var (
	// ErrInvalidToken is returned when the provided token is invalid
	ErrInvalidToken = errors.New("invalid token")
)

// Context keys for storing values (moved from handlers/middleware.go)
const (
	RequestIDKey = "request_id"
	AccountIDKey = "X-Account-ID" // Note: This might conflict if you have a different AccountIDKey in this package already.
	UserIDKey    = "X-User-ID"
)

// Web3AuthWallet represents a wallet in the Web3Auth ID token (actual structure from example)
type Web3AuthWallet struct {
	PublicKey string `json:"public_key"`
	Type      string `json:"type"`
	Curve     string `json:"curve,omitempty"`
	Address   string `json:"address,omitempty"`
}

// Web3AuthClaims represents the actual structure of the Web3Auth JWT claims
type Web3AuthClaims struct {
	jwt.RegisteredClaims
	Email                   string           `json:"email"`
	Name                    string           `json:"name"`
	ProfileImage            string           `json:"profileImage"`
	Verifier                string           `json:"verifier"`
	AuthConnectionId        string           `json:"authConnectionId,omitempty"`
	VerifierId              string           `json:"verifierId"`
	UserId                  string           `json:"userId"`
	AggregateVerifier       string           `json:"aggregateVerifier,omitempty"`
	GroupedAuthConnectionId string           `json:"groupedAuthConnectionId,omitempty"`
	Wallets                 []Web3AuthWallet `json:"wallets,omitempty"`
	Nonce                   string           `json:"nonce,omitempty"`
}

type AuthClient struct {
	Web3AuthClientID string
	Web3AuthJWKSURL  string
	Web3AuthIssuer   string
	Web3AuthAudience string
	jwks             *keyfunc.JWKS
}

func NewAuthClient() *AuthClient {
	client := &AuthClient{
		Web3AuthClientID: os.Getenv("WEB3AUTH_CLIENT_ID"),
		Web3AuthJWKSURL:  os.Getenv("WEB3AUTH_JWKS_ENDPOINT"),
		Web3AuthIssuer:   os.Getenv("WEB3AUTH_ISSUER"),
		Web3AuthAudience: os.Getenv("WEB3AUTH_AUDIENCE"),
	}

	// Initialize JWKS
	if err := client.initializeJWKS(); err != nil {
		logger.Log.Error("Failed to initialize JWKS", zap.Error(err))
	}

	return client
}

// RequestLog represents a structured log entry for an HTTP request (moved from handlers/middleware.go)
type RequestLog struct {
	Method    string    `json:"method"`
	Path      string    `json:"path,omitempty"`
	Query     string    `json:"query,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
	ClientIP  string    `json:"client_ip"`
	RequestID string    `json:"request_id,omitempty"`
	AccountID string    `json:"account_id,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// EnsureValidAPIKeyOrToken is a middleware that checks for either a valid API key or JWT token
func (ac *AuthClient) EnsureValidAPIKeyOrToken(services CommonServicesInterface) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Debug: Log all headers to see what's being sent
		logger.Log.Info("Request headers in auth middleware",
			zap.String("authorization", c.GetHeader("Authorization")),
			zap.String("x-api-key", c.GetHeader("X-API-Key")),
			zap.String("x-workspace-id", c.GetHeader("X-Workspace-ID")),
			zap.String("x-workspace-id-lowercase", c.GetHeader("X-Workspace-Id")),
			zap.String("x-account-id", c.GetHeader("X-Account-ID")),
			zap.String("user-agent", c.GetHeader("User-Agent")),
			zap.String("path", c.Request.URL.Path),
		)

		logger.Log.Info("Checking for API key in header")

		// First check for API key in header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			// Log the API key being used (first few characters for security)
			keyPreview := ""
			if len(apiKey) > 4 {
				keyPreview = apiKey[:4] + "..."
			} else {
				keyPreview = "too_short"
			}
			logger.Log.Debug("Received API key in header",
				zap.String("key_preview", keyPreview),
			)

			workspace, account, key, err := ac.validateAPIKey(c, services, apiKey)
			if err != nil {
				logger.Log.Debug("API key validation failed",
					zap.Error(err),
				)
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				c.Abort()
				return
			}

			// Set context with workspace and account information
			c.Set("workspaceID", workspace.ID.String())
			c.Set("accountID", account.ID.String())
			c.Set("accountType", string(account.AccountType))
			c.Set("apiKeyLevel", string(key.AccessLevel))
			c.Set("authType", constants.AuthTypeAPIKey)
			c.Next()
			return
		}

		logger.Log.Info("No API key found, checking for JWT token")

		// If no API key, check for JWT token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Log.Debug("No authentication provided")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No authentication provided"})
			c.Abort()
			return
		}

		logger.Log.Info("JWT token found, starting validation")

		// Special handling for Web3Auth users without JWT tokens
		if authHeader == "Bearer no_jwt_token_available" {
			logger.Log.Debug("Web3Auth user without JWT token - authentication not supported yet")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Web3Auth JWT token required for authentication"})
			c.Abort()
			return
		}

		logger.Log.Info("Starting JWT token validation",
			zap.String("path", c.Request.URL.Path),
			zap.String("correlation_id", c.GetHeader("X-Correlation-ID")),
		)

		logger.Log.Info("About to call validateJWTToken",
			zap.String("path", c.Request.URL.Path),
			zap.String("correlation_id", c.GetHeader("X-Correlation-ID")),
		)

		user, account, err := ac.validateJWTToken(c, services, authHeader)
		if err != nil {
			logger.Log.Error("JWT token validation failed",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path),
				zap.String("correlation_id", c.GetHeader("X-Correlation-ID")),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		logger.Log.Info("JWT token validated successfully",
			zap.String("user_id", user.ID.String()),
			zap.String("account_id", account.ID.String()),
			zap.String("path", c.Request.URL.Path),
			zap.String("correlation_id", c.GetHeader("X-Correlation-ID")),
		)

		// require workspace ID in the header - check both cases
		logger.Log.Info("Starting workspace validation",
			zap.String("path", c.Request.URL.Path),
			zap.String("correlation_id", c.GetHeader("X-Correlation-ID")),
		)

		workspaceID := c.GetHeader("X-Workspace-ID")
		if workspaceID == "" {
			workspaceID = c.GetHeader("X-Workspace-Id")
		}
		if workspaceID == "" {
			logger.Log.Error("No workspace ID provided",
				zap.String("X-Workspace-ID", c.GetHeader("X-Workspace-ID")),
				zap.String("X-Workspace-Id", c.GetHeader("X-Workspace-Id")),
				zap.Any("headers", c.Request.Header),
				zap.String("path", c.Request.URL.Path),
				zap.String("correlation_id", c.GetHeader("X-Correlation-ID")),
			)
			c.JSON(http.StatusBadRequest, gin.H{"error": "No workspace ID provided"})
			c.Abort()
			return
		}

		logger.Log.Info("Found workspace ID in header",
			zap.String("workspace_id", workspaceID),
			zap.String("path", c.Request.URL.Path),
			zap.String("correlation_id", c.GetHeader("X-Correlation-ID")),
		)

		// Validate that the workspace belongs to the user's account
		workspaceUUID, err := uuid.Parse(workspaceID)
		if err != nil {
			logger.Log.Error("Invalid workspace ID format",
				zap.String("workspace_id", workspaceID),
				zap.Error(err),
			)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid workspace ID format"})
			c.Abort()
			return
		}

		// Get workspace to verify it belongs to the account
		workspace, err := services.GetDB().GetWorkspace(c.Request.Context(), workspaceUUID)
		if err != nil {
			logger.Log.Error("Failed to get workspace",
				zap.String("workspace_id", workspaceID),
				zap.String("account_id", account.ID.String()),
				zap.Error(err),
			)
			c.JSON(http.StatusForbidden, gin.H{"error": "Workspace not found or access denied"})
			c.Abort()
			return
		}

		// Verify workspace belongs to the user's account
		if workspace.AccountID != account.ID {
			logger.Log.Error("Workspace does not belong to user's account",
				zap.String("workspace_id", workspaceID),
				zap.String("workspace_account_id", workspace.AccountID.String()),
				zap.String("user_account_id", account.ID.String()),
			)
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to workspace"})
			c.Abort()
			return
		}

		// Set context with user and account information
		c.Set("userID", user.ID.String())
		c.Set("accountID", account.ID.String())
		c.Set("workspaceID", workspaceID)
		c.Set("accountType", string(account.AccountType))
		c.Set("userRole", string(user.Role))
		c.Set("authType", constants.AuthTypeJWT)
		c.Next()
	}
}

// validateAPIKey validates the API key and returns workspace and account information
func (ac *AuthClient) validateAPIKey(c *gin.Context, services CommonServicesInterface, apiKey string) (db.Workspace, db.Account, db.ApiKey, error) {
	// Log the API key being validated (first few characters for security)
	keyPreview := ""
	if len(apiKey) > 4 {
		keyPreview = apiKey[:4] + "..."
	} else {
		keyPreview = "too_short"
	}
	logger.Log.Debug("Validating API key",
		zap.String("key_preview", keyPreview),
		zap.Int("key_length", len(apiKey)),
	)

	// Get all active API keys to compare against (bcrypt comparison required)
	// Note: In production with many keys, consider caching or using a different strategy
	activeKeys, err := services.GetDB().GetAllActiveAPIKeys(c.Request.Context())
	if err != nil {
		logger.Log.Debug("Failed to retrieve active API keys",
			zap.Error(err),
		)
		return db.Workspace{}, db.Account{}, db.ApiKey{}, fmt.Errorf("authentication service error")
	}

	// Find matching key by comparing bcrypt hashes
	var key db.ApiKey
	found := false
	for _, k := range activeKeys {
		if err := helpers.CompareAPIKeyHash(apiKey, k.KeyHash); err == nil {
			key = k
			found = true
			break
		}
	}

	if !found {
		logger.Log.Debug("API key not found or invalid",
			zap.String("key_preview", keyPreview),
		)
		return db.Workspace{}, db.Account{}, db.ApiKey{}, fmt.Errorf("invalid API key")
	}

	// Update last used timestamp
	go func() {
		if err := services.GetDB().UpdateAPIKeyLastUsed(context.Background(), key.ID); err != nil {
			logger.Log.Warn("Failed to update API key last used timestamp",
				zap.String("key_id", key.ID.String()),
				zap.Error(err),
			)
		}
	}()

	logger.Log.Debug("API key found",
		zap.String("key_id", key.ID.String()),
		zap.String("workspace_id", key.WorkspaceID.String()),
	)

	// Check if API key is expired
	if key.ExpiresAt.Valid && key.ExpiresAt.Time.Before(time.Now()) {
		logger.Log.Debug("API key expired",
			zap.String("key_id", key.ID.String()),
			zap.Time("expires_at", key.ExpiresAt.Time),
		)
		return db.Workspace{}, db.Account{}, db.ApiKey{}, fmt.Errorf("API key has expired")
	}

	// Get workspace associated with API key
	workspace, err := services.GetDB().GetWorkspace(c.Request.Context(), key.WorkspaceID)
	if err != nil {
		logger.Log.Debug("Failed to get workspace for API key",
			zap.String("key_id", key.ID.String()),
			zap.String("workspace_id", key.WorkspaceID.String()),
			zap.Error(err),
		)
		return db.Workspace{}, db.Account{}, db.ApiKey{}, fmt.Errorf("invalid workspace")
	}

	// Get account associated with workspace
	account, err := services.GetDB().GetAccount(c.Request.Context(), workspace.AccountID)
	if err != nil {
		logger.Log.Debug("Failed to get account for workspace",
			zap.String("workspace_id", workspace.ID.String()),
			zap.String("account_id", workspace.AccountID.String()),
			zap.Error(err),
		)
		return db.Workspace{}, db.Account{}, db.ApiKey{}, fmt.Errorf("invalid account")
	}

	logger.Log.Debug("API key validation successful",
		zap.String("key_id", key.ID.String()),
		zap.String("workspace_id", workspace.ID.String()),
		zap.String("account_id", account.ID.String()),
	)

	return workspace, account, key, nil
}

// validateJWTToken validates the Web3Auth JWT token and returns user information
func (ac *AuthClient) validateJWTToken(c *gin.Context, services CommonServicesInterface, authHeader string) (db.User, db.Account, error) {
	logger.Log.Info("validateJWTToken called",
		zap.String("correlation_id", c.GetHeader("X-Correlation-ID")),
		zap.String("path", c.Request.URL.Path),
		zap.Bool("has_auth_header", authHeader != ""),
	)

	claims, err := ac.validateWeb3AuthToken(authHeader)
	if err != nil {
		logger.Log.Info("Web3Auth token validation failed", zap.Error(err))
		return db.User{}, db.Account{}, ErrInvalidToken
	}

	// Extract Web3Auth ID from token - prefer userId over subject
	// Parse the token as MapClaims to get all fields
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	mapToken, _, mapErr := parser.ParseUnverified(tokenString, jwt.MapClaims{})

	var web3AuthID string
	if mapErr == nil {
		mapClaims := mapToken.Claims.(jwt.MapClaims)
		logger.Log.Debug("Raw token claims",
			zap.Any("all_claims", mapClaims),
		)

		// Try to extract Web3Auth ID from various fields
		if userId, ok := mapClaims["userId"].(string); ok && userId != "" {
			web3AuthID = userId
		} else if verifierId, ok := mapClaims["verifierId"].(string); ok && verifierId != "" {
			web3AuthID = verifierId
		} else if claims.Subject != "" {
			web3AuthID = claims.Subject
		} else if claims.Email != "" {
			web3AuthID = claims.Email
		}

		// Also extract other Web3Auth fields for user creation
		if verifier, ok := mapClaims["verifier"].(string); ok {
			claims.Verifier = verifier
		}
		if verifierId, ok := mapClaims["verifierId"].(string); ok {
			claims.VerifierId = verifierId
		}
		if name, ok := mapClaims["name"].(string); ok {
			claims.Name = name
		}
		if email, ok := mapClaims["email"].(string); ok {
			claims.Email = email
		}
	} else {
		// Fallback to struct-based parsing
		if claims.UserId != "" {
			web3AuthID = claims.UserId
		} else if claims.Subject != "" {
			web3AuthID = claims.Subject
		} else {
			web3AuthID = claims.Email
		}
	}

	logger.Log.Debug("Using Web3Auth ID for user lookup",
		zap.String("web3auth_id", web3AuthID),
		zap.String("claims_subject", claims.Subject),
		zap.String("claims_userId", claims.UserId),
		zap.String("claims_email", claims.Email),
		zap.String("claims_verifier", claims.Verifier),
		zap.String("claims_verifierId", claims.VerifierId),
	)

	// Try to get existing user by Web3Auth ID
	user, err := services.GetDB().GetUserByWeb3AuthID(c.Request.Context(), pgtype.Text{String: web3AuthID, Valid: true})
	if err != nil {
		logger.Log.Debug("GetUserByWeb3AuthID returned error",
			zap.Error(err),
			zap.String("error_type", fmt.Sprintf("%T", err)),
			zap.Bool("is_no_rows", err == pgx.ErrNoRows),
		)
		// If user doesn't exist, create new user automatically
		if err == pgx.ErrNoRows {
			logger.Log.Info("Creating new user from Web3Auth claims",
				zap.String("web3auth_id", web3AuthID),
				zap.String("email", claims.Email),
				zap.String("verifier", claims.Verifier),
				zap.String("verifier_id", claims.VerifierId),
				zap.String("user_id", claims.UserId),
			)

			user, err = ac.createUserFromWeb3AuthClaims(c.Request.Context(), services, claims, web3AuthID)
			if err != nil {
				logger.Log.Error("Failed to create user from Web3Auth claims",
					zap.Error(err),
					zap.String("web3auth_id", web3AuthID),
					zap.String("email", claims.Email),
				)
				return db.User{}, db.Account{}, fmt.Errorf("failed to create user: %w", err)
			}
			logger.Log.Info("Successfully auto-created Web3Auth user",
				zap.String("user_id", user.ID.String()),
				zap.String("web3auth_id", web3AuthID),
				zap.String("email", claims.Email),
			)
		} else {
			logger.Log.Error("Failed to get user by Web3Auth ID", zap.Error(err))
			return db.User{}, db.Account{}, fmt.Errorf("user lookup failed: %w", err)
		}
	}

	// Get user's account
	account, err := services.GetDB().GetAccountByID(c.Request.Context(), user.AccountID)
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

func (ac *AuthClient) validateWeb3AuthToken(tokenString string) (*Web3AuthClaims, error) {
	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// Create safe token preview for logging
	tokenPreview := tokenString
	if len(tokenString) > 20 {
		tokenPreview = tokenString[:20] + "..."
	} else if len(tokenString) > 0 {
		tokenPreview = tokenString + " (short_token)"
	} else {
		tokenPreview = "empty_token"
	}

	logger.Log.Info("Validating Web3Auth token",
		zap.String("token_preview", tokenPreview),
		zap.Int("token_length", len(tokenString)),
		zap.String("jwks_url", ac.Web3AuthJWKSURL),
		zap.String("issuer", ac.Web3AuthIssuer),
		zap.String("audience", ac.Web3AuthAudience),
		zap.Bool("jwks_initialized", ac.jwks != nil),
	)

	// Validate that JWKS is initialized
	if ac.jwks == nil {
		return nil, fmt.Errorf("JWKS not initialized")
	}

	// Parse and validate the token using JWKS
	token, err := jwt.ParseWithClaims(tokenString, &Web3AuthClaims{}, ac.jwks.Keyfunc)
	if err != nil {
		logger.Log.Info("Token parsing failed", zap.Error(err))

		// For debugging, also try parsing without verification to see claims
		parser := jwt.NewParser(jwt.WithoutClaimsValidation())
		unverifiedToken, _, parseErr := parser.ParseUnverified(tokenString, &Web3AuthClaims{})
		if parseErr == nil {
			if claims, ok := unverifiedToken.Claims.(*Web3AuthClaims); ok {
				logger.Log.Info("Unverified token claims (for debugging)",
					zap.String("sub", claims.Subject),
					zap.String("email", claims.Email),
					zap.String("iss", claims.Issuer),
					zap.Strings("aud", claims.Audience),
					zap.String("userId", claims.UserId),
					zap.String("verifier", claims.Verifier),
					zap.String("verifierId", claims.VerifierId),
					zap.String("name", claims.Name),
					zap.Int("wallets_count", len(claims.Wallets)),
				)
			}
		}

		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Validate that the token is valid
	if !token.Valid {
		return nil, fmt.Errorf("token is not valid")
	}

	claims, ok := token.Claims.(*Web3AuthClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Basic claims validation
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("token is expired")
	}

	// Validate issuer if configured
	if ac.Web3AuthIssuer != "" && claims.Issuer != ac.Web3AuthIssuer {
		logger.Log.Debug("Issuer mismatch",
			zap.String("expected", ac.Web3AuthIssuer),
			zap.String("actual", claims.Issuer),
		)
		return nil, fmt.Errorf("invalid issuer")
	}

	// Validate audience if configured
	if ac.Web3AuthAudience != "" {
		audienceValid := false
		for _, aud := range claims.Audience {
			if aud == ac.Web3AuthAudience {
				audienceValid = true
				break
			}
		}
		if !audienceValid {
			logger.Log.Debug("Audience mismatch",
				zap.String("expected", ac.Web3AuthAudience),
				zap.Strings("actual", claims.Audience),
			)
			return nil, fmt.Errorf("invalid audience")
		}
	}

	logger.Log.Debug("Web3Auth token validation successful",
		zap.String("sub", claims.Subject),
		zap.String("email", claims.Email),
		zap.String("userId", claims.UserId),
		zap.String("verifier", claims.Verifier),
		zap.String("verifierId", claims.VerifierId),
		zap.Int("wallets_count", len(claims.Wallets)),
	)

	return claims, nil
}

// RequireRoles is a middleware that checks if the user has the required  roles
func (ac *AuthClient) RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountType := c.GetString("accountType")
		apiKeyLevel := c.GetString("apiKeyLevel")
		authType := c.GetString("authType")

		// For API key auth, check access level
		if authType == constants.AuthTypeAPIKey {
			if apiKeyLevel != constants.AccessLevelAdmin {
				logger.Log.Debug("Insufficient API key access level")
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient API key access level"})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// For admin-only operations, check account type
		if roles[0] == constants.RoleAdmin && accountType != constants.AccountTypeAdmin {
			logger.Log.Debug("Admin access required")
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// --- Logging Middleware (Moved from internal/handlers/middleware.go) ---

// shouldSkipLogging determines if request logging should be skipped for a given path
func shouldSkipLogging(path string) bool {
	// Skip logging for health check endpoints
	if path == "/healthz" || path == "/readyz" {
		return true
	}
	return false
}

// getRequestBody safely reads and returns the request body
func getRequestBody(c *gin.Context) ([]byte, error) {
	var bodyBytes []byte
	if c.Request.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(c.Request.Body)
		if err != nil {
			return nil, err
		}
		// Restore the request body for subsequent middleware/handlers
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	return bodyBytes, nil
}

// LogRequest is a middleware that logs the request body
func LogRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for certain paths
		if shouldSkipLogging(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()

		// Get request body
		bodyBytes, err := getRequestBody(c)
		if err != nil {
			logger.Log.Debug("Request body reading failed",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
			)
			logger.Log.Error("Failed to read request body", zap.Error(err))
			c.Next()
			return
		}

		// Attempt to unmarshal body for pretty printing
		var bodyField zap.Field
		var prettyBody interface{}
		if len(bodyBytes) > 0 && json.Unmarshal(bodyBytes, &prettyBody) == nil {
			bodyField = zap.Any("body", prettyBody) // Log parsed JSON
		} else {
			// Fallback to string if not JSON or empty
			bodyField = zap.String("body", string(bodyBytes))
		}

		// Prepare base log entry (without body initially)
		requestLog := RequestLog{
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Query:     c.Request.URL.RawQuery,
			UserAgent: c.Request.UserAgent(),
			ClientIP:  c.ClientIP(),
			RequestID: c.GetHeader(RequestIDKey),
			AccountID: c.GetHeader(AccountIDKey),
			UserID:    c.GetHeader(UserIDKey),
			Timestamp: start.UTC(),
		}

		// Log the request details including the prepared body field
		logger.Log.Debug("Request received",
			zap.String("method", requestLog.Method),
			zap.String("path", requestLog.Path),
			zap.String("query", requestLog.Query),
			zap.String("user_agent", requestLog.UserAgent),
			zap.String("client_ip", requestLog.ClientIP),
			zap.String("request_id", requestLog.RequestID),
			zap.String("account_id", requestLog.AccountID),
			zap.String("user_id", requestLog.UserID),
			bodyField, // Add the dynamically created body field here
			zap.Time("timestamp", requestLog.Timestamp),
		)

		c.Next()
	}
}

func (ac *AuthClient) initializeJWKS() error {
	if ac.Web3AuthJWKSURL == "" {
		return fmt.Errorf("WEB3AUTH_JWKS_ENDPOINT not set")
	}

	// Create JWKS from the Web3Auth JWKS URL
	jwks, err := keyfunc.Get(ac.Web3AuthJWKSURL, keyfunc.Options{
		RefreshInterval:  time.Hour,        // Refresh keys every hour
		RefreshRateLimit: time.Minute,      // Rate limit refreshes to once per minute
		RefreshTimeout:   time.Second * 10, // Timeout for refresh requests
		RefreshErrorHandler: func(err error) {
			logger.Log.Error("JWKS refresh error", zap.Error(err))
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create JWKS: %w", err)
	}

	ac.jwks = jwks

	logger.Log.Info("Web3Auth JWKS initialized successfully",
		zap.String("jwks_url", ac.Web3AuthJWKSURL),
		zap.String("client_id", ac.Web3AuthClientID),
		zap.String("issuer", ac.Web3AuthIssuer),
	)

	return nil
}

func (ac *AuthClient) createUserFromWeb3AuthClaims(ctx context.Context, services CommonServicesInterface, claims *Web3AuthClaims, web3AuthID string) (db.User, error) {
	logger.Log.Debug("Starting user creation from Web3Auth claims",
		zap.String("web3auth_id", web3AuthID),
		zap.String("email", claims.Email),
		zap.String("name", claims.Name),
	)

	// Create account first (assuming AccountType.MERCHANT for Web3Auth users)
	account, err := services.GetDB().CreateAccount(ctx, db.CreateAccountParams{
		Name:         fmt.Sprintf("%s's Account", claims.Name),
		AccountType:  db.AccountTypeMerchant, // Web3Auth users default to merchant accounts
		BusinessName: pgtype.Text{String: fmt.Sprintf("%s's Business", claims.Name), Valid: claims.Name != ""},
		BusinessType: pgtype.Text{String: "Individual", Valid: true},
	})
	if err != nil {
		logger.Log.Error("Failed to create account",
			zap.Error(err),
			zap.String("name", claims.Name),
		)
		return db.User{}, fmt.Errorf("failed to create account: %w", err)
	}
	logger.Log.Debug("Created account", zap.String("account_id", account.ID.String()))

	// Create workspace for the account
	workspace, err := services.GetDB().CreateWorkspace(ctx, db.CreateWorkspaceParams{
		AccountID:    account.ID,
		Name:         fmt.Sprintf("%s's Workspace", claims.Name),
		Description:  pgtype.Text{String: "Auto-created workspace for Web3Auth user", Valid: true},
		BusinessName: pgtype.Text{String: fmt.Sprintf("%s's Business", claims.Name), Valid: claims.Name != ""},
		BusinessType: pgtype.Text{String: "Individual", Valid: true},
	})
	if err != nil {
		logger.Log.Error("Failed to create workspace",
			zap.Error(err),
			zap.String("account_id", account.ID.String()),
		)
		return db.User{}, fmt.Errorf("failed to create workspace: %w", err)
	}
	logger.Log.Debug("Created workspace", zap.String("workspace_id", workspace.ID.String()))

	// Create user with Web3Auth data
	logger.Log.Debug("Creating user with Web3Auth data",
		zap.String("account_id", account.ID.String()),
		zap.String("email", claims.Email),
		zap.String("name", claims.Name),
		zap.String("web3auth_id", web3AuthID),
		zap.String("verifier", claims.Verifier),
		zap.String("verifier_id", claims.VerifierId),
	)

	user, err := services.GetDB().CreateUser(ctx, db.CreateUserParams{
		AccountID:      account.ID,
		Email:          claims.Email,
		Role:           db.UserRoleDeveloper,                 // Default role for Web3Auth users
		IsAccountOwner: pgtype.Bool{Bool: true, Valid: true}, // First user is account owner
		FirstName:      pgtype.Text{String: claims.Name, Valid: claims.Name != ""},
		LastName:       pgtype.Text{Valid: false},
		AddressLine1:   pgtype.Text{Valid: false},
		AddressLine2:   pgtype.Text{Valid: false},
		City:           pgtype.Text{Valid: false},
		StateRegion:    pgtype.Text{Valid: false},
		PostalCode:     pgtype.Text{Valid: false},
		Country:        pgtype.Text{Valid: false},
		DisplayName:    pgtype.Text{String: claims.Name, Valid: claims.Name != ""},
		PictureUrl:     pgtype.Text{String: claims.ProfileImage, Valid: claims.ProfileImage != ""},
		Phone:          pgtype.Text{Valid: false},
		Timezone:       pgtype.Text{Valid: false},
		Locale:         pgtype.Text{Valid: false},
		EmailVerified:  pgtype.Bool{Bool: true, Valid: true}, // Assume Web3Auth emails are verified
		// Web3Auth specific fields
		Web3authID: pgtype.Text{String: web3AuthID, Valid: true},
		Verifier:   pgtype.Text{String: claims.Verifier, Valid: claims.Verifier != ""},
		VerifierID: pgtype.Text{String: claims.VerifierId, Valid: claims.VerifierId != ""},
		// Set default metadata
		Metadata: []byte("{}"),
	})
	if err != nil {
		logger.Log.Error("Failed to create user",
			zap.Error(err),
			zap.String("account_id", account.ID.String()),
			zap.String("email", claims.Email),
		)
		return db.User{}, fmt.Errorf("failed to create user: %w", err)
	}
	logger.Log.Debug("Created user", zap.String("user_id", user.ID.String()))

	// Create Smart Account wallets from the wallets array
	for _, wallet := range claims.Wallets {
		if wallet.Address != "" {
			err = ac.createSmartAccountWallet(ctx, services.GetDB().(*db.Queries), user, wallet.Address, workspace.ID)
			if err != nil {
				// Log error but don't fail user creation
				logger.Log.Error("Failed to create Smart Account wallet",
					zap.String("user_id", user.ID.String()),
					zap.String("wallet_address", wallet.Address),
					zap.String("wallet_type", wallet.Type),
					zap.Error(err),
				)
			}
		}
	}

	logger.Log.Info("Successfully created Web3Auth user",
		zap.String("user_id", user.ID.String()),
		zap.String("account_id", account.ID.String()),
		zap.String("workspace_id", workspace.ID.String()),
		zap.String("web3auth_id", web3AuthID),
		zap.String("email", claims.Email),
		zap.String("verifier", claims.Verifier),
	)

	return user, nil
}

func (ac *AuthClient) createSmartAccountWallet(ctx context.Context, queries *db.Queries, user db.User, smartAccountAddress string, workspaceID uuid.UUID) error {
	// Create Smart Account wallet using existing wallet creation logic
	metadata := map[string]interface{}{
		"created_via": "web3auth_auto_creation",
		"verifier":    user.Verifier.String,
		"verifier_id": user.VerifierID.String,
		"created_at":  time.Now().UTC().Format(time.RFC3339),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	wallet, err := queries.CreateWallet(ctx, db.CreateWalletParams{
		WorkspaceID:   workspaceID,
		WalletType:    "web3auth",
		WalletAddress: smartAccountAddress,
		NetworkType:   db.NetworkTypeEvm, // Default network
		Nickname:      pgtype.Text{String: "Web3Auth Smart Account", Valid: true},
		IsPrimary:     pgtype.Bool{Bool: true, Valid: true},
		Verified:      pgtype.Bool{Bool: true, Valid: true},
		Metadata:      metadataJSON,
		// NetworkID is optional, set to empty UUID for now
		NetworkID: pgtype.UUID{Valid: false},
	})
	if err != nil {
		return fmt.Errorf("failed to create Smart Account wallet: %w", err)
	}

	logger.Log.Info("Created Smart Account wallet for Web3Auth user",
		zap.String("user_id", user.ID.String()),
		zap.String("wallet_id", wallet.ID.String()),
		zap.String("wallet_address", wallet.WalletAddress),
		zap.String("web3auth_id", user.Web3authID.String),
	)

	return nil
}
