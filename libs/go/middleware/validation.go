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
		// Check request size
		if config.MaxBodySize > 0 && c.Request.ContentLength > config.MaxBodySize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("Request body too large. Maximum size: %d bytes", config.MaxBodySize),
			})
			c.Abort()
			return
		}

		// Parse request body
		var body map[string]interface{}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid JSON in request body",
			})
			c.Abort()
			return
		}

		// Validate fields
		errors := validateFields(body, config.Rules, config.AllowUnknownFields)
		if len(errors) > 0 {
			c.JSON(http.StatusBadRequest, ValidationErrors{Errors: errors})
			c.Abort()
			return
		}

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

	// Check each rule
	for _, rule := range rules {
		validatedFields[rule.Field] = true
		value, exists := data[rule.Field]

		// Check required fields
		if rule.Required && (!exists || value == nil || value == "") {
			errors = append(errors, ValidationError{
				Field:   rule.Field,
				Message: fmt.Sprintf("%s is required", rule.Field),
			})
			continue
		}

		// Skip validation if field doesn't exist and not required
		if !exists || value == nil {
			continue
		}

		// Validate based on type
		switch rule.Type {
		case "string":
			err := validateString(value, rule)
			if err != nil {
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
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: err.Error(),
				})
			}

		case "boolean", "bool":
			if _, ok := value.(bool); !ok {
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: "must be a boolean",
				})
			}

		case "uuid":
			if err := validateUUID(value); err != nil {
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: err.Error(),
				})
			}

		case "email":
			if err := validateEmail(value); err != nil {
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: err.Error(),
				})
			}

		case "url":
			if err := validateURL(value); err != nil {
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: err.Error(),
				})
			}

		case "array":
			if _, ok := value.([]interface{}); !ok {
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: "must be an array",
				})
			}

		case "object":
			if _, ok := value.(map[string]interface{}); !ok {
				errors = append(errors, ValidationError{
					Field:   rule.Field,
					Message: "must be an object",
				})
			}
		}

		// Custom validation
		if rule.Custom != nil {
			if err := rule.Custom(value); err != nil {
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
				errors = append(errors, ValidationError{
					Field:   field,
					Message: "unknown field",
				})
			}
		}
	}

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
