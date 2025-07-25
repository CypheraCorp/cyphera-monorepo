package services

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/client/coinmarketcap"
	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// BlockchainSyncHelper helps sync blockchain data with payment records
type BlockchainSyncHelper struct {
	queries              db.Querier
	blockchainService    *BlockchainService
	paymentHelper        *PaymentService
	gasSponsorshipHelper *GasSponsorshipHelper
	cmcClient            *coinmarketcap.Client
	priceCache           map[string]*ethPriceCache // symbol -> cached price
}

// ethPriceCache holds cached ETH price data
type ethPriceCache struct {
	price     float64
	fetchedAt time.Time
}

// NewBlockchainSyncHelper creates a new blockchain sync helper
func NewBlockchainSyncHelper(queries db.Querier, blockchainService *BlockchainService, cmcClient *coinmarketcap.Client) *BlockchainSyncHelper {
	return &BlockchainSyncHelper{
		queries:              queries,
		blockchainService:    blockchainService,
		paymentHelper:        NewPaymentService(queries),
		gasSponsorshipHelper: NewGasSponsorshipHelper(queries),
		cmcClient:            cmcClient,
		priceCache:           make(map[string]*ethPriceCache),
	}
}

// SyncSubscriptionEventWithBlockchain fetches blockchain data and updates payment records
func (h *BlockchainSyncHelper) SyncSubscriptionEventWithBlockchain(ctx context.Context, event *db.SubscriptionEvent) error {
	// Only process events with transaction hashes
	if !event.TransactionHash.Valid || event.TransactionHash.String == "" {
		return nil
	}

	// Get transaction data from blockchain
	txData, err := h.blockchainService.GetTransactionDataFromEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to get transaction data: %w", err)
	}

	// Check if payment already exists for this event
	payment, err := h.queries.GetPaymentBySubscriptionEvent(ctx, pgtype.UUID{Bytes: event.ID, Valid: true})
	if err != nil {
		// Payment doesn't exist, this is handled elsewhere
		return nil
	}

	// Update payment with blockchain data
	err = h.updatePaymentWithBlockchainData(ctx, &payment, txData)
	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	// Create gas fee payment record
	err = h.createGasFeePaymentRecord(ctx, payment.ID, txData, payment.WorkspaceID)
	if err != nil {
		return fmt.Errorf("failed to create gas fee record: %w", err)
	}

	return nil
}

// updatePaymentWithBlockchainData updates a payment record with blockchain data
func (h *BlockchainSyncHelper) updatePaymentWithBlockchainData(ctx context.Context, payment *db.Payment, txData *TransactionData) error {
	// Get ETH price at transaction time
	ethUsdPrice, err := h.getETHPriceUSD(ctx, txData.BlockTimestamp)
	if err != nil {
		// Fall back to a default price if we can't fetch
		logger.Log.Warn("Failed to fetch ETH price, using default",
			zap.Error(err),
			zap.Uint64("block_timestamp", txData.BlockTimestamp),
		)
		ethUsdPrice = 2000.0 // Default fallback price
	}
	gasCostUsdCents := h.weiToUsdCents(txData.TotalGasCostWei, ethUsdPrice)

	// Check if transaction was successful
	status := "completed"
	if txData.Status == 0 {
		status = "failed"
	}

	// Update payment record
	_, err = h.queries.UpdatePaymentWithBlockchainData(ctx, db.UpdatePaymentWithBlockchainDataParams{
		ID:             payment.ID,
		WorkspaceID:    payment.WorkspaceID,
		Status:         status,
		GasFeeUsdCents: pgtype.Int8{Int64: gasCostUsdCents, Valid: true},
		UpdatedAt:      pgtype.Timestamptz{Valid: false}, // Use default CURRENT_TIMESTAMP
	})

	return err
}

// createGasFeePaymentRecord creates a detailed gas fee payment record
func (h *BlockchainSyncHelper) createGasFeePaymentRecord(ctx context.Context, paymentID uuid.UUID, txData *TransactionData, workspaceID uuid.UUID) error {
	// Convert Wei values to strings for storage
	gasFeeWei := txData.TotalGasCostWei.String()
	gasPriceGwei := new(big.Int).Div(txData.GasPrice, big.NewInt(1e9)).String()

	// Get ETH price for gas fee calculation
	ethUsdPrice, err := h.getETHPriceUSD(ctx, txData.BlockTimestamp)
	if err != nil {
		ethUsdPrice = 2000.0 // Fallback price
	}
	gasFeeUsdCents := h.weiToUsdCents(txData.TotalGasCostWei, ethUsdPrice)

	// Get gas sponsorship details using the gas sponsorship helper
	sponsorType := "customer" // Default: customer pays
	gasSponsored := false
	var sponsorID pgtype.UUID
	var sponsorWorkspaceID pgtype.UUID

	// Get payment record to find subscription and customer info
	payment, err := h.queries.GetPayment(ctx, db.GetPaymentParams{
		ID:          paymentID,
		WorkspaceID: workspaceID,
	})
	if err == nil && payment.SubscriptionID.Valid {
		// Get subscription and product info for sponsorship check
		subscription, err := h.queries.GetSubscription(ctx, payment.SubscriptionID.Bytes)
		if err == nil {
			// Check sponsorship using the service
			shouldSponsor, sponsorTypeResult, err := h.gasSponsorshipHelper.QuickSponsorshipCheck(
				ctx,
				workspaceID,
				payment.CustomerID,
				subscription.ProductID,
				gasFeeUsdCents,
			)

			if err == nil && shouldSponsor {
				sponsorType = sponsorTypeResult
				gasSponsored = true
				sponsorWorkspaceID = pgtype.UUID{Bytes: workspaceID, Valid: true}

				// Record the sponsorship
				_ = h.gasSponsorshipHelper.RecordSponsorship(ctx, workspaceID, paymentID, gasFeeUsdCents)
			}
		}
	}

	// Handle EIP-1559 fields if present
	var baseFeeGwei, priorityFeeGwei pgtype.Text
	if txData.BaseFeePerGas != nil {
		baseFeeGwei = pgtype.Text{
			String: new(big.Int).Div(txData.BaseFeePerGas, big.NewInt(1e9)).String(),
			Valid:  true,
		}
	}
	if txData.MaxPriorityFeePerGas != nil {
		priorityFeeGwei = pgtype.Text{
			String: new(big.Int).Div(txData.MaxPriorityFeePerGas, big.NewInt(1e9)).String(),
			Valid:  true,
		}
	}

	// Create gas fee payment record
	_, err = h.queries.CreateGasFeePayment(ctx, db.CreateGasFeePaymentParams{
		PaymentID:          paymentID,
		GasFeeWei:          gasFeeWei,
		GasPriceGwei:       gasPriceGwei,
		GasUnitsUsed:       int64(txData.GasUsed),
		MaxGasUnits:        int64(txData.GasLimit),
		BaseFeeGwei:        baseFeeGwei,
		PriorityFeeGwei:    priorityFeeGwei,
		PaymentMethod:      "native", // ETH was used for gas
		SponsorType:        sponsorType,
		SponsorID:          sponsorID,
		SponsorWorkspaceID: sponsorWorkspaceID,
		NetworkID:          txData.NetworkID,
		BlockNumber:        pgtype.Int8{Int64: int64(txData.BlockNumber), Valid: true},
		EthUsdPrice:        h.floatToNumeric(ethUsdPrice),
		GasFeeUsdCents:     pgtype.Int8{Int64: gasFeeUsdCents, Valid: true},
	})

	if err != nil {
		return fmt.Errorf("failed to create gas fee payment: %w", err)
	}

	// Update payment record to reflect gas sponsorship
	if gasSponsored {
		_, err = h.queries.UpdatePaymentGasSponsorship(ctx, db.UpdatePaymentGasSponsorshipParams{
			ID:           paymentID,
			WorkspaceID:  workspaceID,
			GasSponsored: pgtype.Bool{Bool: true, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update gas sponsorship: %w", err)
		}
	}

	return nil
}

// getETHPriceUSD fetches the ETH/USD price, using cache if available
func (h *BlockchainSyncHelper) getETHPriceUSD(ctx context.Context, blockTimestamp uint64) (float64, error) {
	// Check cache first (cache for 5 minutes)
	cacheKey := "ETH"
	if cached, ok := h.priceCache[cacheKey]; ok {
		if time.Since(cached.fetchedAt) < 5*time.Minute {
			return cached.price, nil
		}
	}

	// Fetch from CoinMarketCap
	if h.cmcClient == nil {
		return 0, fmt.Errorf("CoinMarketCap client not configured")
	}

	response, err := h.cmcClient.GetLatestQuotes([]string{"ETH"}, []string{"USD"})
	if err != nil {
		return 0, fmt.Errorf("failed to fetch ETH price: %w", err)
	}

	// Extract ETH price from response
	ethData, ok := response.Data["ETH"]
	if !ok || len(ethData) == 0 {
		return 0, fmt.Errorf("ETH price data not found in response")
	}

	usdQuote, ok := ethData[0].Quote["USD"]
	if !ok {
		return 0, fmt.Errorf("USD quote not found for ETH")
	}

	// Cache the price
	h.priceCache[cacheKey] = &ethPriceCache{
		price:     usdQuote.Price,
		fetchedAt: time.Now(),
	}

	return usdQuote.Price, nil
}

// floatToNumeric converts a float64 to pgtype.Numeric
func (h *BlockchainSyncHelper) floatToNumeric(f float64) pgtype.Numeric {
	// Convert float to string with reasonable precision
	str := fmt.Sprintf("%.6f", f)
	var num pgtype.Numeric
	err := num.Scan(str)
	if err != nil {
		// Return invalid numeric if conversion fails
		return pgtype.Numeric{Valid: false}
	}
	return num
}

// weiToUsdCents converts Wei to USD cents given an ETH/USD price
func (h *BlockchainSyncHelper) weiToUsdCents(weiAmount *big.Int, ethUsdPrice float64) int64 {
	// Convert Wei to ETH (1 ETH = 10^18 Wei)
	ethAmount := new(big.Float).SetInt(weiAmount)
	ethAmount.Quo(ethAmount, big.NewFloat(1e18))

	// Convert ETH to USD
	usdAmount := new(big.Float).Mul(ethAmount, big.NewFloat(ethUsdPrice))

	// Convert USD to cents
	centsAmount := new(big.Float).Mul(usdAmount, big.NewFloat(100))

	// Convert to int64
	cents, _ := centsAmount.Int64()
	return cents
}

// SyncPendingTransactions syncs all pending subscription events with blockchain data
func (h *BlockchainSyncHelper) SyncPendingTransactions(ctx context.Context, workspaceID uuid.UUID) error {
	// Get events that have transaction hashes but haven't been synced
	events, err := h.queries.GetUnsyncedSubscriptionEventsWithTxHash(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("failed to get unsynced events: %w", err)
	}

	syncedCount := 0
	errorCount := 0

	for _, event := range events {
		err := h.SyncSubscriptionEventWithBlockchain(ctx, &event)
		if err != nil {
			// Log error but continue with other events
			errorCount++
			continue
		}
		syncedCount++
	}

	if errorCount > 0 {
		return fmt.Errorf("synced %d events but %d failed", syncedCount, errorCount)
	}

	return nil
}
