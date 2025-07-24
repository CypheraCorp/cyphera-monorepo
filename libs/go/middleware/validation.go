package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ValidationRule defines a single validation rule
type ValidationRule struct {
	Field         string                  // Field name to validate
	Required      bool                    // Whether the field is required
	Type          string                  // Expected type: string, number, boolean, uuid, email, etc.
	MinLength     int                     // Minimum length for strings
	MaxLength     int                     // Maximum length for strings
	Pattern       string                  // Regex pattern for validation
	Min           *float64                // Minimum value for numbers
	Max           *float64                // Maximum value for numbers
	AllowedValues []string                // List of allowed values
	Sanitize      bool                    // Whether to sanitize the input
	Custom        func(interface{}) error // Custom validation function
}

// ValidationConfig holds validation rules for an endpoint
type ValidationConfig struct {
	Rules              []ValidationRule
	MaxBodySize        int64 // Maximum request body size in bytes
	AllowUnknownFields bool  // Whether to allow fields not in rules
}

// Common regex patterns
var (
	EmailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	AlphanumRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	PhoneRegex    = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	URLRegex      = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	// Safe characters for most text inputs (prevents XSS)
	SafeTextRegex = regexp.MustCompile(`^[a-zA-Z0-9\s\-_.,!?'"@#$%&*()+=/:;]+$`)
)

// Validation errors
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// ValidateInput creates a validation middleware with the given configuration
func ValidateInput(config ValidationConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get correlation ID for logging
		correlationID, _ := c.Get("correlation_id")
		correlationIDStr, _ := correlationID.(string)

		// Log validation start
		logger.Log.Info("Starting validation middleware",
			zap.String("correlation_id", correlationIDStr),
			zap.String("endpoint", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
			zap.Int("rules_count", len(config.Rules)),
			zap.Bool("allow_unknown_fields", config.AllowUnknownFields),
		)

		// Check request size
		if config.MaxBodySize > 0 && c.Request.ContentLength > config.MaxBodySize {
			logger.Log.Warn("Request body too large",
				zap.String("correlation_id", correlationIDStr),
				zap.Int64("content_length", c.Request.ContentLength),
				zap.Int64("max_size", config.MaxBodySize),
			)
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("Request body too large. Maximum size: %d bytes", config.MaxBodySize),
			})
			c.Abort()
			return
		}

		// Parse request body
		var body map[string]interface{}
		if err := c.ShouldBindJSON(&body); err != nil {
			logger.Log.Error("Failed to parse JSON body",
				zap.String("correlation_id", correlationIDStr),
				zap.Error(err),
			)
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid JSON in request body",
			})
			c.Abort()
			return
		}

		// Log parsed body (excluding sensitive fields)
		logBody := make(map[string]interface{})
		for k, v := range body {
			if k != "password" && k != "api_key" && k != "secret" {
				logBody[k] = v
			}
		}
		logger.Log.Debug("Parsed request body",
			zap.String("correlation_id", correlationIDStr),
			zap.Any("body", logBody),
		)

		// Validate fields
		errors := validateFields(body, config.Rules, config.AllowUnknownFields)
		if len(errors) > 0 {
			logger.Log.Info("Validation failed",
				zap.String("correlation_id", correlationIDStr),
				zap.Any("errors", errors),
				zap.Any("request_body", logBody),
			)
			c.JSON(http.StatusBadRequest, ValidationErrors{Errors: errors})
			c.Abort()
			return
		}

		logger.Log.Info("Validation successful",
			zap.String("correlation_id", correlationIDStr),
		)

		// Store validated body back to context for handler use
		bodyBytes, _ := json.Marshal(body)
		c.Set("validatedBody", body)
		c.Request.Body = NewBodyReader(bodyBytes)

		c.Next()
	}
}

// validateFields validates the fields according to the rules
func validateFields(data map[string]interface{}, rules []ValidationRule, allowUnknown bool) []ValidationError {
	var errors []ValidationError
	validatedFields := make(map[string]bool)

	logger.Log.Debug("Starting field validation",
		zap.Int("field_count", len(data)),
		zap.Int("rule_count", len(rules)),
		zap.Bool("allow_unknown", allowUnknown),
	)

	// Check each rule
	for _, rule := range rules {
		validatedFields[rule.Field] = true
		value, exists := data[rule.Field]

		logger.Log.Debug("Validating field",
			zap.String("field", rule.Field),
			zap.Bool("exists", exists),
			zap.Bool("required", rule.Required),
			zap.String("type", rule.Type),
			zap.Any("value", value),
		)

		// Check required fields
		if rule.Required && (!exists || value == nil || value == "") {
			logger.Log.Debug("Required field missing or empty",
				zap.String("field", rule.Field),
				zap.Bool("exists", exists),
				zap.Any("value", value),
			)
			errors = append(errors, ValidationError{
				Field:   rule.Field,
				Message: fmt.Sprintf("%s is required", rule.Field),
			})
			continue
		}

		// Skip validation if field doesn't exist and not required
		if !exists || value == nil {
			logger.Log.Debug("Skipping non-existent optional field",
				zap.String("field", rule.Field),
			)
			continue
		}

		// Validate based on type
		switch rule.Type {
		case "string":
			err := validateString(value, rule)
			if err != nil {
				logger.Log.Debug("String validation failed",
					zap.String("field", rule.Field),
					zap.Error(err),
				)
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: err.Error(),
				})
			} else if rule.Sanitize {
				// Sanitize if validation passed
				if strVal, ok := value.(string); ok {
					data[rule.Field] = sanitizeString(strVal)
				}
			}

		case "number", "int", "float":
			if err := validateNumber(value, rule); err != nil {
				logger.Log.Debug("Number validation failed",
					zap.String("field", rule.Field),
					zap.Error(err),
				)
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: err.Error(),
				})
			}

		case "boolean", "bool":
			if _, ok := value.(bool); !ok {
				logger.Log.Debug("Boolean validation failed",
					zap.String("field", rule.Field),
					zap.Any("value", value),
				)
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: "must be a boolean",
				})
			}

		case "uuid":
			if err := validateUUID(value); err != nil {
				logger.Log.Debug("UUID validation failed",
					zap.String("field", rule.Field),
					zap.Error(err),
				)
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: err.Error(),
				})
			}

		case "email":
			if err := validateEmail(value); err != nil {
				logger.Log.Debug("Email validation failed",
					zap.String("field", rule.Field),
					zap.Error(err),
				)
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: err.Error(),
				})
			}

		case "url":
			if err := validateURL(value); err != nil {
				logger.Log.Debug("URL validation failed",
					zap.String("field", rule.Field),
					zap.Error(err),
				)
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: err.Error(),
				})
			}

		case "array":
			if _, ok := value.([]interface{}); !ok {
				logger.Log.Debug("Array validation failed",
					zap.String("field", rule.Field),
					zap.Any("value_type", fmt.Sprintf("%T", value)),
				)
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: "must be an array",
				})
			}

		case "object":
			if _, ok := value.(map[string]interface{}); !ok {
				logger.Log.Debug("Object validation failed",
					zap.String("field", rule.Field),
					zap.Any("value_type", fmt.Sprintf("%T", value)),
				)
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: "must be an object",
				})
			}
		}

		// Custom validation
		if rule.Custom != nil {
			if err := rule.Custom(value); err != nil {
				logger.Log.Debug("Custom validation failed",
					zap.String("field", rule.Field),
					zap.Error(err),
				)
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: err.Error(),
				})
			}
		}
	}

	// Check for unknown fields
	if !allowUnknown {
		for field := range data {
			if !validatedFields[field] {
				logger.Log.Debug("Unknown field detected",
					zap.String("field", field),
				)
				errors = append(errors, ValidationError{
					Field:   field,
					Message: "unknown field",
				})
			}
		}
	}

	logger.Log.Debug("Field validation complete",
		zap.Int("error_count", len(errors)),
	)

	return errors
}

// String validation
func validateString(value interface{}, rule ValidationRule) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	// Check length
	length := utf8.RuneCountInString(str)
	if rule.MinLength > 0 && length < rule.MinLength {
		return fmt.Errorf("must be at least %d characters long", rule.MinLength)
	}
	if rule.MaxLength > 0 && length > rule.MaxLength {
		return fmt.Errorf("must be at most %d characters long", rule.MaxLength)
	}

	// Check pattern
	if rule.Pattern != "" {
		regex, err := regexp.Compile(rule.Pattern)
		if err != nil {
			logger.Log.Error("Invalid regex pattern", zap.String("pattern", rule.Pattern), zap.Error(err))
			return fmt.Errorf("invalid validation pattern")
		}
		if !regex.MatchString(str) {
			return fmt.Errorf("invalid format")
		}
	}

	// Check allowed values
	if len(rule.AllowedValues) > 0 {
		allowed := false
		for _, v := range rule.AllowedValues {
			if str == v {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("must be one of: %s", strings.Join(rule.AllowedValues, ", "))
		}
	}

	return nil
}

// Number validation
func validateNumber(value interface{}, rule ValidationRule) error {
	var num float64
	switch v := value.(type) {
	case float64:
		num = v
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	default:
		return fmt.Errorf("must be a number")
	}

	if rule.Min != nil && num < *rule.Min {
		return fmt.Errorf("must be at least %v", *rule.Min)
	}
	if rule.Max != nil && num > *rule.Max {
		return fmt.Errorf("must be at most %v", *rule.Max)
	}

	return nil
}

// UUID validation
func validateUUID(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	if _, err := uuid.Parse(str); err != nil {
		return fmt.Errorf("must be a valid UUID")
	}

	return nil
}

// Email validation
func validateEmail(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	if !EmailRegex.MatchString(str) {
		return fmt.Errorf("must be a valid email address")
	}

	return nil
}

// URL validation
func validateURL(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	if !URLRegex.MatchString(str) {
		return fmt.Errorf("must be a valid URL")
	}

	return nil
}

// sanitizeString removes potentially dangerous characters from strings
func sanitizeString(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Trim whitespace
	input = strings.TrimSpace(input)

	// Basic XSS prevention - encode HTML special characters
	replacements := map[string]string{
		"<":  "&lt;",
		">":  "&gt;",
		"&":  "&amp;",
		"\"": "&quot;",
		"'":  "&#x27;",
		"/":  "&#x2F;",
	}

	for old, new := range replacements {
		input = strings.ReplaceAll(input, old, new)
	}

	return input
}

// Common validation configurations
var (
	// IDValidation for UUID parameters
	IDValidation = ValidationRule{
		Type:     "uuid",
		Required: true,
	}

	// EmailValidation for email fields
	EmailValidation = ValidationRule{
		Type:      "email",
		Required:  true,
		MaxLength: 255,
		Sanitize:  true,
	}

	// NameValidation for name fields
	NameValidation = ValidationRule{
		Type:      "string",
		Required:  true,
		MinLength: 1,
		MaxLength: 100,
		Pattern:   `^[a-zA-Z0-9\s\-'.]+$`,
		Sanitize:  true,
	}

	// DescriptionValidation for description fields
	DescriptionValidation = ValidationRule{
		Type:      "string",
		Required:  false,
		MaxLength: 1000,
		Sanitize:  true,
	}

	// PasswordValidation for password fields
	PasswordValidation = ValidationRule{
		Type:      "string",
		Required:  true,
		MinLength: 8,
		MaxLength: 128,
		Pattern:   `^[\S]+$`, // No whitespace
	}

	// PhoneValidation for phone numbers
	PhoneValidation = ValidationRule{
		Type:     "string",
		Required: false,
		Pattern:  `^\+?[1-9]\d{1,14}$`,
		Sanitize: true,
	}

	// AmountValidation for monetary amounts
	AmountValidation = ValidationRule{
		Type:     "number",
		Required: true,
		Min:      float64Ptr(0),
	}
)

// Helper function to create float64 pointer
func float64Ptr(f float64) *float64 {
	return &f
}
