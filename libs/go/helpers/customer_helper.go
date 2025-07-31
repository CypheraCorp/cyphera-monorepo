package helpers

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/cyphera/cyphera-api/libs/go/db"
	"github.com/cyphera/cyphera-api/libs/go/types/api/responses"
	"github.com/jackc/pgx/v5/pgtype"
)

// ToCustomerResponse converts database model to API response
func ToCustomerResponse(c db.Customer) responses.CustomerResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(c.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling customer metadata: %v", err)
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	return responses.CustomerResponse{
		ID:                 c.ID.String(),
		Object:             "customer",
		ExternalID:         c.ExternalID.String,
		Email:              c.Email.String,
		Name:               c.Name.String,
		Phone:              c.Phone.String,
		Description:        c.Description.String,
		FinishedOnboarding: c.FinishedOnboarding.Bool,
		Metadata:           metadata,
		CreatedAt:          c.CreatedAt.Time.Unix(),
		UpdatedAt:          c.UpdatedAt.Time.Unix(),
	}
}

// ToCustomerResponseWithRevenue converts database ListWorkspaceCustomersWithRevenueRow to API response
func ToCustomerResponseWithRevenue(c db.ListWorkspaceCustomersWithRevenueRow) responses.CustomerResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(c.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling customer metadata: %v", err)
		metadata = make(map[string]interface{}) // Use empty map if unmarshal fails
	}

	// Convert TotalRevenue interface{} to int64
	var totalRevenue int64
	if c.TotalRevenue != nil {
		switch v := c.TotalRevenue.(type) {
		case int64:
			totalRevenue = v
		case int:
			totalRevenue = int64(v)
		case string:
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				totalRevenue = parsed
			}
		case float64:
			totalRevenue = int64(v)
		case pgtype.Numeric:
			// Handle PostgreSQL numeric type
			if v.Valid {
				// Convert pgtype.Numeric to int64 using SQL driver value
				if sqlValue, err := v.Value(); err == nil {
					if intValue, ok := sqlValue.(int64); ok {
						totalRevenue = intValue
					} else if stringValue, ok := sqlValue.(string); ok {
						if parsed, err := strconv.ParseInt(stringValue, 10, 64); err == nil {
							totalRevenue = parsed
						} else {
							log.Printf("Error parsing Numeric string to int64: %v, value: %s", err, stringValue)
						}
					} else {
						log.Printf("Unexpected SQL value type from pgtype.Numeric: %T, value: %v", sqlValue, sqlValue)
					}
				} else {
					log.Printf("Error getting SQL value from pgtype.Numeric: %v", err)
				}
			}
		default:
			log.Printf("Unexpected TotalRevenue type: %T", v)
		}
	}

	return responses.CustomerResponse{
		ID:                 c.ID.String(),
		Object:             "customer",
		ExternalID:         c.ExternalID.String,
		Email:              c.Email.String,
		Name:               c.Name.String,
		Phone:              c.Phone.String,
		Description:        c.Description.String,
		FinishedOnboarding: c.FinishedOnboarding.Bool,
		Metadata:           metadata,
		TotalRevenue:       totalRevenue,
		CreatedAt:          c.CreatedAt.Time.Unix(),
		UpdatedAt:          c.UpdatedAt.Time.Unix(),
	}
}

// ToCustomerWalletResponse converts database CustomerWallet to API response
func ToCustomerWalletResponse(w db.CustomerWallet) responses.CustomerWalletResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(w.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling wallet metadata: %v", err)
		metadata = make(map[string]interface{})
	}

	return responses.CustomerWalletResponse{
		ID:            w.ID.String(),
		Object:        "customer_wallet",
		CustomerID:    w.CustomerID.String(),
		WalletAddress: w.WalletAddress,
		NetworkType:   string(w.NetworkType),
		Nickname:      w.Nickname.String,
		ENS:           w.Ens.String,
		IsPrimary:     w.IsPrimary.Bool,
		Verified:      w.Verified.Bool,
		Metadata:      metadata,
		CreatedAt:     w.CreatedAt.Time.Unix(),
		UpdatedAt:     w.UpdatedAt.Time.Unix(),
	}
}

// ToCustomerDetailsResponse creates a customer details response
func ToCustomerDetailsResponse(customer db.Customer, wallet *db.CustomerWallet) responses.CustomerDetailsResponse {
	response := responses.CustomerDetailsResponse{
		Customer: ToCustomerResponse(customer),
	}

	if wallet != nil {
		response.Wallet = ToCustomerWalletResponse(*wallet)
	}

	return response
}

// ToResponsesCustomerResponse converts helpers.CustomerResponse to responses.CustomerResponse
func ToResponsesCustomerResponse(helperResponse responses.CustomerResponse) responses.CustomerResponse {
	return responses.CustomerResponse{
		ID:                 helperResponse.ID,
		Object:             helperResponse.Object,
		ExternalID:         helperResponse.ExternalID,
		Email:              helperResponse.Email,
		Name:               helperResponse.Name,
		Phone:              helperResponse.Phone,
		Description:        helperResponse.Description,
		FinishedOnboarding: helperResponse.FinishedOnboarding,
		Metadata:           helperResponse.Metadata,
		CreatedAt:          helperResponse.CreatedAt,
		UpdatedAt:          helperResponse.UpdatedAt,
	}
}

// ToResponsesCustomerWalletResponse converts helpers.CustomerWalletResponse to responses.CustomerWalletResponse
func ToResponsesCustomerWalletResponse(helperResponse responses.CustomerWalletResponse) responses.CustomerWalletResponse {
	return responses.CustomerWalletResponse{
		ID:            helperResponse.ID,
		Object:        helperResponse.Object,
		CustomerID:    helperResponse.CustomerID,
		WalletAddress: helperResponse.WalletAddress,
		NetworkType:   helperResponse.NetworkType,
		Nickname:      helperResponse.Nickname,
		ENS:           helperResponse.ENS,
		IsPrimary:     helperResponse.IsPrimary,
		Verified:      helperResponse.Verified,
		Metadata:      helperResponse.Metadata,
		CreatedAt:     helperResponse.CreatedAt,
		UpdatedAt:     helperResponse.UpdatedAt,
	}
}

// ToResponsesCustomerDetailsResponse converts helpers.CustomerDetailsResponse to responses.CustomerDetailsResponse
func ToResponsesCustomerDetailsResponse(helperResponse responses.CustomerDetailsResponse) responses.CustomerDetailsResponse {
	return responses.CustomerDetailsResponse{
		Customer: ToResponsesCustomerResponse(helperResponse.Customer),
		Wallet:   ToResponsesCustomerWalletResponse(helperResponse.Wallet),
	}
}
