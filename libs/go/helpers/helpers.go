package helpers

// Stage constants define the possible deployment/runtime environments.
const (
	StageProd  = "prod"
	StageDev   = "dev"
	StageLocal = "local"
)

// IsValidStage checks if the provided stage string is one of the defined valid stages.
func IsValidStage(stage string) bool {
	switch stage {
	case StageProd, StageDev, StageLocal:
		return true
	default:
		return false
	}
}
