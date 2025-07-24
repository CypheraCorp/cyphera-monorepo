package middleware

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Common validation configurations for different endpoints

// Product validation rules
var CreateProductValidation = ValidationConfig{
	MaxBodySize: 1024 * 1024, // 1MB
	Rules: []ValidationRule{
		{
			Field:     "name",
			Type:      "string",
			Required:  true,
			MinLength: 1,
			MaxLength: 255,
			Sanitize:  true,
		},
		{
			Field:     "description",
			Type:      "string",
			Required:  false,
			MaxLength: 1000,
			Sanitize:  true,
		},
		{
			Field:         "type",
			Type:          "string",
			Required:      true,
			AllowedValues: []string{"subscription", "one_time"},
		},
		{
			Field:    "currency",
			Type:     "string",
			Required: true,
			Pattern:  `^[A-Z]{3}$`, // ISO 4217 currency code
		},
		{
			Field:    "amount",
			Type:     "number",
			Required: true,
			Min:      float64Ptr(0),
			Max:      float64Ptr(1000000), // Max 1 million
		},
		{
			Field:         "billing_period",
			Type:          "string",
			Required:      false,
			AllowedValues: []string{"daily", "weekly", "monthly", "yearly"},
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
			AllowedValues: []string{"metamask", "circle", "web3auth"},
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
			Field:    "price_id",
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

// ValidateQueryParams creates validation for URL query parameters
func ValidateQueryParams(config ValidationConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse query parameters into a map
		params := make(map[string]interface{})
		for key, values := range c.Request.URL.Query() {
			if len(values) > 0 {
				// Try to parse as number if it looks like one
				if num, err := strconv.ParseFloat(values[0], 64); err == nil {
					params[key] = num
				} else if values[0] == "true" || values[0] == "false" {
					params[key] = values[0] == "true"
				} else {
					params[key] = values[0]
				}
			}
		}

		// Validate fields
		errors := validateFields(params, config.Rules, config.AllowUnknownFields)
		if len(errors) > 0 {
			c.JSON(http.StatusBadRequest, ValidationErrors{Errors: errors})
			c.Abort()
			return
		}

		// Store validated params in context
		c.Set("validatedQuery", params)
		c.Next()
	}
}
