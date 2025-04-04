package circle

import (
	"context"
	httpClient "cyphera-api/internal/client/http"
	"fmt"
	"time"
)

// UserResponse represents the response from the Circle API when getting a user
type UserResponse struct {
	Data struct {
		ID                     string    `json:"id"`
		CreateDate             time.Time `json:"createDate"`
		PinStatus              string    `json:"pinStatus"`
		Status                 string    `json:"status"`
		SecurityQuestionStatus string    `json:"securityQuestionStatus"`
		PinDetails             struct {
			FailedAttempts       int    `json:"failedAttempts"`
			LockedDate           string `json:"lockedDate"`
			LockedExpiryDate     string `json:"lockedExpiryDate"`
			LastLockOverrideDate string `json:"lastLockOverrideDate"`
		} `json:"pinDetails"`
		SecurityQuestionDetails struct {
			FailedAttempts       int    `json:"failedAttempts"`
			LockedDate           string `json:"lockedDate"`
			LockedExpiryDate     string `json:"lockedExpiryDate"`
			LastLockOverrideDate string `json:"lastLockOverrideDate"`
		} `json:"securityQuestionDetails"`
	} `json:"data"`
}

// UserByIDResponse represents the response from the Circle API when getting a user by ID
type UserByIDResponse struct {
	Data struct {
		User struct {
			ID                     string    `json:"id"`
			CreateDate             time.Time `json:"createDate"`
			PinStatus              string    `json:"pinStatus"`
			Status                 string    `json:"status"`
			SecurityQuestionStatus string    `json:"securityQuestionStatus"`
			PinDetails             struct {
				FailedAttempts       int    `json:"failedAttempts"`
				LockedDate           string `json:"lockedDate"`
				LockedExpiryDate     string `json:"lockedExpiryDate"`
				LastLockOverrideDate string `json:"lastLockOverrideDate"`
			} `json:"pinDetails"`
			SecurityQuestionDetails struct {
				FailedAttempts       int    `json:"failedAttempts"`
				LockedDate           string `json:"lockedDate"`
				LockedExpiryDate     string `json:"lockedExpiryDate"`
				LastLockOverrideDate string `json:"lastLockOverrideDate"`
			} `json:"securityQuestionDetails"`
		} `json:"user"`
	} `json:"data"`
}

// ChallengeResponse represents the response from the Circle API when getting a challenge
type ChallengeResponse struct {
	Data struct {
		Challenge struct {
			ID             string   `json:"id"`
			CorrelationIds []string `json:"correlationIds"`
			ErrorCode      int      `json:"errorCode"`
			ErrorMessage   string   `json:"errorMessage"`
			Status         string   `json:"status"`
			Type           string   `json:"type"`
		} `json:"challenge"`
	} `json:"data"`
}

// InitializeUserRequest represents the request to initialize a user and create wallets
type InitializeUserRequest struct {
	IdempotencyKey string   `json:"idempotencyKey"`
	AccountType    string   `json:"accountType,omitempty"`
	Blockchains    []string `json:"blockchains"`
	Metadata       []struct {
		Name  string `json:"name"`
		RefID string `json:"refId"`
	} `json:"metadata,omitempty"`
}

// InitializeUserResponse represents the response from initializing a user
type InitializeUserResponse struct {
	Data struct {
		ChallengeID string `json:"challengeId"`
	} `json:"data"`
}

// UserTokenResponse represents the response from creating a user token
type UserTokenResponse struct {
	Data struct {
		UserToken     string `json:"userToken"`
		EncryptionKey string `json:"encryptionKey"`
	} `json:"data"`
}

// GetUserByToken retrieves a user from Circle API using an auth token
func (c *CircleClient) GetUserByToken(ctx context.Context, userToken string) (*UserResponse, error) {
	resp, err := c.httpClient.Get(
		ctx,
		"/users",
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by token: %w", err)
	}

	var userResponse UserResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &userResponse); err != nil {
		return nil, fmt.Errorf("failed to process user response: %w", err)
	}

	return &userResponse, nil
}

// GetUserByID retrieves a user from Circle API by their ID
func (c *CircleClient) GetUserByID(ctx context.Context, userID string) (*UserByIDResponse, error) {
	resp, err := c.httpClient.Get(
		ctx,
		fmt.Sprintf("/users/%s", userID),
		httpClient.WithBearerToken(c.apiKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	var userResponse UserByIDResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &userResponse); err != nil {
		return nil, fmt.Errorf("failed to process user response: %w", err)
	}

	return &userResponse, nil
}

// GetChallenge retrieves a challenge from Circle API by its ID
func (c *CircleClient) GetChallenge(ctx context.Context, challengeID string, userToken string) (*ChallengeResponse, error) {
	resp, err := c.httpClient.Get(
		ctx,
		fmt.Sprintf("/user/challenges/%s", challengeID),
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenge: %w", err)
	}

	var challengeResponse ChallengeResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &challengeResponse); err != nil {
		return nil, fmt.Errorf("failed to process challenge response: %w", err)
	}

	return &challengeResponse, nil
}

// InitializeUser creates a challenge for user initialization and creates wallet(s)
func (c *CircleClient) InitializeUser(ctx context.Context, request InitializeUserRequest, userToken string) (*InitializeUserResponse, error) {
	// Validate blockchains
	if err := ValidateBlockchains(request.Blockchains); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(
		ctx,
		"/user/initialize",
		request,
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user: %w", err)
	}

	var response InitializeUserResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process initialization response: %w", err)
	}

	return &response, nil
}

// CreateUserToken generates a user session token and encryption key
func (c *CircleClient) CreateUserToken(ctx context.Context, userID string) (*UserTokenResponse, error) {
	// The API expects the userID as a JSON string in the request body
	resp, err := c.httpClient.Post(
		ctx,
		"/users/token",
		userID, // Send the userID directly as the body
		httpClient.WithBearerToken(c.apiKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user token: %w", err)
	}

	var response UserTokenResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process user token response: %w", err)
	}

	return &response, nil
}
