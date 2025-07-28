package processor

import (
	"context"

	"github.com/cyphera/cyphera-api/libs/go/interfaces"
)

// SubscriptionProcessor handles subscription processing
type SubscriptionProcessor struct {
	subscriptionService interfaces.SubscriptionService
}

// NewSubscriptionProcessor creates a new subscription processor
func NewSubscriptionProcessor(subscriptionService interfaces.SubscriptionService) *SubscriptionProcessor {
	return &SubscriptionProcessor{
		subscriptionService: subscriptionService,
	}
}

// ProcessDueSubscriptions processes all due subscriptions
func (p *SubscriptionProcessor) ProcessDueSubscriptions(ctx context.Context) (*ProcessingResults, error) {
	// Use the subscription service to process due subscriptions
	result, err := p.subscriptionService.ProcessDueSubscriptions(ctx)
	if err != nil {
		return nil, err
	}

	// Convert the service result to ProcessingResults
	return &ProcessingResults{
		Total:     result.ProcessedCount,
		Succeeded: result.SuccessfulCount,
		Failed:    result.FailedCount,
	}, nil
}

// ProcessingResults holds the results of subscription processing
type ProcessingResults struct {
	Total     int
	Succeeded int
	Failed    int
}
