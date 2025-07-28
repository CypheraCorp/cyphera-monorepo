package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	dsClient "github.com/cyphera/cyphera-api/libs/go/client/delegation_server"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/helpers"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/params"
	"github.com/cyphera/cyphera-api/libs/go/types/business"

	"go.uber.org/zap"
)

// PaymentServiceInterface defines only what we need from PaymentService to avoid import cycles
type PaymentServiceInterface interface {
	CreatePaymentFromSubscriptionEvent(ctx context.Context, params params.CreatePaymentFromSubscriptionEventParams) (*db.Payment, error)
}

// RedemptionProcessor processes redemption tasks from a queue
type RedemptionProcessor struct {
	tasks            chan business.RedemptionTask
	dbQueries        db.Querier
	delegationClient *dsClient.DelegationClient
	paymentService   PaymentServiceInterface
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
	pendingTasks        []business.RedemptionTask
}

// NewRedemptionProcessor creates a new redemption processor with the given number of workers
// and queue buffer size
func NewRedemptionProcessor(
	dbQueries db.Querier,
	delegationClient *dsClient.DelegationClient,
	paymentService PaymentServiceInterface,
	workerCount int,
	bufferSize int,
) *RedemptionProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	rp := &RedemptionProcessor{
		tasks:            make(chan business.RedemptionTask, bufferSize),
		dbQueries:        dbQueries,
		delegationClient: delegationClient,
		paymentService:   paymentService,
		workerCount:      workerCount,
		ctx:              ctx,
		cancel:           cancel,
		failureThreshold: 3,
		resetTimeout:     5 * time.Minute,
		pendingTasks:     make([]business.RedemptionTask, 0),
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
func (rp *RedemptionProcessor) QueueRedemption(task business.RedemptionTask) error {
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

// processRedemption processes a redemption task
func (rp *RedemptionProcessor) processRedemption(task business.RedemptionTask) error {
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
			ErrorMessage:   helpers.StringToNullableText(err.Error()),
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

	subscription, err := rp.dbQueries.GetSubscription(ctx, task.SubscriptionID)
	if err != nil {
		logger.Error("Failed to get subscription details",
			zap.Error(err),
			zap.String("subscription_id", task.SubscriptionID.String()),
		)
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

	executionObject := dsClient.ExecutionObject{
		MerchantAddress:      merchantWallet.WalletAddress,
		TokenContractAddress: token.ContractAddress,
		TokenAmount:          int64(subscription.TokenAmount),
		TokenDecimals:        token.Decimals,
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
			ErrorMessage:   helpers.StringToNullableText(err.Error()),
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
	event, err := rp.dbQueries.CreateRedemptionEvent(ctx, db.CreateRedemptionEventParams{
		SubscriptionID:  task.SubscriptionID,
		TransactionHash: helpers.StringToNullableText(txHash),
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

	// Create payment record for this successful redemption
	err = rp.createPaymentFromRedemption(ctx, event, subscription, product, token, network, txHash)
	if err != nil {
		logger.Error("Failed to create payment record from redemption",
			zap.Error(err),
			zap.String("subscription_id", task.SubscriptionID.String()),
			zap.String("tx_hash", txHash),
		)
		// Don't return error here - the redemption was successful, we just failed to record the payment
		// This can be retried later via a reconciliation process
	}

	// Update subscription next redemption date if needed
	_, err = rp.dbQueries.UpdateSubscription(ctx, db.UpdateSubscriptionParams{
		ID:                 task.SubscriptionID,
		NextRedemptionDate: helpers.TimeToNullableTimestamptz(time.Now().Add(30 * 24 * time.Hour)), // Example: monthly redemption
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
					rp.pendingTasks = make([]business.RedemptionTask, 0)
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

// createPaymentFromRedemption creates a payment record from a successful subscription redemption
func (rp *RedemptionProcessor) createPaymentFromRedemption(
	ctx context.Context,
	event db.SubscriptionEvent,
	subscription db.Subscription,
	product db.Product,
	token db.Token,
	network db.Network,
	txHash string,
) error {
	// Get the customer associated with this subscription
	customer, err := rp.dbQueries.GetCustomer(ctx, subscription.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	// Get the price for the subscription
	price, err := rp.dbQueries.GetPrice(ctx, subscription.PriceID)
	if err != nil {
		return fmt.Errorf("failed to get price: %w", err)
	}

	// Create payment from subscription event parameters
	params := params.CreatePaymentFromSubscriptionEventParams{
		SubscriptionEvent: &event,
		Subscription:      &subscription,
		Product:           &product,
		Price:             &price,
		Customer:          &customer,
		TransactionHash:   txHash,
		NetworkID:         network.ID,
		TokenID:           token.ID,
		// TODO: Add crypto amount and exchange rate if available
		// These would need to be calculated or retrieved from the transaction
		CryptoAmount: "",
		ExchangeRate: "",
		// TODO: Add gas fee information if available
		GasFeeUSDCents: 0,
		GasSponsored:   false,
	}

	// Create the payment
	payment, err := rp.paymentService.CreatePaymentFromSubscriptionEvent(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create payment from subscription event: %w", err)
	}

	logger.Info("Payment created successfully from redemption",
		zap.String("payment_id", payment.ID.String()),
		zap.String("subscription_id", subscription.ID.String()),
		zap.String("transaction_hash", txHash),
		zap.Int64("amount_cents", payment.AmountInCents))

	return nil
}
