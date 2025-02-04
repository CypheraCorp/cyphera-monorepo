package auth

import (
	"context"
	"cyphera-api/internal/constants"
	"cyphera-api/internal/db"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-gonic/gin"
)

var (
	// ErrInvalidToken is returned when the provided token is invalid
	ErrInvalidToken = errors.New("invalid token")
	// jwtValidator is a singleton instance of the JWT validator
	jwtValidator *validator.Validator
)

// CustomClaims contains custom data we want from the token
type CustomClaims struct {
	Scope string `json:"scope"`
}

// Validate implements the validator.CustomClaims interface
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

// TokenClaims represents the expected structure of the JWT claims
type TokenClaims struct {
	Subject string `json:"sub"`
	Issuer  string `json:"iss"`
}

// validateAPIKey validates the API key and returns workspace and account information
// It checks if the key exists, is not expired, and retrieves associated workspace and account
func validateAPIKey(c *gin.Context, queries *db.Queries, apiKey string) (db.Workspace, db.Account, db.ApiKey, error) {
	// Validate API key
	fmt.Println("apiKey what is it", apiKey)
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

// setupAuth initializes the JWT validator with Auth0 configuration
func setupAuth() (*validator.Validator, error) {
	if jwtValidator != nil {
		return jwtValidator, nil
	}

	issuerURL, err := url.Parse("https://" + os.Getenv("AUTH0_DOMAIN") + "/")
	if err != nil {
		log.Fatalf("Failed to parse the issuer url: %v", err)
	}
	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{os.Getenv("AUTH0_AUDIENCE")},
		validator.WithCustomClaims(
			func() validator.CustomClaims {
				return &CustomClaims{}
			},
		),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set up validator: %w", err)
	}
	return jwtValidator, nil
}

// validateJWTToken validates the JWT token and returns user information
func validateJWTToken(c *gin.Context, queries *db.Queries, authHeader string) (db.User, db.Account, error) {
	// Remove "Bearer " prefix
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return db.User{}, db.Account{}, ErrInvalidToken
	}

	// Get or setup validator
	v, err := setupAuth()
	if err != nil {
		log.Printf("Auth setup failed: %v", err)
		return db.User{}, db.Account{}, fmt.Errorf("auth setup failed: %w", err)
	}

	// Validate token and get claims
	claims, err := v.ValidateToken(c.Request.Context(), token)
	if err != nil {
		log.Printf("Token validation failed: %v", err)
		return db.User{}, db.Account{}, ErrInvalidToken
	}

	validatedClaims, ok := claims.(*validator.ValidatedClaims)
	if !ok {
		return db.User{}, db.Account{}, fmt.Errorf("invalid claims type")
	}

	// Get user by Auth0 ID (sub claim)
	user, err := queries.GetUserByAuth0ID(c.Request.Context(), validatedClaims.RegisteredClaims.Subject)
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

// EnsureValidAPIKeyOrToken is a middleware that checks for either a valid API key or JWT token
// It first checks for an API key in the X-API-Key header, then falls back to JWT token validation
// Sets various context values based on the authentication method used
func EnsureValidAPIKeyOrToken(queries *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check for API key in header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			log.Println("API key found ", apiKey)
			workspace, account, key, err := validateAPIKey(c, queries, apiKey)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				c.Abort()
				return
			}

			// Set context with workspace and account information
			c.Set("workspaceID", workspace.ID.String())
			c.Set("accountID", workspace.AccountID.String())
			c.Set("accountType", string(account.AccountType))
			c.Set("apiKeyLevel", string(key.AccessLevel))
			c.Set("authType", constants.AuthTypeAPIKey)
			c.Next()
			return
		}

		// If no API key, check for JWT token
		jwtToken := c.GetHeader("Authorization")
		if jwtToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No authentication provided"})
			c.Abort()
			return
		}

		user, account, err := validateJWTToken(c, queries, jwtToken)
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

// RequireRoles is a middleware that checks if the user has the required roles
// For API key auth, it checks the access level
// For JWT auth, it checks the account type for admin operations
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

// CustomClaims contains custom data we want from the token.
// type CustomClaims struct {
// 	Scope string `json:"scope"`
// }

// // Validate does nothing for this example, but we need
// // it to satisfy validator.CustomClaims interface.
// func (c CustomClaims) Validate(ctx context.Context) error {
// 	return nil
// }
