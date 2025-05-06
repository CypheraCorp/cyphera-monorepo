package handlers

import (
	"context"
	dsClient "cyphera-api/internal/client/delegation_server"
	"cyphera-api/internal/db"
	"cyphera-api/internal/logger"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// RedemptionTask represents a task to be processed by the redemption processor
type RedemptionTask struct {
	SubscriptionID uuid.UUID
	DelegationID   uuid.UUID
	ProductID      uuid.UUID
	ProductTokenID uuid.UUID
	AmountInCents  int32
	Metadata       map[string]interface{}
}

// RedemptionProcessor processes redemption tasks from a queue
type RedemptionProcessor struct {
	tasks            chan RedemptionTask
	dbQueries        *db.Queries
	delegationClient *dsClient.DelegationClient
	workerCount      int
	wg               sync.WaitGroup
	ctx              context.Context
	cancel           context.CancelFunc

	// Circuit breaker pattern to handle delegation server downtime
	mu                  sync.Mutex
	circuitOpen         bool
	consecutiveFailures int
	failureThreshold    int
	resetTimeout        time.Duration
	lastFailureTime     time.Time
	pendingTasks        []RedemptionTask
}

// NewRedemptionProcessor creates a new redemption processor with the given number of workers
// and queue buffer size
func NewRedemptionProcessor(
	dbQueries *db.Queries,
	delegationClient *dsClient.DelegationClient,
	workerCount int,
	bufferSize int,
) *RedemptionProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	rp := &RedemptionProcessor{
		tasks:            make(chan RedemptionTask, bufferSize),
		dbQueries:        dbQueries,
		delegationClient: delegationClient,
		workerCount:      workerCount,
		ctx:              ctx,
		cancel:           cancel,
		failureThreshold: 3,
		resetTimeout:     5 * time.Minute,
		pendingTasks:     make([]RedemptionTask, 0),
	}

	return rp
}

// Start starts the redemption processor
func (rp *RedemptionProcessor) Start() {
	logger.Info("Starting redemption processor with workers", zap.Int("worker_count", rp.workerCount))

	// Start a separate goroutine to monitor the delegation server health
	go rp.monitorDelegationServerHealth()

	// Start worker goroutines
	for i := 0; i < rp.workerCount; i++ {
		workerID := i
		rp.wg.Add(1)

		go func() {
			defer rp.wg.Done()
			logger.Debug("Redemption worker started", zap.Int("worker_id", workerID))

			for {
				select {
				case <-rp.ctx.Done():
					logger.Debug("Redemption worker stopped", zap.Int("worker_id", workerID))
					return
				case task := <-rp.tasks:
					err := rp.processRedemption(task)
					if err != nil {
						logger.Error("Failed to process redemption",
							zap.Error(err),
							zap.String("subscription_id", task.SubscriptionID.String()),
						)
					}
				}
			}
		}()
	}
}

// Stop stops the redemption processor
func (rp *RedemptionProcessor) Stop() {
	logger.Info("Stopping redemption processor")
	rp.cancel()
	rp.wg.Wait()
	logger.Info("Redemption processor stopped")
}

// QueueRedemption adds a redemption task to the queue
func (rp *RedemptionProcessor) QueueRedemption(task RedemptionTask) error {
	// Check if circuit breaker is open
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.circuitOpen {
		// Store task for later processing when the circuit breaker resets
		logger.Info("Circuit breaker open, storing task for later",
			zap.String("subscription_id", task.SubscriptionID.String()),
		)
		rp.pendingTasks = append(rp.pendingTasks, task)
		return nil
	}

	// Try to add the task to the queue, with a timeout to avoid blocking
	select {
	case rp.tasks <- task:
		logger.Debug("Redemption task queued",
			zap.String("subscription_id", task.SubscriptionID.String()),
		)
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("redemption queue is full, try again later")
	}
}

// Helper function to convert string to nullable text
func stringToNullableText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// Helper function to convert time to nullable timestamptz
func timeToNullableTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// processRedemption processes a redemption task
func (rp *RedemptionProcessor) processRedemption(task RedemptionTask) error {
	ctx, cancel := context.WithTimeout(rp.ctx, 60*time.Second)
	defer cancel()

	// Check if the delegation server is available before attempting redemption
	err := rp.delegationClient.HealthCheck(ctx)
	if err != nil {
		logger.Warn("Delegation server unavailable, incrementing failure counter",
			zap.Error(err),
			zap.String("subscription_id", task.SubscriptionID.String()),
		)

		// Increment consecutive failures and consider opening circuit breaker
		rp.mu.Lock()
		rp.consecutiveFailures++
		rp.lastFailureTime = time.Now()

		if rp.consecutiveFailures >= rp.failureThreshold && !rp.circuitOpen {
			logger.Warn("Opening circuit breaker due to consecutive failures",
				zap.Int("failure_count", rp.consecutiveFailures),
				zap.Int("threshold", rp.failureThreshold),
			)
			rp.circuitOpen = true
		}

		// Store task for later processing
		rp.pendingTasks = append(rp.pendingTasks, task)
		rp.mu.Unlock()

		// Log failed redemption event
		metadataBytes, _ := json.Marshal(map[string]interface{}{
			"error":       err.Error(),
			"retry_count": rp.consecutiveFailures,
		})

		_, dbErr := rp.dbQueries.CreateFailedRedemptionEvent(ctx, db.CreateFailedRedemptionEventParams{
			SubscriptionID: task.SubscriptionID,
			AmountInCents:  task.AmountInCents,
			ErrorMessage:   stringToNullableText(err.Error()),
			Metadata:       metadataBytes,
		})

		if dbErr != nil {
			logger.Error("Failed to log failed redemption event",
				zap.Error(dbErr),
				zap.String("subscription_id", task.SubscriptionID.String()),
			)
		}

		return fmt.Errorf("delegation server unavailable: %w", err)
	}

	// Reset consecutive failures counter since server is available
	rp.mu.Lock()
	if rp.consecutiveFailures > 0 {
		rp.consecutiveFailures = 0
		logger.Info("Reset consecutive failures counter, delegation server is available")
	}
	rp.mu.Unlock()

	// Get delegation data from database
	delegationData, err := rp.dbQueries.GetDelegationData(ctx, task.DelegationID)
	if err != nil {
		logger.Error("Failed to get delegation data",
			zap.Error(err),
			zap.String("delegation_id", task.DelegationID.String()),
		)
		return fmt.Errorf("failed to get delegation data: %w", err)
	}

	// get the product details
	product, err := rp.dbQueries.GetProductWithoutWorkspaceId(ctx, task.ProductID)
	if err != nil {
		logger.Error("Failed to get product details",
			zap.Error(err),
			zap.String("product_id", task.ProductID.String()),
		)
		return fmt.Errorf("failed to get product details: %w", err)
	}

	// get the merchant's wallet details
	merchantWallet, err := rp.dbQueries.GetWalletByID(ctx, db.GetWalletByIDParams{
		ID:          product.WalletID,
		WorkspaceID: product.WorkspaceID,
	})
	if err != nil {
		logger.Error("Failed to get merchant wallet details",
			zap.Error(err),
			zap.String("delegation_id", task.DelegationID.String()),
		)
		return fmt.Errorf("failed to get merchant wallet details: %w", err)
	}

	// get the product token details
	productToken, err := rp.dbQueries.GetProductToken(ctx, task.ProductTokenID)
	if err != nil {
		logger.Error("Failed to get product token details",
			zap.Error(err),
			zap.String("product_token_id", task.ProductTokenID.String()),
		)
		return fmt.Errorf("failed to get product token details: %w", err)
	}

	// get the token details
	token, err := rp.dbQueries.GetToken(ctx, productToken.TokenID)
	if err != nil {
		logger.Error("Failed to get token details",
			zap.Error(err),
			zap.String("token_id", productToken.TokenID.String()),
		)
		return fmt.Errorf("failed to get token details: %w", err)
	}

	// get the network details
	network, err := rp.dbQueries.GetNetwork(ctx, token.NetworkID)
	if err != nil {
		logger.Error("Failed to get network details",
			zap.Error(err),
			zap.String("network_id", token.NetworkID.String()),
		)
		return fmt.Errorf("failed to get network details: %w", err)
	}

	// convert price in pennies to float
	price := float64(product.PriceInPennies) / 100.0

	executionObject := dsClient.ExecutionObject{
		MerchantAddress:      merchantWallet.WalletAddress,
		TokenContractAddress: token.ContractAddress,
		Price:                strconv.FormatFloat(price, 'f', -1, 64),
		ChainID:              uint32(network.ChainID),
		NetworkName:          network.Name,
	}

	// Create the delegation JSON with the required fields
	delegationJSON, err := json.Marshal(map[string]interface{}{
		"delegate":  delegationData.Delegate,
		"delegator": delegationData.Delegator,
		"authority": delegationData.Authority,
		"caveats":   delegationData.Caveats,
		"salt":      delegationData.Salt,
		"signature": delegationData.Signature,
	})

	if err != nil {
		logger.Error("Failed to marshal delegation data",
			zap.Error(err),
			zap.String("delegation_id", task.DelegationID.String()),
		)
		return fmt.Errorf("failed to marshal delegation data: %w", err)
	}

	// Call delegation service to redeem delegation
	logger.Info("Attempting to redeem delegation",
		zap.String("subscription_id", task.SubscriptionID.String()),
		zap.String("delegation_id", task.DelegationID.String()),
	)

	txHash, err := rp.delegationClient.RedeemDelegation(ctx, delegationJSON, executionObject)
	if err != nil {
		logger.Error("Failed to redeem delegation",
			zap.Error(err),
			zap.String("subscription_id", task.SubscriptionID.String()),
			zap.String("delegation_id", task.DelegationID.String()),
		)

		// Log failed redemption event
		metadataBytes, _ := json.Marshal(map[string]interface{}{
			"error": err.Error(),
		})

		_, dbErr := rp.dbQueries.CreateFailedRedemptionEvent(ctx, db.CreateFailedRedemptionEventParams{
			SubscriptionID: task.SubscriptionID,
			AmountInCents:  task.AmountInCents,
			ErrorMessage:   stringToNullableText(err.Error()),
			Metadata:       metadataBytes,
		})

		if dbErr != nil {
			logger.Error("Failed to log failed redemption event",
				zap.Error(dbErr),
				zap.String("subscription_id", task.SubscriptionID.String()),
			)
		}

		return fmt.Errorf("failed to redeem delegation: %w", err)
	}

	// Log success
	logger.Info("Delegation redemption successful",
		zap.String("subscription_id", task.SubscriptionID.String()),
		zap.String("tx_hash", txHash),
	)

	// Record successful redemption in database
	metadataBytes, _ := json.Marshal(task.Metadata)
	_, err = rp.dbQueries.CreateRedemptionEvent(ctx, db.CreateRedemptionEventParams{
		SubscriptionID:  task.SubscriptionID,
		TransactionHash: stringToNullableText(txHash),
		AmountInCents:   task.AmountInCents,
		Metadata:        metadataBytes,
	})

	if err != nil {
		logger.Error("Failed to record redemption event",
			zap.Error(err),
			zap.String("subscription_id", task.SubscriptionID.String()),
			zap.String("tx_hash", txHash),
		)
		return fmt.Errorf("failed to record redemption event: %w", err)
	}

	// Update subscription next redemption date if needed
	// We need to use a direct query since we don't have a dedicated method in the generated code
	// This is a simplified example - you'll need to adjust based on your actual schema
	_, err = rp.dbQueries.UpdateSubscription(ctx, db.UpdateSubscriptionParams{
		ID:                 task.SubscriptionID,
		NextRedemptionDate: timeToNullableTimestamptz(time.Now().Add(30 * 24 * time.Hour)), // Example: monthly redemption
	})

	if err != nil {
		logger.Error("Failed to update subscription next redemption date",
			zap.Error(err),
			zap.String("subscription_id", task.SubscriptionID.String()),
		)
		return fmt.Errorf("failed to update subscription next redemption date: %w", err)
	}

	return nil
}

// monitorDelegationServerHealth periodically checks if the delegation server is available
// and resets the circuit breaker if it becomes available again
func (rp *RedemptionProcessor) monitorDelegationServerHealth() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rp.ctx.Done():
			return
		case <-ticker.C:
			// Only check health if circuit breaker is open
			rp.mu.Lock()
			if !rp.circuitOpen {
				rp.mu.Unlock()
				continue
			}

			// Check if we need to attempt reset based on timeout
			if time.Since(rp.lastFailureTime) < rp.resetTimeout {
				rp.mu.Unlock()
				continue
			}

			rp.mu.Unlock()

			// Check if server is available
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := rp.delegationClient.HealthCheck(ctx)
			cancel()

			if err == nil {
				// Server is available, reset circuit breaker
				rp.mu.Lock()
				if rp.circuitOpen {
					logger.Info("Delegation server is available, resetting circuit breaker")
					rp.circuitOpen = false
					rp.consecutiveFailures = 0

					// Process any pending tasks
					pendingTasks := rp.pendingTasks
					rp.pendingTasks = make([]RedemptionTask, 0)
					rp.mu.Unlock()

					// Queue pending tasks
					for _, task := range pendingTasks {
						logger.Info("Requeuing pending task after circuit breaker reset",
							zap.String("subscription_id", task.SubscriptionID.String()),
						)
						_ = rp.QueueRedemption(task)
					}
				} else {
					rp.mu.Unlock()
				}
			}
		}
	}
}
