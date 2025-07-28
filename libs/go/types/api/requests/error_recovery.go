package requests

// WebhookReplayRequest represents a webhook replay request
type WebhookReplayRequest struct {
	WorkspaceID    string `json:"workspace_id" binding:"required"`
	ProviderName   string `json:"provider_name" binding:"required"`
	WebhookEventID string `json:"webhook_event_id" binding:"required"`
	ForceReplay    bool   `json:"force_replay"`
	ReplayReason   string `json:"replay_reason,omitempty"`
}

// SyncRecoveryRequest represents a sync session recovery request
type SyncRecoveryRequest struct {
	WorkspaceID  string   `json:"workspace_id" binding:"required"`
	SessionID    string   `json:"session_id" binding:"required"`
	RecoveryMode string   `json:"recovery_mode"` // "resume", "restart"
	EntityTypes  []string `json:"entity_types,omitempty"`
}
