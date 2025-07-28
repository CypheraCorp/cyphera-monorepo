package requests

// BatchEmailRequest represents a single email in a batch send request
type BatchEmailRequest struct {
	ToEmail     string                 `json:"to_email" binding:"required,email"`
	ToName      string                 `json:"to_name,omitempty"`
	Subject     string                 `json:"subject" binding:"required"`
	HTMLContent string                 `json:"html_content" binding:"required"`
	TextContent string                 `json:"text_content,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Tags        map[string]string      `json:"tags,omitempty"`
}
