package business

import "github.com/google/uuid"

// AttemptAction represents a dunning retry attempt configuration
type AttemptAction struct {
	Attempt         int32      `json:"attempt"`
	Actions         []string   `json:"actions"`
	EmailTemplateID *uuid.UUID `json:"email_template_id,omitempty"`
}
