package helpers

import (
	"encoding/json"
	"log"

	"github.com/cyphera/cyphera-api/libs/go/db"
)

// UserResponse represents the standardized API response for user operations
type UserResponse struct {
	ID                 string                 `json:"id"`
	Object             string                 `json:"object"`
	Web3AuthID         string                 `json:"web3auth_id,omitempty"`
	Verifier           string                 `json:"verifier,omitempty"`
	VerifierID         string                 `json:"verifier_id,omitempty"`
	Email              string                 `json:"email"`
	FirstName          string                 `json:"first_name,omitempty"`
	LastName           string                 `json:"last_name,omitempty"`
	AddressLine1       string                 `json:"address_line_1,omitempty"`
	AddressLine2       string                 `json:"address_line_2,omitempty"`
	City               string                 `json:"city,omitempty"`
	StateRegion        string                 `json:"state_region,omitempty"`
	PostalCode         string                 `json:"postal_code,omitempty"`
	Country            string                 `json:"country,omitempty"`
	DisplayName        string                 `json:"display_name,omitempty"`
	PictureURL         string                 `json:"picture_url,omitempty"`
	Phone              string                 `json:"phone,omitempty"`
	Timezone           string                 `json:"timezone,omitempty"`
	Locale             string                 `json:"locale,omitempty"`
	EmailVerified      bool                   `json:"email_verified"`
	TwoFactorEnabled   bool                   `json:"two_factor_enabled"`
	FinishedOnboarding bool                   `json:"finished_onboarding"`
	Status             string                 `json:"status"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          int64                  `json:"created_at"`
	UpdatedAt          int64                  `json:"updated_at"`
}

// UserAccountResponse represents a user's relationship with an account
type UserAccountResponse struct {
	UserResponse
	AccountName string `json:"account_name"`
	Role        string `json:"role"`
	IsOwner     bool   `json:"is_owner"`
}

// ToUserResponse converts database model to API response
func ToUserResponse(u db.User) UserResponse {
	var metadata map[string]interface{}
	if err := json.Unmarshal(u.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling user metadata: %v", err)
		metadata = make(map[string]interface{})
	}

	var status string
	if u.Status.Valid {
		status = string(u.Status.UserStatus)
	}

	return UserResponse{
		ID:                 u.ID.String(),
		Object:             "user",
		Web3AuthID:         u.Web3authID.String,
		Verifier:           u.Verifier.String,
		VerifierID:         u.VerifierID.String,
		Email:              u.Email,
		FirstName:          u.FirstName.String,
		LastName:           u.LastName.String,
		AddressLine1:       u.AddressLine1.String,
		AddressLine2:       u.AddressLine2.String,
		City:               u.City.String,
		StateRegion:        u.StateRegion.String,
		PostalCode:         u.PostalCode.String,
		Country:            u.Country.String,
		DisplayName:        u.DisplayName.String,
		PictureURL:         u.PictureUrl.String,
		Phone:              u.Phone.String,
		Timezone:           u.Timezone.String,
		Locale:             u.Locale.String,
		EmailVerified:      u.EmailVerified.Bool,
		TwoFactorEnabled:   u.TwoFactorEnabled.Bool,
		FinishedOnboarding: u.FinishedOnboarding.Bool,
		Status:             status,
		Metadata:           metadata,
		CreatedAt:          u.CreatedAt.Time.Unix(),
		UpdatedAt:          u.UpdatedAt.Time.Unix(),
	}
}

// ToUserAccountResponse converts GetUserAccountRow to API response
func ToUserAccountResponse(u db.GetUserAccountRow) UserAccountResponse {
	userResponse := UserResponse{
		ID:                 u.ID.String(),
		Object:             "user",
		Web3AuthID:         u.Web3authID.String,
		Verifier:           u.Verifier.String,
		VerifierID:         u.VerifierID.String,
		Email:              u.Email,
		FirstName:          u.FirstName.String,
		LastName:           u.LastName.String,
		AddressLine1:       u.AddressLine1.String,
		AddressLine2:       u.AddressLine2.String,
		City:               u.City.String,
		StateRegion:        u.StateRegion.String,
		PostalCode:         u.PostalCode.String,
		Country:            u.Country.String,
		DisplayName:        u.DisplayName.String,
		PictureURL:         u.PictureUrl.String,
		Phone:              u.Phone.String,
		Timezone:           u.Timezone.String,
		Locale:             u.Locale.String,
		EmailVerified:      u.EmailVerified.Bool,
		TwoFactorEnabled:   u.TwoFactorEnabled.Bool,
		FinishedOnboarding: u.FinishedOnboarding.Bool,
		Status:             string(u.Status.UserStatus),
		CreatedAt:          u.CreatedAt.Time.Unix(),
		UpdatedAt:          u.UpdatedAt.Time.Unix(),
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(u.Metadata, &metadata); err != nil {
		log.Printf("Error unmarshaling user metadata: %v", err)
		metadata = make(map[string]interface{})
	}
	userResponse.Metadata = metadata

	return UserAccountResponse{
		UserResponse: userResponse,
		AccountName:  u.AccountName,
		Role:         string(u.Role),
		IsOwner:      u.IsAccountOwner.Bool,
	}
}
