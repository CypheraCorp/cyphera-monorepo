package params

import "github.com/google/uuid"

// CreateWorkspaceParams contains parameters for creating a workspace
type CreateWorkspaceParams struct {
	Name         string
	Description  string
	BusinessName string
	BusinessType string
	WebsiteURL   string
	SupportEmail string
	SupportPhone string
	AccountID    uuid.UUID
	Metadata     map[string]interface{}
	Livemode     bool
}

// UpdateWorkspaceParams contains parameters for updating a workspace
type UpdateWorkspaceParams struct {
	ID           uuid.UUID
	Name         *string
	Description  *string
	BusinessName *string
	BusinessType *string
	WebsiteURL   *string
	SupportEmail *string
	SupportPhone *string
	Metadata     map[string]interface{}
	Livemode     *bool
}
