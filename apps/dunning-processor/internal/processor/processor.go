package processor

import (
	"context"
	"fmt"

	"github.com/cyphera/cyphera-api/libs/go/services"
	"go.uber.org/zap"
)

// DunningProcessor handles dunning campaign processing
type DunningProcessor struct {
	retryEngine *services.DunningRetryEngine
	logger      *zap.Logger
}

// NewDunningProcessor creates a new dunning processor
func NewDunningProcessor(retryEngine *services.DunningRetryEngine, logger *zap.Logger) *DunningProcessor {
	return &DunningProcessor{
		retryEngine: retryEngine,
		logger:      logger,
	}
}

// ProcessingResults holds the results of dunning processing
type ProcessingResults struct {
	Total           int
	Succeeded       int
	Failed          int
	EmailsSent      int
	PaymentsRetried int
}

// ProcessDueCampaigns processes all due dunning campaigns
func (p *DunningProcessor) ProcessDueCampaigns(ctx context.Context) (*ProcessingResults, error) {
	p.logger.Info("Starting dunning campaign processing")

	// Process campaigns with a limit of 100 per execution
	limit := int32(100)
	err := p.retryEngine.ProcessDueCampaigns(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to process dunning campaigns: %w", err)
	}

	// Get processing statistics
	// TODO: Implement proper statistics tracking
	results := &ProcessingResults{
		Total:           0, // Will be updated when we track stats
		Succeeded:       0,
		Failed:          0,
		EmailsSent:      0,
		PaymentsRetried: 0,
	}

	p.logger.Info("Dunning campaign processing completed",
		zap.Int("total", results.Total),
		zap.Int("succeeded", results.Succeeded),
		zap.Int("failed", results.Failed),
	)

	return results, nil
}