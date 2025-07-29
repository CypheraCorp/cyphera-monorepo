package circle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	httpClient "github.com/cyphera/cyphera-api/libs/go/client/http"
	"io"
	"net/http"
	"time"
)

// ErrUserAlreadyExists is returned when Circle API indicates the user already exists (409 Conflict with code 155101)
var ErrUserAlreadyExists = errors.New("circle user already exists")

const UserAlreadyExistsErrorCode = 155101

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

// CreateUserRequest defines the payload for the Create User API call
type CreateUserRequest struct {
	UserID string `json:"userId"`
}

// CircleAPIErrorResponse represents the standard error response format from Circle API
type CircleAPIErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// GetUserByToken retrieves a user from Circle API using an auth token
func (c *CircleClient) GetUserByToken(ctx context.Context, userToken string) (*UserResponse, error) {
	resp, err := c.httpClient.Get(
		ctx,
		"users",
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by token: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
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
		fmt.Sprintf("users/%s", userID),
		httpClient.WithBearerToken(c.apiKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
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
		fmt.Sprintf("user/challenges/%s", challengeID),
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get challenge: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
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
		"user/initialize",
		request,
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	var response InitializeUserResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process initialization response: %w", err)
	}

	return &response, nil
}

// CreateUserToken generates a user session token and encryption key
func (c *CircleClient) CreateUserToken(ctx context.Context, userID string) (*UserTokenResponse, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	// Prepare the request payload
	requestPayload := CreateUserRequest{
		UserID: userID,
	}

	// The API expects the payload as a JSON object
	resp, err := c.httpClient.Post(
		ctx,
		"users/token",  // Relative path
		requestPayload, // Send the struct, httpClient will marshal it
		httpClient.WithBearerToken(c.apiKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user token: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	var response UserTokenResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process user token response: %w", err)
	}

	return &response, nil
}

// CreateUser creates a new user in the Circle system
// The userId should be your unique external identifier for the user
//
// This endpoint creates a user entity in the W3S system so the client-side SDK
// can authenticate and request wallet operations.
func (c *CircleClient) CreateUserWithPinAuth(ctx context.Context, externalUserID string) (*UserResponse, error) {
	if externalUserID == "" {
		return nil, fmt.Errorf("externalUserID is required")
	}

	// Prepare the request payload
	requestPayload := CreateUserRequest{
		UserID: externalUserID,
	}

	// Make the API request
	resp, err := c.httpClient.Post(
		ctx,
		"users", // Relative path
		requestPayload,
		httpClient.WithBearerToken(c.apiKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to execute create user request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read create user response body: %w", readErr)
	}

	if resp.StatusCode == http.StatusCreated { // 201 Success
		var successResponse UserResponse
		if err := json.Unmarshal(bodyBytes, &successResponse); err != nil {
			return nil, fmt.Errorf("failed to unmarshal successful user creation response (status %d): %w, body: %s", resp.StatusCode, err, string(bodyBytes))
		}
		return &successResponse, nil
	} else {
		// Attempt to parse as standard Circle error response regardless of status code > 201
		var errorResponse CircleAPIErrorResponse
		if jsonErr := json.Unmarshal(bodyBytes, &errorResponse); jsonErr == nil {
			// Check if it's the specific 'user already exists' error
			if resp.StatusCode == http.StatusConflict && errorResponse.Code == UserAlreadyExistsErrorCode {
				return nil, ErrUserAlreadyExists // Return the specific sentinel error
			}
			// Otherwise, return a generic Circle API error
			return nil, fmt.Errorf("circle API error (status %d): %s (code: %d)", resp.StatusCode, errorResponse.Message, errorResponse.Code)
		} else {
			// If parsing the error body fails, return a generic HTTP error, including the parsing error
			return nil, fmt.Errorf("unexpected status code from circle API: %d, failed to parse error body: %w, body: %s", resp.StatusCode, jsonErr, string(bodyBytes))
		}
	}
}
