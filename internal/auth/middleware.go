package auth

import (
	"context"
	"cyphera-api/internal/db"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CustomClaims struct {
	Scope string `json:"scope"`
}

// Validate does nothing for now but we can use it to check scopes
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

// EnsureValidToken is a middleware that will check the validity of our JWT
func EnsureValidToken() gin.HandlerFunc {
	issuerURL, err := url.Parse("https://" + os.Getenv("AUTH0_DOMAIN") + "/")
	if err != nil {
		log.Fatalf("Failed to parse the issuer url: %v", err)
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	// Set up the validator
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
		log.Fatalf("Failed to set up the validator: %v", err)
	}

	middleware := jwtmiddleware.New(
		jwtValidator.ValidateToken,
		jwtmiddleware.WithErrorHandler(errorHandler),
	)

	return func(c *gin.Context) {
		encounteredError := true
		var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
			encounteredError = false
			c.Request = r
			c.Next()
		}

		middleware.CheckJWT(handler).ServeHTTP(c.Writer, c.Request)

		if encounteredError {
			c.Abort()
		}
	}
}

func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("Encountered error while validating JWT: %v", err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	if _, err := w.Write([]byte(`{"message":"Invalid token"}`)); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// GetUserIDFromToken extracts the Auth0 user ID from the token
func GetUserIDFromToken(c *gin.Context) (string, error) {
	claims, ok := c.Request.Context().Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	if !ok {
		return "", ErrNoValidatedClaims
	}

	sub := claims.RegisteredClaims.Subject

	return sub, nil
}

// RequireRoles middleware checks if the user has any of the required roles
func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authType := c.GetString("authType")

		switch authType {
		case "api_key":
			userRole := c.GetString("userRole")
			apiKeyLevel := c.GetString("apiKeyLevel")

			// First check user role
			hasRequiredRole := false
			for _, role := range roles {
				if role == userRole {
					hasRequiredRole = true
					break
				}
			}

			if !hasRequiredRole {
				c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient user role permissions"})
				c.Abort()
				return
			}

			// Then check API key level
			switch apiKeyLevel {
			case "read":
				// Read keys can only perform GET operations
				if c.Request.Method != "GET" {
					c.JSON(http.StatusForbidden, gin.H{"error": "Read-only API key cannot perform write operations"})
					c.Abort()
					return
				}
			case "write":
				// Write keys can perform all operations except DELETE
				if c.Request.Method == "DELETE" {
					c.JSON(http.StatusForbidden, gin.H{"error": "Write-level API key cannot perform delete operations"})
					c.Abort()
					return
				}
			case "admin":
				// Admin keys can do everything
				// No additional checks needed
			default:
				c.JSON(http.StatusForbidden, gin.H{"error": "Invalid API key level"})
				c.Abort()
				return
			}

			c.Next()
			return
		case "jwt":
			token := c.GetHeader("Authorization")
			if token == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "No authorization header"})
				c.Abort()
				return
			}

			// Remove 'Bearer ' prefix if it exists and validate
			token = strings.TrimPrefix(token, "Bearer ")
			if token == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
				c.Abort()
				return
			}

			// Here you would validate the token and extract roles
			// For now, we'll implement a simple check
			// In production, you should verify this against your users table

			// TODO: Implement role checking against database
			c.Next()
		default:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication type"})
			c.Abort()
			return
		}
	}
}

// EnsureValidAPIKeyOrToken is a middleware that will check for either a valid API key or JWT token
func EnsureValidAPIKeyOrToken(queries *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check for API key
		apiKey := c.GetHeader("x-api-key")
		if apiKey != "" {
			// Validate API key against database
			key, err := queries.GetAPIKeyByHash(c.Request.Context(), apiKey)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
				c.Abort()
				return
			}

			// Check if API key is active and not expired
			if !key.IsActive.Bool || (key.ExpiresAt.Valid && key.ExpiresAt.Time.Before(time.Now())) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "API key is inactive or expired"})
				c.Abort()
				return
			}

			// Update last used timestamp
			err = queries.UpdateAPIKeyLastUsed(c.Request.Context(), key.ID)
			if err != nil {
				log.Printf("Failed to update API key last used timestamp: %v", err)
				// Don't fail the request for this
			}

			// Get the account and associated user to determine the role
			if !key.AccountID.Valid {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key: no account associated"})
				c.Abort()
				return
			}

			accountUUID, err := uuid.FromBytes(key.AccountID.Bytes[:])
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key: malformed account ID"})
				c.Abort()
				return
			}

			account, err := queries.GetAccount(c.Request.Context(), accountUUID)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key: associated account not found"})
				c.Abort()
				return
			}

			user, err := queries.GetUser(c.Request.Context(), account.UserID)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key: associated user not found"})
				c.Abort()
				return
			}

			// Set context values for downstream handlers
			c.Set("authType", "api_key")
			c.Set("accountID", accountUUID.String())
			c.Set("userRole", string(user.Role))    // User's role (admin/account)
			c.Set("apiKeyLevel", string(key.Level)) // API key level (read/write/admin)
			c.Next()
			return
		}

		// If no API key, check for JWT token
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No authentication provided"})
			c.Abort()
			return
		}

		// Use the existing JWT validation logic
		encounteredError := true
		var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
			encounteredError = false
			c.Request = r
			c.Set("authType", "jwt")
			c.Next()
		}

		issuerURL, err := url.Parse("https://" + os.Getenv("AUTH0_DOMAIN") + "/")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication configuration error"})
			c.Abort()
			return
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication configuration error"})
			c.Abort()
			return
		}

		middleware := jwtmiddleware.New(
			jwtValidator.ValidateToken,
			jwtmiddleware.WithErrorHandler(errorHandler),
		)

		middleware.CheckJWT(handler).ServeHTTP(c.Writer, c.Request)

		if encounteredError {
			c.Abort()
		}
	}
}
