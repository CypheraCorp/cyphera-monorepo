package circle

import (
	"context"
	"fmt"
	httpClient "github.com/cyphera/cyphera-api/libs/go/client/http"
	"time"
)

// CreateWalletsRequest represents the request to create wallets
type CreateWalletsRequest struct {
	IdempotencyKey string   `json:"idempotencyKey"`
	Blockchains    []string `json:"blockchains"`
	AccountType    string   `json:"accountType,omitempty"`
	Metadata       []struct {
		Name  string `json:"name"`
		RefID string `json:"refId"`
	} `json:"metadata,omitempty"`
}

// CreateWalletsResponse represents the response from creating wallets
type CreateWalletsResponse struct {
	Data struct {
		ChallengeID string `json:"challengeId"`
	} `json:"data"`
}

// TokenInfo represents information about a token
type TokenInfo struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Standard     string    `json:"standard"`
	Blockchain   string    `json:"blockchain"`
	Decimals     int       `json:"decimals"`
	IsNative     bool      `json:"isNative"`
	Symbol       string    `json:"symbol"`
	TokenAddress string    `json:"tokenAddress"`
	UpdateDate   time.Time `json:"updateDate"`
	CreateDate   time.Time `json:"createDate"`
}

// TokenBalance represents a token balance
type TokenBalance struct {
	Amount     string    `json:"amount"`
	Token      TokenInfo `json:"token"`
	UpdateDate time.Time `json:"updateDate"`
}

// WalletBalanceResponse represents the response from getting wallet balances
type WalletBalanceResponse struct {
	Data struct {
		TokenBalances []TokenBalance `json:"tokenBalances"`
	} `json:"data"`
}

// GetWalletBalanceParams represents optional query parameters for getting wallet balances
type GetWalletBalanceParams struct {
	IncludeAll   *bool   `json:"includeAll,omitempty"`
	Name         *string `json:"name,omitempty"`
	TokenAddress *string `json:"tokenAddress,omitempty"`
	Standard     *string `json:"standard,omitempty"`
	PageSize     *int    `json:"pageSize,omitempty"`
	PageBefore   *string `json:"pageBefore,omitempty"`
	PageAfter    *string `json:"pageAfter,omitempty"`
}

// Wallet represents a user-controlled wallet
type Wallet struct {
	ID               string    `json:"id"`
	Address          string    `json:"address"`
	Blockchain       string    `json:"blockchain"`
	CreateDate       time.Time `json:"createDate"`
	UpdateDate       time.Time `json:"updateDate"`
	CustodyType      string    `json:"custodyType"`
	Name             string    `json:"name"`
	RefID            string    `json:"refId"`
	State            string    `json:"state"`
	UserID           string    `json:"userId"`
	WalletSetID      string    `json:"walletSetId"`
	InitialPublicKey string    `json:"initialPublicKey"`
	AccountType      string    `json:"accountType"`
}

// WalletResponse represents the response from getting a wallet
type WalletResponse struct {
	Data struct {
		Wallet Wallet `json:"wallet"`
	} `json:"data"`
}

// ListWalletsResponse represents the response from listing wallets
type ListWalletsResponse struct {
	Data struct {
		Wallets []Wallet `json:"wallets"`
	} `json:"data"`
}

// ListWalletsParams represents query parameters for listing wallets
type ListWalletsParams struct {
	Address     *string    `json:"address,omitempty"`
	Blockchain  *string    `json:"blockchain,omitempty"`
	ScaCore     *string    `json:"scaCore,omitempty"`
	WalletSetID *string    `json:"walletSetId,omitempty"`
	RefID       *string    `json:"refId,omitempty"`
	From        *time.Time `json:"from,omitempty"`
	To          *time.Time `json:"to,omitempty"`
	PageBefore  *string    `json:"pageBefore,omitempty"`
	PageAfter   *string    `json:"pageAfter,omitempty"`
	PageSize    *int       `json:"pageSize,omitempty"`
}

// CreateWallets generates a challenge for creating new user-controlled wallets
func (c *CircleClient) CreateWallets(ctx context.Context, request CreateWalletsRequest, userToken string) (*CreateWalletsResponse, error) {
	// Validate blockchains
	if err := ValidateBlockchains(request.Blockchains); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(
		ctx,
		"user/wallets",
		request,
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallets: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	var response CreateWalletsResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process wallet creation response: %w", err)
	}

	return &response, nil
}

// GetWalletBalance retrieves token balances for a specific wallet
func (c *CircleClient) GetWalletBalance(ctx context.Context, walletID string, userToken string, params *GetWalletBalanceParams) (*WalletBalanceResponse, error) {
	// Build the request with options
	options := []httpClient.RequestOption{
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	}

	// Add optional query parameters if provided
	if params != nil {
		if params.IncludeAll != nil && *params.IncludeAll {
			options = append(options, httpClient.WithQueryParam("includeAll", "true"))
		}
		if params.Name != nil && *params.Name != "" {
			options = append(options, httpClient.WithQueryParam("name", *params.Name))
		}
		if params.TokenAddress != nil && *params.TokenAddress != "" {
			options = append(options, httpClient.WithQueryParam("tokenAddress", *params.TokenAddress))
		}
		if params.Standard != nil && *params.Standard != "" {
			options = append(options, httpClient.WithQueryParam("standard", *params.Standard))
		}
		if params.PageSize != nil {
			options = append(options, httpClient.WithQueryParam("pageSize", fmt.Sprintf("%d", *params.PageSize)))
		}
		if params.PageBefore != nil && *params.PageBefore != "" {
			options = append(options, httpClient.WithQueryParam("pageBefore", *params.PageBefore))
		}
		if params.PageAfter != nil && *params.PageAfter != "" {
			options = append(options, httpClient.WithQueryParam("pageAfter", *params.PageAfter))
		}
	}

	// Make the API request
	resp, err := c.httpClient.Get(
		ctx,
		fmt.Sprintf("wallets/%s/balances", walletID),
		options...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet balances: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	var response WalletBalanceResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process wallet balance response: %w", err)
	}

	return &response, nil
}

// GetWallet retrieves details about a specific wallet by its ID
func (c *CircleClient) GetWallet(ctx context.Context, walletID string, userToken string) (*WalletResponse, error) {
	resp, err := c.httpClient.Get(
		ctx,
		fmt.Sprintf("wallets/%s", walletID),
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	var response WalletResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process wallet response: %w", err)
	}

	return &response, nil
}

// ListWallets retrieves a list of wallets that match the specified parameters
func (c *CircleClient) ListWallets(ctx context.Context, userToken string, params *ListWalletsParams) (*ListWalletsResponse, error) {
	// Build the request with options
	options := []httpClient.RequestOption{
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	}

	// Add optional query parameters if provided
	if params != nil {
		if params.Address != nil && *params.Address != "" {
			options = append(options, httpClient.WithQueryParam("address", *params.Address))
		}
		if params.Blockchain != nil && *params.Blockchain != "" {
			options = append(options, httpClient.WithQueryParam("blockchain", *params.Blockchain))
		}
		if params.ScaCore != nil && *params.ScaCore != "" {
			options = append(options, httpClient.WithQueryParam("scaCore", *params.ScaCore))
		}
		if params.WalletSetID != nil && *params.WalletSetID != "" {
			options = append(options, httpClient.WithQueryParam("walletSetId", *params.WalletSetID))
		}
		if params.RefID != nil && *params.RefID != "" {
			options = append(options, httpClient.WithQueryParam("refId", *params.RefID))
		}
		if params.From != nil {
			options = append(options, httpClient.WithQueryParam("from", params.From.Format(time.RFC3339)))
		}
		if params.To != nil {
			options = append(options, httpClient.WithQueryParam("to", params.To.Format(time.RFC3339)))
		}
		if params.PageSize != nil {
			options = append(options, httpClient.WithQueryParam("pageSize", fmt.Sprintf("%d", *params.PageSize)))
		}
		if params.PageBefore != nil && *params.PageBefore != "" {
			options = append(options, httpClient.WithQueryParam("pageBefore", *params.PageBefore))
		}
		if params.PageAfter != nil && *params.PageAfter != "" {
			options = append(options, httpClient.WithQueryParam("pageAfter", *params.PageAfter))
		}
	}

	// Make the API request
	resp, err := c.httpClient.Get(
		ctx,
		"wallets",
		options...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list wallets: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	var response ListWalletsResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process wallet list response: %w", err)
	}

	return &response, nil
}
