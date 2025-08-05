package middleware

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Common validation configurations for different endpoints

// Product validation rules
var CreateProductValidation = ValidationConfig{
	MaxBodySize:        1024 * 1024, // 1MB
	AllowUnknownFields: false,       // Strict validation - reject unknown fields
	Rules: []ValidationRule{
		{
			Field:     "name",
			Type:      "string",
			Required:  true,
			MinLength: 1,
			MaxLength: 100,
			Sanitize:  true,
		},
		{
			Field:     "description",
			Type:      "string",
			Required:  false,
			MaxLength: 500,
			Sanitize:  true,
		},
		{
			Field:    "wallet_id",
			Type:     "uuid",
			Required: true,
		},
		{
			Field:    "active",
			Type:     "boolean",
			Required: true, // Changed to required to match frontend schema
		},
		{
			Field:    "image_url",
			Type:     "string",
			Required: false,
			Custom: func(value interface{}) error {
				// Allow empty string or valid URL
				str, ok := value.(string)
				if !ok {
					return fmt.Errorf("must be a string")
				}
				if str == "" {
					return nil // Empty string is allowed
				}
				// Validate URL format
				if !URLRegex.MatchString(str) {
					return fmt.Errorf("must be a valid URL")
				}
				return nil
			},
		},
		{
			Field:    "url",
			Type:     "string",
			Required: false,
			Custom: func(value interface{}) error {
				// Allow empty string or valid URL
				str, ok := value.(string)
				if !ok {
					return fmt.Errorf("must be a string")
				}
				if str == "" {
					return nil // Empty string is allowed
				}
				// Validate URL format
				if !URLRegex.MatchString(str) {
					return fmt.Errorf("must be a valid URL")
				}
				return nil
			},
		},
		{
			Field:    "metadata",
			Type:     "object",
			Required: false,
		},
		// Embedded price fields (required since prices table was merged into products)
		{
			Field:    "price_type",
			Type:     "string",
			Required: true,
			Custom: func(value interface{}) error {
				str, ok := value.(string)
				if !ok {
					return fmt.Errorf("must be a string")
				}
				if str != "one_time" && str != "recurring" {
					return fmt.Errorf("must be either 'one_time' or 'recurring'")
				}
				return nil
			},
		},
		{
			Field:    "currency",
			Type:     "string",
			Required: true,
			Custom: func(value interface{}) error {
				str, ok := value.(string)
				if !ok {
					return fmt.Errorf("must be a string")
				}
				if len(str) != 3 {
					return fmt.Errorf("must be a 3-character currency code (e.g. USD)")
				}
				return nil
			},
		},
		{
			Field:    "unit_amount_in_pennies",
			Type:     "number",
			Required: true,
			Custom: func(value interface{}) error {
				// Handle both int and float64 from JSON
				var amount float64
				switch v := value.(type) {
				case float64:
					amount = v
				case int:
					amount = float64(v)
				default:
					return fmt.Errorf("must be a number")
				}
				if amount <= 0 {
					return fmt.Errorf("must be positive")
				}
				return nil
			},
		},
		{
			Field:    "interval_type",
			Type:     "string",
			Required: false, // Only required for recurring prices
			Custom: func(value interface{}) error {
				if value == nil {
					return nil // Optional field
				}
				str, ok := value.(string)
				if !ok {
					return fmt.Errorf("must be a string")
				}
				validIntervals := []string{"1min", "5mins", "daily", "week", "month", "year"}
				for _, valid := range validIntervals {
					if str == valid {
						return nil
					}
				}
				return fmt.Errorf("must be one of: %s", strings.Join(validIntervals, ", "))
			},
		},
		{
			Field:    "term_length",
			Type:     "number",
			Required: false,
			Custom: func(value interface{}) error {
				if value == nil {
					return nil // Optional field
				}
				// Handle both int and float64 from JSON
				var length float64
				switch v := value.(type) {
				case float64:
					length = v
				case int:
					length = float64(v)
				default:
					return fmt.Errorf("must be a number")
				}
				if length <= 0 {
					return fmt.Errorf("must be positive")
				}
				return nil
			},
		},
		{
			Field:     "price_nickname",
			Type:      "string",
			Required:  false,
			MaxLength: 100,
		},
		{
			Field:    "product_tokens",
			Type:     "array",
			Required: false,
		},
	},
}

// Customer validation rules
var CreateCustomerValidation = ValidationConfig{
	MaxBodySize: 512 * 1024, // 512KB
	Rules: []ValidationRule{
		EmailValidation,
		{
			Field:     "name",
			Type:      "string",
			Required:  true,
			MinLength: 1,
			MaxLength: 255,
			Pattern:   `^[a-zA-Z0-9\s\-'.]+$`,
			Sanitize:  true,
		},
		{
			Field:    "phone",
			Type:     "string",
			Required: false,
			Pattern:  `^\+?[1-9]\d{1,14}$`,
			Sanitize: true,
		},
		{
			Field:    "metadata",
			Type:     "object",
			Required: false,
		},
	},
}

// Wallet validation rules
var CreateWalletValidation = ValidationConfig{
	MaxBodySize: 256 * 1024, // 256KB
	Rules: []ValidationRule{
		{
			Field:         "wallet_type",
			Type:          "string",
			Required:      true,
			AllowedValues: []string{"wallet", "circle", "web3auth"},
		},
		{
			Field:         "network_type",
			Type:          "string",
			Required:      true,
			AllowedValues: []string{"evm", "solana"},
		},
		{
			Field:    "wallet_address",
			Type:     "string",
			Required: true,
			Custom: func(value interface{}) error {
				address, ok := value.(string)
				if !ok {
					return fmt.Errorf("must be a string")
				}
				// Basic Ethereum address validation
				if matched, _ := regexp.MatchString(`^0x[a-fA-F0-9]{40}$`, address); !matched {
					// Check if it might be a Solana address (base58, 32-44 chars)
					if matched, _ := regexp.MatchString(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`, address); !matched {
						return fmt.Errorf("invalid wallet address format")
					}
				}
				return nil
			},
		},
		{
			Field:     "nickname",
			Type:      "string",
			Required:  false,
			MaxLength: 100,
			Sanitize:  true,
		},
	},
}

// Transaction validation rules
var CreateTransactionValidation = ValidationConfig{
	MaxBodySize: 256 * 1024, // 256KB
	Rules: []ValidationRule{
		{
			Field:    "from_wallet_id",
			Type:     "uuid",
			Required: true,
		},
		{
			Field:    "to_address",
			Type:     "string",
			Required: true,
			Custom: func(value interface{}) error {
				address, ok := value.(string)
				if !ok {
					return fmt.Errorf("must be a string")
				}
				// Basic address validation
				if len(address) < 20 || len(address) > 100 {
					return fmt.Errorf("invalid address length")
				}
				return nil
			},
		},
		{
			Field:    "amount",
			Type:     "string", // String to handle precision
			Required: true,
			Pattern:  `^\d+(\.\d+)?$`,
		},
		{
			Field:    "token_address",
			Type:     "string",
			Required: true,
		},
		{
			Field:    "network_id",
			Type:     "uuid",
			Required: true,
		},
	},
}

// API Key validation rules
var CreateAPIKeyValidation = ValidationConfig{
	MaxBodySize: 64 * 1024, // 64KB
	Rules: []ValidationRule{
		{
			Field:     "name",
			Type:      "string",
			Required:  true,
			MinLength: 1,
			MaxLength: 100,
			Sanitize:  true,
		},
		{
			Field:     "description",
			Type:      "string",
			Required:  false,
			MinLength: 0,
			MaxLength: 200,
			Sanitize:  true,
		},
		{
			Field:         "access_level",
			Type:          "string",
			Required:      true,
			AllowedValues: []string{"read", "write", "admin"},
		},
		{
			Field:    "expires_at",
			Type:     "string",
			Required: false,
			Pattern:  `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(Z|[+-]\d{2}:\d{2})$`, // ISO 8601
		},
	},
}

// Subscription validation rules
var CreateSubscriptionValidation = ValidationConfig{
	MaxBodySize: 256 * 1024, // 256KB
	Rules: []ValidationRule{
		{
			Field:    "product_id",
			Type:     "uuid",
			Required: true,
		},
		{
			Field:    "customer_id",
			Type:     "uuid",
			Required: true,
		},
		{
			Field:         "payment_method",
			Type:          "string",
			Required:      true,
			AllowedValues: []string{"crypto", "delegation"},
		},
		{
			Field:    "metadata",
			Type:     "object",
			Required: false,
		},
	},
}

// Subscription validation rules for delegation-based subscription
var CreateDelegationSubscriptionValidation = ValidationConfig{
	MaxBodySize: 256 * 1024, // 256KB
	Rules: []ValidationRule{
		{
			Field:    "product_id",
			Type:     "uuid",
			Required: true,
		},
		{
			Field:    "subscriber_address",
			Type:     "string",
			Required: true,
			Pattern:  `^0x[a-fA-F0-9]{40}$`, // Ethereum address format
		},
		{
			Field:    "product_token_id",
			Type:     "uuid",
			Required: true,
		},
		{
			Field:    "token_amount",
			Type:     "string",
			Required: true,
			Pattern:  `^\d+$`, // Must be a positive integer string
		},
		{
			Field:    "delegation",
			Type:     "object",
			Required: true,
			Custom: func(value interface{}) error {
				delegation, ok := value.(map[string]interface{})
				if !ok {
					return fmt.Errorf("delegation must be an object")
				}

				// Check required delegation fields
				requiredFields := []string{"delegate", "delegator", "authority", "signature"}
				for _, field := range requiredFields {
					if _, exists := delegation[field]; !exists {
						return fmt.Errorf("delegation.%s is required", field)
					}
				}

				return nil
			},
		},
	},
}

// User validation rules
var UpdateUserValidation = ValidationConfig{
	MaxBodySize:        512 * 1024, // 512KB
	AllowUnknownFields: false,
	Rules: []ValidationRule{
		{
			Field:     "first_name",
			Type:      "string",
			Required:  false,
			MaxLength: 100,
			Sanitize:  true,
		},
		{
			Field:     "last_name",
			Type:      "string",
			Required:  false,
			MaxLength: 100,
			Sanitize:  true,
		},
		{
			Field:     "display_name",
			Type:      "string",
			Required:  false,
			MaxLength: 100,
			Sanitize:  true,
		},
		{
			Field:    "email",
			Type:     "email",
			Required: false,
			Sanitize: true,
		},
		{
			Field:    "phone",
			Type:     "string",
			Required: false,
			Pattern:  `^\+?[1-9]\d{1,14}$`,
			Sanitize: true,
		},
		{
			Field:     "timezone",
			Type:      "string",
			Required:  false,
			MaxLength: 50,
			Custom: func(value interface{}) error {
				tz, ok := value.(string)
				if !ok {
					return fmt.Errorf("must be a string")
				}
				// Basic timezone validation (e.g., "America/New_York")
				if !strings.Contains(tz, "/") || len(tz) < 3 {
					return fmt.Errorf("invalid timezone format")
				}
				return nil
			},
		},
	},
}

// Search/Filter validation
var ListQueryValidation = ValidationConfig{
	Rules: []ValidationRule{
		{
			Field:    "page",
			Type:     "number",
			Required: false,
			Min:      float64Ptr(1),
			Max:      float64Ptr(1000),
		},
		{
			Field:    "limit",
			Type:     "number",
			Required: false,
			Min:      float64Ptr(1),
			Max:      float64Ptr(100),
		},
		{
			Field:     "search",
			Type:      "string",
			Required:  false,
			MaxLength: 255,
			Sanitize:  true,
		},
		{
			Field:         "sort",
			Type:          "string",
			Required:      false,
			AllowedValues: []string{"created_at", "updated_at", "name", "-created_at", "-updated_at", "-name"},
		},
		{
			Field:         "status",
			Type:          "string",
			Required:      false,
			AllowedValues: []string{"active", "inactive", "pending", "cancelled"},
		},
		{
			Field:         "include_circle_data",
			Type:          "string",
			Required:      false,
			AllowedValues: []string{"true", "false"},
		},
		{
			Field:         "wallet_type",
			Type:          "string",
			Required:      false,
			AllowedValues: []string{"wallet", "circle_wallet"},
		},
	},
}

// Circle wallet validation
var CircleWalletValidation = ValidationConfig{
	MaxBodySize: 256 * 1024, // 256KB
	Rules: []ValidationRule{
		{
			Field:         "blockchain",
			Type:          "string",
			Required:      true,
			AllowedValues: []string{"ETH", "MATIC", "AVAX"},
		},
		{
			Field:     "user_token",
			Type:      "string",
			Required:  true,
			MinLength: 10,
			MaxLength: 2000, // JWT tokens can be long
		},
		{
			Field:     "idempotency_key",
			Type:      "string",
			Required:  true,
			MinLength: 1,
			MaxLength: 255,
		},
	},
}

// Address validation helper
func ValidateBlockchainAddress(blockchain string) func(interface{}) error {
	return func(value interface{}) error {
		address, ok := value.(string)
		if !ok {
			return fmt.Errorf("must be a string")
		}

		switch strings.ToUpper(blockchain) {
		case "ETH", "MATIC", "AVAX", "EVM":
			// Ethereum-style address
			if matched, _ := regexp.MatchString(`^0x[a-fA-F0-9]{40}$`, address); !matched {
				return fmt.Errorf("invalid Ethereum address format")
			}
		case "SOL", "SOLANA":
			// Solana address
			if matched, _ := regexp.MatchString(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`, address); !matched {
				return fmt.Errorf("invalid Solana address format")
			}
		default:
			return fmt.Errorf("unsupported blockchain type")
		}

		return nil
	}
}

// Invoice validation rules
var CreateInvoiceValidation = ValidationConfig{
	MaxBodySize: 1024 * 1024, // 1MB for invoices with line items
	Rules: []ValidationRule{
		{
			Field:    "customer_id",
			Type:     "uuid",
			Required: true,
		},
		{
			Field:    "subscription_id",
			Type:     "uuid",
			Required: false,
		},
		{
			Field:     "currency",
			Type:      "string",
			Required:  true,
			MinLength: 3,
			MaxLength: 3,
			Pattern:   `^[A-Z]{3}$`,
		},
		{
			Field:    "due_date",
			Type:     "string",
			Required: false,
			Pattern:  `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(Z|[+-]\d{2}:\d{2})$`, // ISO 8601
		},
		{
			Field:     "discount_code",
			Type:      "string",
			Required:  false,
			MaxLength: 50,
		},
		{
			Field:    "line_items",
			Type:     "array",
			Required: true,
			Custom: func(value interface{}) error {
				// Custom validation for line items array
				items, ok := value.([]interface{})
				if !ok {
					return fmt.Errorf("line_items must be an array")
				}
				if len(items) < 1 {
					return fmt.Errorf("at least one line item is required")
				}
				if len(items) > 100 {
					return fmt.Errorf("maximum 100 line items allowed")
				}
				return nil
			},
		},
		{
			Field:    "metadata",
			Type:     "object",
			Required: false,
		},
	},
}

// ValidateQueryParams creates validation for URL query parameters
func ValidateQueryParams(config ValidationConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add debug logging
		logger.Log.Info("=== QUERY PARAM VALIDATION START ===",
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("correlation_id", c.GetHeader("X-Correlation-ID")),
		)

		// Parse query parameters into a map
		params := make(map[string]interface{})
		for key, values := range c.Request.URL.Query() {
			if len(values) > 0 {
				// Try to parse as number if it looks like one
				if num, err := strconv.ParseFloat(values[0], 64); err == nil {
					params[key] = num
					logger.Log.Info("Query param parsed as number", zap.String("key", key), zap.Float64("value", num))
				} else if values[0] == "true" || values[0] == "false" {
					// KEEP AS STRING for validation rules that expect string type
					params[key] = values[0]
					logger.Log.Info("Query param kept as string", zap.String("key", key), zap.String("value", values[0]))
				} else {
					params[key] = values[0]
					logger.Log.Info("Query param as string", zap.String("key", key), zap.String("value", values[0]))
				}
			}
		}

		logger.Log.Info("Parsed query params", zap.Any("params", params))

		// Validate fields
		errors := validateFields(params, config.Rules, config.AllowUnknownFields)
		if len(errors) > 0 {
			logger.Log.Error("Query validation failed",
				zap.Any("errors", errors),
				zap.Any("params", params),
				zap.String("correlation_id", c.GetHeader("X-Correlation-ID")),
			)
			c.JSON(http.StatusBadRequest, ValidationErrors{Errors: errors})
			c.Abort()
			return
		}

		logger.Log.Info("=== QUERY PARAM VALIDATION PASSED ===",
			zap.String("correlation_id", c.GetHeader("X-Correlation-ID")),
		)

		// Store validated params in context
		c.Set("validatedQuery", params)
		c.Next()
	}
}
