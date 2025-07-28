package params

import "github.com/google/uuid"

// TransactionalEmailParams contains parameters for sending transactional emails
type TransactionalEmailParams struct {
	WorkspaceID uuid.UUID
	To          []string
	From        string
	FromName    string
	Subject     string
	HTMLContent string
	TextContent string
	ReplyTo     *string
	CC          []string
	BCC         []string
	Attachments []EmailAttachment
	Headers     map[string]string
	Tags        map[string]interface{}
	Metadata    map[string]interface{}
}

// EmailAttachment represents an email attachment
type EmailAttachment struct {
	Filename    string
	Content     []byte
	ContentType string
}
