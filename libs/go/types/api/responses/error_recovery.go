package responses

// DetailedErrorResponse represents an error response with additional details
type DetailedErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// DetailedSuccessResponse represents a success response with additional data
type DetailedSuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// WebhookReplayResponse represents the result of webhook replay
type WebhookReplayResponse struct {
	Success         bool   `json:"success"`
	ReplayEventID   string `json:"replay_event_id,omitempty"`
	OriginalEventID string `json:"original_event_id"`
	ReplayedAt      string `json:"replayed_at"`
	Message         string `json:"message"`
	Error           string `json:"error,omitempty"`
}

// SyncRecoveryResponse represents the result of sync recovery
type SyncRecoveryResponse struct {
	Success     bool                   `json:"success"`
	SessionID   string                 `json:"session_id"`
	RecoveredAt string                 `json:"recovered_at"`
	Progress    map[string]interface{} `json:"progress,omitempty"`
	Message     string                 `json:"message"`
	Error       string                 `json:"error,omitempty"`
}

// DLQProcessingStats represents DLQ processing statistics
type DLQProcessingStats struct {
	TotalMessages         int64   `json:"total_messages"`
	SuccessfullyProcessed int64   `json:"successfully_processed"`
	ProcessingFailed      int64   `json:"processing_failed"`
	MaxRetriesExceeded    int64   `json:"max_retries_exceeded"`
	LastProcessedAt       string  `json:"last_processed_at"`
	SuccessRate           float64 `json:"success_rate"`
}
