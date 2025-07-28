package helpers

import (
	"strings"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/jackc/pgx/v5/pgtype"
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

	resp := responses.SubscriptionResponse{
		ID:                 subDetails.SubscriptionID,
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
