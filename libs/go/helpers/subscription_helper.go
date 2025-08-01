package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/logger"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"
)

// CalculateNextRedemption calculates the next redemption time based on the interval type.
// For testing and development purposes, 1min and 5mins intervals are supported.
func CalculateNextRedemption(intervalType string, currentTime time.Time) time.Time {
	switch intervalType {
	case "1min":
		return currentTime.Add(1 * time.Minute)
	case "5mins":
		return currentTime.Add(5 * time.Minute)
	case "daily":
		return currentTime.AddDate(0, 0, 1) // Next day
	case "week":
		return currentTime.AddDate(0, 0, 7) // Next week
	case "month":
		return currentTime.AddDate(0, 1, 0) // Next month
	case "year":
		return currentTime.AddDate(1, 0, 0) // Next year
	default:
		return currentTime.AddDate(0, 1, 0) // Default to monthly
	}
}

// CalculatePeriodEnd determines the end date of a subscription period based on interval type and term length.
// This function is used when creating or updating subscription periods.
func CalculatePeriodEnd(start time.Time, intervalType string, termLength int32) time.Time {
	switch intervalType {
	case "1min":
		return start.Add(time.Duration(termLength) * time.Minute)
	case "5mins":
		return start.Add(time.Duration(termLength*5) * time.Minute)
	case "daily":
		return start.AddDate(0, 0, int(termLength))
	case "week":
		return start.AddDate(0, 0, int(termLength*7))
	case "month":
		return start.AddDate(0, int(termLength), 0)
	case "year":
		return start.AddDate(int(termLength), 0, 0)
	default:
		return start.AddDate(0, int(termLength), 0) // Default to monthly
	}
}

// isPermanentRedemptionError determines if a redemption error should not be retried
func IsPermanentRedemptionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for signatures of permanent errors
	errMsg := err.Error()
	permanentErrorSigns := []string{
		"invalid signature",
		"delegation expired",
		"invalid delegation format",
		"invalid token",
		"unauthorized",
		"insufficient funds",
	}

	for _, sign := range permanentErrorSigns {
		if strings.Contains(strings.ToLower(errMsg), sign) {
			return true
		}
	}

	return false
}

// ToSubscriptionResponse converts a db.ListSubscriptionDetailsWithPaginationRow to a SubscriptionResponse.
func ToSubscriptionResponse(subDetails db.ListSubscriptionDetailsWithPaginationRow) (responses.SubscriptionResponse, error) {
	// Helper to convert pgtype.Text to string for enums
	helperPgEnumTextToString := func(pgText pgtype.Text) string {
		if pgText.Valid {
			return pgText.String
		}
		return ""
	}

	// Prepare PriceResponse.CreatedAt and PriceResponse.UpdatedAt (assuming they are int64 in PriceResponse)
	var priceCreatedAtUnix int64
	if subDetails.PriceCreatedAt.Valid {
		priceCreatedAtUnix = subDetails.PriceCreatedAt.Time.Unix()
	}
	var priceUpdatedAtUnix int64
	if subDetails.PriceUpdatedAt.Valid {
		priceUpdatedAtUnix = subDetails.PriceUpdatedAt.Time.Unix()
	}

	// Prepare nullable fields for PriceResponse
	var intervalTypeStr string
	if subDetails.PriceIntervalType != "" {
		intervalTypeStr = string(subDetails.PriceIntervalType)
	}

	// Parse customer metadata
	var customerMetadata map[string]interface{}
	if len(subDetails.CustomerMetadata) > 0 {
		if err := json.Unmarshal(subDetails.CustomerMetadata, &customerMetadata); err != nil {
			logger.Error("Error unmarshaling customer metadata", zap.Error(err))
			customerMetadata = make(map[string]interface{})
		}
	}

	resp := responses.SubscriptionResponse{
		ID:                 subDetails.SubscriptionID,
		NumID:              subDetails.SubscriptionNumID,
		WorkspaceID:        subDetails.ProductWorkspaceID,
		Status:             string(subDetails.SubscriptionStatus),
		CurrentPeriodStart: subDetails.SubscriptionCurrentPeriodStart.Time,
		CurrentPeriodEnd:   subDetails.SubscriptionCurrentPeriodEnd.Time,
		CreatedAt:          subDetails.SubscriptionCreatedAt.Time,
		UpdatedAt:          subDetails.SubscriptionUpdatedAt.Time,
		CustomerID:         subDetails.CustomerID,
		TokenAmount:        int32(subDetails.SubscriptionTokenAmount),
		CustomerName:       subDetails.CustomerName.String,
		CustomerEmail:      subDetails.CustomerEmail.String,
		Customer: &responses.SubscriptionCustomerResponse{
			ID:                 subDetails.CustomerID,
			NumID:              subDetails.CustomerNumID,
			Name:               subDetails.CustomerName.String,
			Email:              subDetails.CustomerEmail.String,
			Phone:              subDetails.CustomerPhone.String,
			Description:        subDetails.CustomerDescription.String,
			FinishedOnboarding: subDetails.CustomerFinishedOnboarding.Bool,
			Metadata:           customerMetadata,
			CreatedAt:          subDetails.CustomerCreatedAt.Time,
			UpdatedAt:          subDetails.CustomerUpdatedAt.Time,
		},
		Price: responses.PriceResponse{
			ID:                  subDetails.PriceID.String(),
			Object:              "price", // Typically static for PriceResponse
			ProductID:           subDetails.PriceProductID.String(),
			Active:              subDetails.PriceActive,
			Type:                string(subDetails.PriceType),
			Nickname:            helperPgEnumTextToString(subDetails.PriceNickname),
			Currency:            string(subDetails.PriceCurrency),
			UnitAmountInPennies: int64(subDetails.PriceUnitAmountInPennies),
			IntervalType:        intervalTypeStr,
			TermLength:          subDetails.PriceTermLength,
			Metadata:            subDetails.PriceMetadata, // Expecting json.RawMessage
			CreatedAt:           priceCreatedAtUnix,
			UpdatedAt:           priceUpdatedAtUnix,
		},
		Product: responses.ProductResponse{
			ID:          subDetails.ProductID.String(),
			Name:        subDetails.ProductName,
			Description: subDetails.ProductDescription.String,
			ImageURL:    subDetails.ProductImageUrl.String,
			Active:      subDetails.ProductActive,
			Metadata:    subDetails.ProductMetadata, // Expecting json.RawMessage
		},
		ProductToken: responses.ProductTokenResponse{
			ID:          subDetails.ProductTokenID.String(),
			TokenID:     subDetails.ProductTokenTokenID.String(),
			TokenSymbol: subDetails.TokenSymbol,
			NetworkID:   subDetails.ProductTokenNetworkID.String(),
			CreatedAt:   subDetails.ProductTokenCreatedAt.Time.Unix(),
			UpdatedAt:   subDetails.ProductTokenUpdatedAt.Time.Unix(),
		},
	}
	return resp, nil
}

// ToSubscriptionResponseFromDBSubscription converts a db.Subscription to a SubscriptionResponse.
func ToSubscriptionResponseFromDBSubscription(sub db.Subscription) (responses.SubscriptionResponse, error) {
	resp := responses.SubscriptionResponse{
		ID:          sub.ID,
		NumID:       sub.NumID,
		CustomerID:  sub.CustomerID,
		WorkspaceID: sub.WorkspaceID,
		Status:      string(sub.Status),
		TokenAmount: int32(sub.TokenAmount),
		CreatedAt:   sub.CreatedAt.Time,
		UpdatedAt:   sub.UpdatedAt.Time,
	}

	if sub.CurrentPeriodStart.Valid {
		resp.CurrentPeriodStart = sub.CurrentPeriodStart.Time
	}
	if sub.CurrentPeriodEnd.Valid {
		resp.CurrentPeriodEnd = sub.CurrentPeriodEnd.Time
	}

	return resp, nil
}

// CalculateSubscriptionPeriods determines the start, end, and next redemption dates based on a price
func CalculateSubscriptionPeriods(price db.Price) (time.Time, time.Time, time.Time) {
	now := time.Now()
	periodStart := now
	var periodEnd time.Time
	var nextRedemption time.Time

	termLength := 1
	if price.TermLength != 0 {
		termLength = int(price.TermLength)
	}

	if price.Type == db.PriceTypeRecurring {
		periodEnd = CalculatePeriodEnd(now, string(price.IntervalType), int32(termLength))
		nextRedemption = CalculateNextRedemption(string(price.IntervalType), now)
	} else {
		periodEnd = now
		nextRedemption = now
	}

	return periodStart, periodEnd, nextRedemption
}

// DetermineErrorType determines the error type based on the error message
func DetermineErrorType(err error) db.SubscriptionEventType {
	errorMsg := err.Error()
	if strings.Contains(errorMsg, "validation") {
		return db.SubscriptionEventTypeFailedValidation
	} else if strings.Contains(errorMsg, "customer") && strings.Contains(errorMsg, "create") {
		return db.SubscriptionEventTypeFailedCustomerCreation
	} else if strings.Contains(errorMsg, "wallet") && strings.Contains(errorMsg, "create") {
		return db.SubscriptionEventTypeFailedWalletCreation
	} else if strings.Contains(errorMsg, "delegation") {
		return db.SubscriptionEventTypeFailedDelegationStorage
	} else if strings.Contains(errorMsg, "subscription already exists") {
		return db.SubscriptionEventTypeFailedDuplicate
	} else if strings.Contains(errorMsg, "database") || strings.Contains(errorMsg, "db") {
		return db.SubscriptionEventTypeFailedSubscriptionDb
	} else {
		return db.SubscriptionEventTypeFailed
	}
}

// ToComprehensiveSubscriptionResponse converts a db.Subscription to a comprehensive SubscriptionResponse
// that includes all subscription fields plus the initial transaction hash from subscription events
func ToComprehensiveSubscriptionResponse(ctx context.Context, queries db.Querier, subscription db.Subscription) (*responses.SubscriptionResponse, error) {
	logger.Log.Info("Starting ToComprehensiveSubscriptionResponse",
		zap.String("subscription_id", subscription.ID.String()),
		zap.String("workspace_id", subscription.WorkspaceID.String()))

	// Get the subscription details with related data
	logger.Log.Info("Calling GetSubscriptionWithDetails query",
		zap.String("subscription_id", subscription.ID.String()))

	subscriptionDetails, err := queries.GetSubscriptionWithDetails(ctx, db.GetSubscriptionWithDetailsParams{
		ID:          subscription.ID,
		WorkspaceID: subscription.WorkspaceID,
	})
	if err != nil {
		logger.Log.Error("GetSubscriptionWithDetails query failed",
			zap.Error(err),
			zap.String("subscription_id", subscription.ID.String()))
		return nil, fmt.Errorf("failed to get subscription details: %w", err)
	}

	logger.Log.Info("GetSubscriptionWithDetails query completed successfully",
		zap.String("subscription_id", subscription.ID.String()))

	// Get the initial transaction hash from the first redemption event
	logger.Log.Info("Calling ListSubscriptionEventsBySubscription query",
		zap.String("subscription_id", subscription.ID.String()))

	initialTxHash := ""
	events, err := queries.ListSubscriptionEventsBySubscription(ctx, subscription.ID)
	if err == nil {
		logger.Log.Info("ListSubscriptionEventsBySubscription query completed successfully",
			zap.String("subscription_id", subscription.ID.String()),
			zap.Int("events_count", len(events)))

		for _, event := range events {
			if event.EventType == db.SubscriptionEventTypeRedeemed && event.TransactionHash.Valid {
				initialTxHash = event.TransactionHash.String
				break
			}
		}
	} else {
		logger.Log.Error("ListSubscriptionEventsBySubscription query failed",
			zap.Error(err),
			zap.String("subscription_id", subscription.ID.String()))
	}

	// Parse metadata
	var metadata map[string]interface{}
	if len(subscription.Metadata) > 0 {
		if err := json.Unmarshal(subscription.Metadata, &metadata); err != nil {
			logger.Error("Error unmarshaling subscription metadata", zap.Error(err))
			metadata = make(map[string]interface{})
		}
	}

	// Parse customer metadata
	var customerMetadata map[string]interface{}
	if len(subscriptionDetails.CustomerMetadata) > 0 {
		if err := json.Unmarshal(subscriptionDetails.CustomerMetadata, &customerMetadata); err != nil {
			logger.Error("Error unmarshaling customer metadata", zap.Error(err))
			customerMetadata = make(map[string]interface{})
		}
	}

	// Convert to response
	response := &responses.SubscriptionResponse{
		ID:                     subscription.ID,
		NumID:                  subscription.NumID,
		WorkspaceID:            subscription.WorkspaceID,
		CustomerID:             subscription.CustomerID,
		Status:                 string(subscription.Status),
		CurrentPeriodStart:     subscription.CurrentPeriodStart.Time,
		CurrentPeriodEnd:       subscription.CurrentPeriodEnd.Time,
		TotalRedemptions:       subscription.TotalRedemptions,
		TotalAmountInCents:     subscription.TotalAmountInCents,
		TokenAmount:            subscription.TokenAmount,
		DelegationID:           subscription.DelegationID,
		InitialTransactionHash: initialTxHash,
		Metadata:               metadata,
		CreatedAt:              subscription.CreatedAt.Time,
		UpdatedAt:              subscription.UpdatedAt.Time,
		CustomerName:           subscriptionDetails.CustomerName.String,
		CustomerEmail:          subscriptionDetails.CustomerEmail.String,
		Customer: &responses.SubscriptionCustomerResponse{
			ID:                 subscriptionDetails.CustomerID,
			NumID:              subscriptionDetails.CustomerNumID,
			Name:               subscriptionDetails.CustomerName.String,
			Email:              subscriptionDetails.CustomerEmail.String,
			Phone:              subscriptionDetails.CustomerPhone.String,
			Description:        subscriptionDetails.CustomerDescription.String,
			FinishedOnboarding: subscriptionDetails.CustomerFinishedOnboarding.Bool,
			Metadata:           customerMetadata,
			CreatedAt:          subscriptionDetails.CustomerCreatedAt.Time,
			UpdatedAt:          subscriptionDetails.CustomerUpdatedAt.Time,
		},
		Price: responses.PriceResponse{
			ID:                  subscriptionDetails.PriceID.String(),
			Object:              "price",
			ProductID:           subscriptionDetails.ProductID.String(),
			Active:              true, // Assuming active for subscriptions
			Type:                string(subscriptionDetails.PriceType),
			Currency:            string(subscriptionDetails.PriceCurrency),
			UnitAmountInPennies: int64(subscriptionDetails.PriceUnitAmountInPennies), // Convert int32 to int64
			IntervalType:        string(subscriptionDetails.PriceIntervalType),
			TermLength:          subscriptionDetails.PriceTermLength,
			CreatedAt:           time.Now().Unix(), // Default timestamp
			UpdatedAt:           time.Now().Unix(), // Default timestamp
		},
		Product: responses.ProductResponse{
			ID:     subscriptionDetails.ProductID.String(),
			Name:   subscriptionDetails.ProductName,
			Active: true, // Assuming active for subscriptions
			Object: "product",
		},
		ProductToken: responses.ProductTokenResponse{
			ID:          subscriptionDetails.ProductTokenID.String(),
			TokenSymbol: subscriptionDetails.TokenSymbol,
			NetworkID:   subscriptionDetails.ProductTokenID.String(),
			CreatedAt:   time.Now().Unix(), // Default timestamp
			UpdatedAt:   time.Now().Unix(), // Default timestamp
		},
	}

	// Handle nullable fields
	if subscription.NextRedemptionDate.Valid {
		response.NextRedemptionDate = &subscription.NextRedemptionDate.Time
	}

	if subscription.CustomerWalletID.Valid {
		walletID := uuid.UUID(subscription.CustomerWalletID.Bytes)
		response.CustomerWalletID = &walletID
	}

	if subscription.ExternalID.Valid {
		response.ExternalID = subscription.ExternalID.String
	}

	if subscription.PaymentSyncStatus.Valid {
		response.PaymentSyncStatus = subscription.PaymentSyncStatus.String
	}

	if subscription.PaymentSyncedAt.Valid {
		response.PaymentSyncedAt = &subscription.PaymentSyncedAt.Time
	}

	if subscription.PaymentSyncVersion.Valid {
		response.PaymentSyncVersion = subscription.PaymentSyncVersion.Int32
	}

	if subscription.PaymentProvider.Valid {
		response.PaymentProvider = subscription.PaymentProvider.String
	}

	return response, nil
}
