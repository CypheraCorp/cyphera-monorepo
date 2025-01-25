package handlers

import "cyphera-api/internal/pkg/actalink"

// Consts
const (
	UserExists = "exists"
)

// Request Types
type UserLoginRegisterRequest = actalink.UserLoginRegisterRequest
type SubscriptionRequest = actalink.SubscriptionRequest
type DeleteSubscriptionRequest = actalink.DeleteSubscriptionRequest

// Response Types
type GetUserResponse = actalink.UserAvailabilityResponse
type GetNonceResponse = actalink.GetNonceResponse
type RegisterUserResponse = actalink.RegisterUserResponse
type LoginUserResponse = actalink.LoginUserResponse
type GetSubscriptionsResponse = actalink.GetSubscriptionsResponse
type CreateSubscriptionResponse = actalink.CreateSubscriptionResponse
type DeleteSubscriptionResponse = actalink.DeleteSubscriptionResponse
type GetSubscribersResponse = actalink.GetSubscribersResponse
type GetTokensResponse = actalink.GetTokensResponse
type GetNetworksResponse = actalink.GetNetworksResponse
type OperationsResponse = actalink.OperationsResponse

type ErrorResponse struct {
	Error string `json:"error"`
}

type UserAvailabilityResponse struct {
	Exists bool `json:"exists"`
}

type HealthResponse struct {
	Status string `json:"status"`
}
