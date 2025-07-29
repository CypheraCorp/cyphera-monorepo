package responses

import "github.com/google/uuid"

// DetectionResult represents the result of payment failure detection
type DetectionResult struct {
	NewCampaigns     int         `json:"new_campaigns"`
	UpdatedCampaigns int         `json:"updated_campaigns"`
	FailedDetections int         `json:"failed_detections"`
	CampaignIDs      []uuid.UUID `json:"campaign_ids"`
	Errors           []string    `json:"errors,omitempty"`
}
