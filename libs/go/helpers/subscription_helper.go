package helpers

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/google/uuid"
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

// SubscriptionResponse represents the response structure for subscription data
type SubscriptionResponse struct {
	ID                 uuid.UUID            `json:"id"`
	WorkspaceID        uuid.UUID            `json:"workspace_id"`
	Status             string               `json:"status"`
	CurrentPeriodStart time.Time            `json:"current_period_start"`
	CurrentPeriodEnd   time.Time            `json:"current_period_end"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
	CustomerID         uuid.UUID            `json:"customer_id"`
	TokenAmount        int64                `json:"token_amount"`
	CustomerName       string               `json:"customer_name"`
	CustomerEmail      string               `json:"customer_email"`
	Price              PriceResponse        `json:"price"`
	Product            ProductResponse      `json:"product"`
	ProductToken       ProductTokenResponse `json:"product_token"`
}

// PriceResponse represents the response structure for price data
type PriceResponse struct {
	ID                  string          `json:"id"`
	Object              string          `json:"object"`
	ProductID           string          `json:"product_id"`
	Active              bool            `json:"active"`
	Type                string          `json:"type"`
	Nickname            string          `json:"nickname"`
	Currency            string          `json:"currency"`
	UnitAmountInPennies int64           `json:"unit_amount_in_pennies"`
	IntervalType        string          `json:"interval_type"`
	TermLength          int32           `json:"term_length"`
	Metadata            json.RawMessage `json:"metadata"`
	CreatedAt           int64           `json:"created_at"`
	UpdatedAt           int64           `json:"updated_at"`
}

// ProductResponse represents the response structure for product data
type ProductResponse struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	ImageURL    string          `json:"image_url"`
	Active      bool            `json:"active"`
	Metadata    json.RawMessage `json:"metadata"`
}

// ToSubscriptionResponse converts a db.ListSubscriptionDetailsWithPaginationRow to a SubscriptionResponse.
func ToSubscriptionResponse(subDetails db.ListSubscriptionDetailsWithPaginationRow) (SubscriptionResponse, error) {
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

	resp := SubscriptionResponse{
		ID:                 subDetails.SubscriptionID,
		WorkspaceID:        subDetails.ProductWorkspaceID,
		Status:             string(subDetails.SubscriptionStatus),
		CurrentPeriodStart: subDetails.SubscriptionCurrentPeriodStart.Time,
		CurrentPeriodEnd:   subDetails.SubscriptionCurrentPeriodEnd.Time,
		CreatedAt:          subDetails.SubscriptionCreatedAt.Time,
		UpdatedAt:          subDetails.SubscriptionUpdatedAt.Time,
		CustomerID:         subDetails.CustomerID,
		TokenAmount:        int64(subDetails.SubscriptionTokenAmount),
		CustomerName:       subDetails.CustomerName.String,
		CustomerEmail:      subDetails.CustomerEmail.String,
		Price: PriceResponse{
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
		Product: ProductResponse{
			ID:          subDetails.ProductID.String(),
			Name:        subDetails.ProductName,
			Description: subDetails.ProductDescription.String,
			ImageURL:    subDetails.ProductImageUrl.String,
			Active:      subDetails.ProductActive,
			Metadata:    subDetails.ProductMetadata, // Expecting json.RawMessage
		},
		ProductToken: ProductTokenResponse{
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
func ToSubscriptionResponseFromDBSubscription(sub db.Subscription) (SubscriptionResponse, error) {
	resp := SubscriptionResponse{
		ID:          sub.ID,
		CustomerID:  sub.CustomerID,
		WorkspaceID: sub.WorkspaceID,
		Status:      string(sub.Status),
		TokenAmount: int64(sub.TokenAmount),
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
