package helpers

import "github.com/cyphera/cyphera-api/libs/go/constants"

// Stage constants define the possible deployment/runtime environments.
const (
	StageProd  = constants.ProdEnvironment
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
