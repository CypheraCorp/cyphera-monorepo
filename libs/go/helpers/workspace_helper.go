package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/google/uuid"
)

// WorkspaceHelper provides utility functions for workspace operations
type WorkspaceHelper struct {
	queries db.Querier
}

// NewWorkspaceHelper creates a new workspace helper
func NewWorkspaceHelper(queries db.Querier) *WorkspaceHelper {
	return &WorkspaceHelper{
		queries: queries,
	}
}

// ValidateWorkspaceName validates a workspace name
func (h *WorkspaceHelper) ValidateWorkspaceName(name string) error {
	// Trim spaces
	name = strings.TrimSpace(name)

	// Check length
	if len(name) < 3 {
		return fmt.Errorf("workspace name must be at least 3 characters long")
	}
	if len(name) > 50 {
		return fmt.Errorf("workspace name must not exceed 50 characters")
	}

	// Check for valid characters (alphanumeric, spaces, hyphens, underscores)
	for _, char := range name {
		if !isValidWorkspaceNameChar(char) {
			return fmt.Errorf("workspace name contains invalid characters")
		}
	}

	return nil
}

// isValidWorkspaceNameChar checks if a character is valid for workspace names
func isValidWorkspaceNameChar(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9') ||
		char == ' ' || char == '-' || char == '_'
}

// GenerateWorkspaceSlug generates a URL-safe slug from workspace name
func (h *WorkspaceHelper) GenerateWorkspaceSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove any characters that aren't alphanumeric or hyphens
	var result strings.Builder
	for _, char := range slug {
		if (char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '-' {
			result.WriteRune(char)
		}
	}

	// Remove multiple consecutive hyphens
	slug = result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	return slug
}

// CheckWorkspaceNameUniqueness checks if a workspace name is unique for an account
func (h *WorkspaceHelper) CheckWorkspaceNameUniqueness(ctx context.Context, accountID uuid.UUID, name string) (bool, error) {
	workspaces, err := h.queries.ListWorkspacesByAccountID(ctx, accountID)
	if err != nil {
		return false, fmt.Errorf("failed to check workspace uniqueness: %w", err)
	}

	// Check if name already exists
	for _, workspace := range workspaces {
		if strings.EqualFold(workspace.Name, name) {
			return false, nil
		}
	}

	return true, nil
}

// WorkspaceMetadata represents structured workspace metadata
type WorkspaceMetadata struct {
	Industry     string                 `json:"industry,omitempty"`
	CompanySize  string                 `json:"company_size,omitempty"`
	Country      string                 `json:"country,omitempty"`
	Timezone     string                 `json:"timezone,omitempty"`
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"`
}

// ParseWorkspaceMetadata parses workspace metadata from JSON
func (h *WorkspaceHelper) ParseWorkspaceMetadata(metadataJSON []byte) (*WorkspaceMetadata, error) {
	var metadata WorkspaceMetadata
	if len(metadataJSON) == 0 || string(metadataJSON) == "{}" {
		return &metadata, nil
	}

	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse workspace metadata: %w", err)
	}

	return &metadata, nil
}

// SerializeWorkspaceMetadata serializes workspace metadata to JSON
func (h *WorkspaceHelper) SerializeWorkspaceMetadata(metadata *WorkspaceMetadata) ([]byte, error) {
	if metadata == nil {
		return []byte("{}"), nil
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize workspace metadata: %w", err)
	}

	return data, nil
}

// GetWorkspaceDefaults returns default values for workspace creation
func (h *WorkspaceHelper) GetWorkspaceDefaults() map[string]interface{} {
	return map[string]interface{}{
		"livemode":      false,
		"business_type": "other",
		"metadata":      map[string]interface{}{},
		"timezone":      "UTC",
	}
}

// ValidateBusinessEmail validates a business email address
func (h *WorkspaceHelper) ValidateBusinessEmail(email string) error {
	email = strings.TrimSpace(email)

	// Basic email validation
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return fmt.Errorf("invalid email format")
	}

	// Check for common free email providers (optional business rule)
	freeProviders := []string{
		"gmail.com", "yahoo.com", "hotmail.com", "outlook.com",
		"aol.com", "mail.com", "protonmail.com",
	}

	emailLower := strings.ToLower(email)
	for _, provider := range freeProviders {
		if strings.HasSuffix(emailLower, "@"+provider) {
			return fmt.Errorf("please use a business email address")
		}
	}

	return nil
}

// ValidateWebsiteURL validates a website URL
func (h *WorkspaceHelper) ValidateWebsiteURL(url string) error {
	url = strings.TrimSpace(url)

	// Check if it starts with http:// or https://
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("website URL must start with http:// or https://")
	}

	// Basic domain validation
	if len(url) < 10 { // Minimum: http://a.b
		return fmt.Errorf("invalid website URL")
	}

	return nil
}

// FormatPhoneNumber formats a phone number for storage
func (h *WorkspaceHelper) FormatPhoneNumber(phone string) string {
	// Remove all non-numeric characters
	var cleaned strings.Builder
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			cleaned.WriteRune(char)
		}
	}

	return cleaned.String()
}

// GetWorkspaceFeatures returns enabled features for a workspace based on its type
func (h *WorkspaceHelper) GetWorkspaceFeatures(workspaceType string, livemode bool) map[string]bool {
	// Default features
	features := map[string]bool{
		"subscriptions":     true,
		"one_time_payments": true,
		"invoices":          true,
		"customers":         true,
		"analytics":         true,
		"webhooks":          true,
		"api_access":        true,
	}

	// Add features based on mode
	if livemode {
		features["live_payments"] = true
		features["payouts"] = true
	} else {
		features["test_mode"] = true
		features["sandbox"] = true
	}

	// Add features based on business type
	switch workspaceType {
	case "saas":
		features["recurring_billing"] = true
		features["usage_based_billing"] = true
	case "marketplace":
		features["split_payments"] = true
		features["vendor_management"] = true
	case "ecommerce":
		features["shopping_cart"] = true
		features["product_catalog"] = true
	}

	return features
}
