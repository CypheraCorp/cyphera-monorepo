package responses

// BatchEmailResult represents the result of sending a single email in a batch
type BatchEmailResult struct {
	Index      int    `json:"index"`
	ToEmail    string `json:"to_email"`
	Success    bool   `json:"success"`
	MessageID  string `json:"message_id,omitempty"`
	Error      string `json:"error,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}
