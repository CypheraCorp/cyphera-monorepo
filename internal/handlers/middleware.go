package handlers

import (
	"bytes"
	"cyphera-api/internal/logger"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// Context keys for storing values
const (
	RequestIDKey  = "request_id"
	AccountIDKey  = "X-Account-ID"
	UserIDKey     = "X-User-ID"
	SupabaseIDKey = "supabase_id"
)

// RequestLog represents a structured log entry for an HTTP request
type RequestLog struct {
	Method     string    `json:"method"`
	Path       string    `json:"path,omitempty"`
	Query      string    `json:"query,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	ClientIP   string    `json:"client_ip"`
	RequestID  string    `json:"request_id,omitempty"`
	AccountID  string    `json:"account_id,omitempty"`
	UserID     string    `json:"user_id,omitempty"`
	SupabaseID string    `json:"supabase_id,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// SupabaseClaims represents the expected structure of the Supabase JWT claims
type SupabaseClaims struct {
	Sub         string `json:"sub"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	Aud         string `json:"aud"`
	Exp         int64  `json:"exp"`
	SupabaseRef string `json:"reference_id"`
	jwt.RegisteredClaims
}

// SiweMessage represents a Sign-In with Ethereum message
type SiweMessage struct {
	Domain    string `json:"domain"`
	Address   string `json:"address"`
	Statement string `json:"statement"`
	URI       string `json:"uri"`
	Version   string `json:"version"`
	ChainID   int64  `json:"chainId"`
	Nonce     string `json:"nonce"`
}

// ValidateSupabaseToken validates the Supabase JWT token
func ValidateSupabaseToken(tokenString string) (*SupabaseClaims, error) {
	// Remove "Bearer " prefix if present
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &SupabaseClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			logger.Log.Debug("Token validation failed",
				zap.String("reason", "unexpected signing method"),
				zap.String("algorithm", token.Header["alg"].(string)),
			)
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get JWT secret from environment
		jwtSecret := os.Getenv("SUPABASE_JWT_SECRET")
		if jwtSecret == "" {
			logger.Log.Debug("Token validation failed",
				zap.String("reason", "SUPABASE_JWT_SECRET not set"),
			)
			return nil, fmt.Errorf("SUPABASE_JWT_SECRET not set")
		}

		return []byte(jwtSecret), nil
	})

	if err != nil {
		logger.Log.Debug("Token parsing failed",
			zap.Error(err),
			zap.String("token_prefix", tokenString[:10]+"..."), // Safely log token prefix
		)
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*SupabaseClaims); ok && token.Valid {
		// Check if token is expired
		if time.Unix(claims.Exp, 0).Before(time.Now()) {
			logger.Log.Debug("Token validation failed",
				zap.String("reason", "token expired"),
				zap.Time("expiration", time.Unix(claims.Exp, 0)),
				zap.Time("current_time", time.Now()),
			)
			return nil, fmt.Errorf("token is expired")
		}
		return claims, nil
	}

	logger.Log.Debug("Token validation failed",
		zap.String("reason", "invalid token claims"),
	)
	return nil, fmt.Errorf("invalid token claims")
}

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

// AuthMiddleware handles authentication for both Supabase JWT and API keys
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check for API key
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" {
			logger.Log.Debug("Using API key authentication")
			c.Next()
			return
		}

		// Check for Supabase JWT token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger.Log.Debug("Authentication failed",
				zap.String("reason", "No authentication header provided"),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("client_ip", c.ClientIP()),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "No authentication provided"})
			c.Abort()
			return
		}

		// Validate Supabase token
		claims, err := ValidateSupabaseToken(authHeader)
		if err != nil {
			logger.Log.Debug("Supabase token validation failed",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path),
				zap.String("method", c.Request.Method),
				zap.String("client_ip", c.ClientIP()),
				zap.String("auth_header", strings.Replace(authHeader, "Bearer ", "***.", 1)), // Safely log partial token
			)
			logger.Log.Error("Failed to validate Supabase token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		logger.Log.Debug("Supabase authentication successful",
			zap.String("user_id", claims.Sub),
			zap.String("email", claims.Email),
			zap.String("role", claims.Role),
			zap.String("path", c.Request.URL.Path),
		)

		// Set Supabase user information in context
		c.Set("supabase_id", claims.Sub)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
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
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			Query:      c.Request.URL.RawQuery,
			UserAgent:  c.Request.UserAgent(),
			ClientIP:   c.ClientIP(),
			RequestID:  c.GetHeader(RequestIDKey),  // Use defined constant
			AccountID:  c.GetHeader(AccountIDKey),  // Use defined constant
			UserID:     c.GetHeader(UserIDKey),     // Use defined constant
			SupabaseID: c.GetHeader(SupabaseIDKey), // Use defined constant
			Timestamp:  start.UTC(),
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
			zap.String("supabase_id", requestLog.SupabaseID),
			bodyField, // Add the dynamically created body field here
			zap.Time("timestamp", requestLog.Timestamp),
		)

		c.Next()
	}
}

// GenerateAndSignMessage creates and signs a SIWE message with the given private key
func GenerateAndSignMessage(privateKeyHex string, message *SiweMessage) (msgString string, signature string, err error) {
	if message == nil {
		return "", "", fmt.Errorf("message is nil")
	}

	requiredFields := map[string]string{
		"nonce":     message.Nonce,
		"address":   message.Address,
		"domain":    message.Domain,
		"statement": message.Statement,
		"uri":       message.URI,
	}

	for field, value := range requiredFields {
		if value == "" {
			return "", "", fmt.Errorf("%s is empty", field)
		}
	}

	logger.Log.Debug("Generating and signing SIWE message",
		zap.String("nonce", message.Nonce),
	)

	// Remove 0x prefix if present
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")

	// Parse the private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		logger.Log.Debug("Failed to parse private key",
			zap.Error(err),
		)
		return "", "", fmt.Errorf("invalid private key: %w", err)
	}

	// Prepare message string in SIWE format
	messageString := fmt.Sprintf("%s wants you to sign in with your Ethereum account:\n%s\n\n%s\n\nURI: %s\nVersion: %s\nChain ID: %d\nNonce: %s\nIssued At: %s",
		message.Domain,
		message.Address,
		message.Statement,
		message.URI,
		message.Version,
		message.ChainID,
		message.Nonce,
		time.Now().UTC().Format(time.RFC3339),
	)

	// Create Ethereum specific message hash
	messageHash := accounts.TextHash([]byte(messageString))

	// Sign the hash
	sig, err := crypto.Sign(messageHash, privateKey)
	if err != nil {
		logger.Log.Debug("Failed to sign message",
			zap.Error(err),
		)
		return "", "", fmt.Errorf("failed to sign message: %w", err)
	}

	// Adjust V value in signature (Ethereum's specific requirement)
	sig[crypto.RecoveryIDOffset] += 27

	// Convert signature to hex using go-ethereum's hexutil
	signature = hexutil.Encode(sig)

	err = VerifySignature(message.Address, signature, messageString)
	if err != nil {
		return "", "", fmt.Errorf("failed to verify signature: %w", err)
	}

	logger.Log.Debug("Successfully generated and signed message",
		zap.String("message", messageString),
		zap.String("signature_prefix", signature[:10]+"..."),
	)

	return messageString, signature, nil
}

// VerifySignature verifies an Ethereum signed message
func VerifySignature(fromAddress, signatureHex, message string) error {
	// Decode the hex signature
	signature, err := hexutil.Decode(signatureHex)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Transform V value back from 27/28 to 0/1
	signature[crypto.RecoveryIDOffset] -= 27

	// Create message hash
	messageHash := accounts.TextHash([]byte(message))

	// Recover public key from signature
	pubKey, err := crypto.SigToPub(messageHash, signature)
	if err != nil {
		return fmt.Errorf("failed to recover public key: %w", err)
	}

	// Verify the address matches
	recoveredAddr := crypto.PubkeyToAddress(*pubKey).Hex()
	if !strings.EqualFold(recoveredAddr, fromAddress) {
		return fmt.Errorf("signature is not from the expected address")
	}

	return nil
}
