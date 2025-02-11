package auth

import (
	"cyphera-api/internal/constants"
	"cyphera-api/internal/db"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	// ErrInvalidToken is returned when the provided token is invalid
	ErrInvalidToken = errors.New("invalid token")
)

// SupabaseClaims represents the expected structure of the Supabase JWT claims
type SupabaseClaims struct {
	jwt.RegisteredClaims
	Email        string                 `json:"email"`
	Sub          string                 `json:"sub"` // This is the user ID
	Role         string                 `json:"role"`
	AppMetadata  map[string]interface{} `json:"app_metadata"`
	UserMetadata map[string]interface{} `json:"user_metadata"`
}

// EnsureValidAPIKeyOrToken is a middleware that checks for either a valid API key or JWT token
func EnsureValidAPIKeyOrToken(queries *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check for API key in header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			workspace, account, key, err := validateAPIKey(c, queries, apiKey)
			if err != nil {
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

		// If no API key, check for JWT token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No authentication provided"})
			c.Abort()
			return
		}

		user, account, err := validateJWTToken(c, queries, authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// Get workspaces for the account
		workspaces, err := queries.ListWorkspacesByAccountID(c.Request.Context(), account.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve workspaces"})
			c.Abort()
			return
		}

		if len(workspaces) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No workspaces found for account"})
			c.Abort()
			return
		}

		defaultWorkspace := workspaces[0]

		// Set context with user and account information
		c.Set("userID", user.ID.String())
		c.Set("accountID", account.ID.String())
		c.Set("workspaceID", defaultWorkspace.ID.String())
		c.Set("accountType", string(account.AccountType))
		c.Set("userRole", string(user.Role))
		c.Set("authType", constants.AuthTypeJWT)
		c.Next()
	}
}

// validateAPIKey validates the API key and returns workspace and account information
func validateAPIKey(c *gin.Context, queries *db.Queries, apiKey string) (db.Workspace, db.Account, db.ApiKey, error) {
	// Validate API key
	key, err := queries.GetAPIKeyByKey(c.Request.Context(), apiKey)
	if err != nil {
		return db.Workspace{}, db.Account{}, db.ApiKey{}, fmt.Errorf("invalid API key")
	}

	// Check if API key is expired
	if key.ExpiresAt.Valid && key.ExpiresAt.Time.Before(time.Now()) {
		return db.Workspace{}, db.Account{}, db.ApiKey{}, fmt.Errorf("API key has expired")
	}

	// Get workspace associated with API key
	workspace, err := queries.GetWorkspace(c.Request.Context(), key.WorkspaceID)
	if err != nil {
		return db.Workspace{}, db.Account{}, db.ApiKey{}, fmt.Errorf("invalid workspace")
	}

	// Get account associated with workspace
	account, err := queries.GetAccount(c.Request.Context(), workspace.AccountID)
	if err != nil {
		return db.Workspace{}, db.Account{}, db.ApiKey{}, fmt.Errorf("invalid account")
	}

	return workspace, account, key, nil
}

// validateJWTToken validates the Supabase JWT token and returns user information
func validateJWTToken(c *gin.Context, queries *db.Queries, authHeader string) (db.User, db.Account, error) {
	claims, err := validateSupabaseToken(authHeader)
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

func validateSupabaseToken(tokenString string) (*SupabaseClaims, error) {
	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// Get JWT secret from environment
	jwtSecret := os.Getenv("SUPABASE_JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("SUPABASE_JWT_SECRET not set")
	}

	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &SupabaseClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method - Supabase uses HS256
		if token.Method.Alg() != "HS256" {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(jwtSecret), nil
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

// RequireRoles is a middleware that checks if the user has the required roles
func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountType := c.GetString("accountType")
		apiKeyLevel := c.GetString("apiKeyLevel")
		authType := c.GetString("authType")

		// For API key auth, check access level
		if authType == constants.AuthTypeAPIKey {
			if apiKeyLevel != constants.AccessLevelAdmin {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient API key access level"})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// For admin-only operations, check account type
		if roles[0] == constants.RoleAdmin && accountType != constants.AccountTypeAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}
