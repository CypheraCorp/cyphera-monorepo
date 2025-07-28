package responses

// WorkspaceResponse represents a workspace in API responses
type WorkspaceResponse struct {
	ID           string                 `json:"id"`
	Object       string                 `json:"object"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	BusinessName string                 `json:"business_name,omitempty"`
	BusinessType string                 `json:"business_type,omitempty"`
	WebsiteURL   string                 `json:"website_url,omitempty"`
	SupportEmail string                 `json:"support_email,omitempty"`
	SupportPhone string                 `json:"support_phone,omitempty"`
	AccountID    string                 `json:"account_id"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Livemode     bool                   `json:"livemode"`
	CreatedAt    int64                  `json:"created_at"`
	UpdatedAt    int64                  `json:"updated_at"`
}

// ListWorkspacesResponse represents the response for listing workspaces
type ListWorkspacesResponse struct {
	Object  string              `json:"object"`
	Data    []WorkspaceResponse `json:"data"`
	HasMore bool                `json:"has_more,omitempty"`
	Total   int64               `json:"total,omitempty"`
}
