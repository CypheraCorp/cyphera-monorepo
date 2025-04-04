package circle

import (
	"context"
	httpClient "cyphera-api/internal/client/http"
	"fmt"
)

// PinChallengeResponse represents the response when creating a PIN setup challenge
type PinChallengeResponse struct {
	Data struct {
		ChallengeID string `json:"challengeId"`
	} `json:"data"`
}

// CreatePinChallenge creates a challenge for PIN setup without setting up wallets
func (c *CircleClient) CreatePinChallenge(ctx context.Context, idempotencyKey string, userToken string) (*PinChallengeResponse, error) {
	// The API expects the idempotencyKey as a JSON string in the request body
	resp, err := c.httpClient.Post(
		ctx,
		"/user/pin",
		idempotencyKey, // Send the idempotencyKey directly as the body
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create PIN challenge: %w", err)
	}

	var response PinChallengeResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process PIN challenge response: %w", err)
	}

	return &response, nil
}

// UpdatePinChallenge creates a challenge to update a user's PIN via the existing PIN
func (c *CircleClient) UpdatePinChallenge(ctx context.Context, idempotencyKey string, userToken string) (*PinChallengeResponse, error) {
	// The API expects the idempotencyKey as a JSON string in the request body
	resp, err := c.httpClient.Put(
		ctx,
		"/user/pin",
		idempotencyKey, // Send the idempotencyKey directly as the body
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create PIN update challenge: %w", err)
	}

	var response PinChallengeResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process PIN update challenge response: %w", err)
	}

	return &response, nil
}

// CreatePinRestoreChallenge creates a challenge to change a user's PIN via Security Questions
func (c *CircleClient) CreatePinRestoreChallenge(ctx context.Context, idempotencyKey string, userToken string) (*PinChallengeResponse, error) {
	// The API expects the idempotencyKey as a JSON string in the request body
	resp, err := c.httpClient.Post(
		ctx,
		"/user/pin/restore",
		idempotencyKey, // Send the idempotencyKey directly as the body
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create PIN restore challenge: %w", err)
	}

	var response PinChallengeResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process PIN restore challenge response: %w", err)
	}

	return &response, nil
}
