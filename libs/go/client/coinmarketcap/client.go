package coinmarketcap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	httpClient "github.com/cyphera/cyphera-api/libs/go/client/http"
	"github.com/cyphera/cyphera-api/libs/go/logger"

	"go.uber.org/zap"
)

const (
	defaultBaseURL = "https://pro-api.coinmarketcap.com"
	defaultTimeout = 10 * time.Second
)

// Client manages communication with the CoinMarketCap API.
type Client struct {
	apiKey     string
	httpClient *httpClient.HTTPClient
	baseURL    string
}

// NewClient creates a new CoinMarketCap API client.
func NewClient(apiKey string) *Client {
	if apiKey == "" {
		// Consider returning an error or logging a warning
		// For now, let it proceed but calls will fail without a key
	}

	httpClient := httpClient.NewHTTPClient(
		httpClient.WithBaseURL(defaultBaseURL),
	)
	return &Client{
		apiKey:     apiKey,
		httpClient: httpClient,
		baseURL:    defaultBaseURL,
	}
}

// --- CMC API Response Structs ---
// These structs match the expected JSON structure from CMC API.
// Adjust based on the specific endpoint (v1 vs v2) and fields needed.

type CmcQuote struct {
	Price            float64 `json:"price"`
	Volume24h        float64 `json:"volume_24h"`
	PercentChange1h  float64 `json:"percent_change_1h"`
	PercentChange24h float64 `json:"percent_change_24h"`
	PercentChange7d  float64 `json:"percent_change_7d"`
	MarketCap        float64 `json:"market_cap"`
	LastUpdated      string  `json:"last_updated"` // Consider parsing to time.Time if needed
}

type CmcQuoteMap map[string]CmcQuote // Keyed by fiat symbol (e.g., "USD")

type CmcTokenData struct {
	ID          int         `json:"id"`
	Name        string      `json:"name"`
	Symbol      string      `json:"symbol"`
	Slug        string      `json:"slug"`
	LastUpdated string      `json:"last_updated"`
	Quote       CmcQuoteMap `json:"quote"`
	// Add other fields like CirculatingSupply, MaxSupply etc. if needed
}

// V2 uses an array even for a single symbol query
type CmcResponseData map[string][]CmcTokenData // Keyed by token symbol (e.g., "BTC")

type CmcStatus struct {
	Timestamp    string `json:"timestamp"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Elapsed      int    `json:"elapsed"`
	CreditCount  int    `json:"credit_count"`
	// Notice string `json:"notice"` // Might be null or string
}

type CmcAPIResponse struct {
	Status CmcStatus       `json:"status"`
	Data   CmcResponseData `json:"data"`
}

// Error represents an API error returned by CoinMarketCap.
type Error struct {
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	return fmt.Sprintf("CoinMarketCap API Error: Status %d, Message: %s", e.StatusCode, e.Message)
}

// GetLatestQuotes fetches the latest quotes for given token symbols.
func (c *Client) GetLatestQuotes(tokenSymbols []string, convertSymbols []string) (*CmcAPIResponse, error) {
	if len(tokenSymbols) == 0 {
		return nil, fmt.Errorf("tokenSymbols cannot be empty")
	}

	// Use V2 endpoint path
	endpointPath := "/v2/cryptocurrency/quotes/latest"

	// Prepare request options using the httpclient package functions
	requestOptions := []httpClient.RequestOption{
		httpClient.WithQueryParam("symbol", strings.ToUpper(strings.Join(tokenSymbols, ","))),
		httpClient.WithHeader("X-CMC_PRO_API_KEY", c.apiKey),
		// "Accepts: application/json" is likely a default header in HTTPClient,
		// but can be explicitly set if needed:
		// httpclient.WithHeader("Accepts", "application/json"),
	}

	if len(convertSymbols) > 0 {
		requestOptions = append(requestOptions, httpClient.WithQueryParam("convert", strings.ToUpper(strings.Join(convertSymbols, ","))))
	}

	// Make the GET request using the HTTPClient's Get method
	resp, err := c.httpClient.Get(context.Background(), endpointPath, requestOptions...)
	if err != nil {
		// The HTTPClient already logs details, so we just wrap the error
		logger.Error("CoinMarketCap API request failed", zap.Error(err))
		return nil, fmt.Errorf("failed to get latest quotes from CoinMarketCap: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP errors returned by the client's DoRequest/ProcessJSONResponse logic
	// Note: If resp is non-nil, it might still contain an HTTPError in the err return value from DoRequest.
	// However, the current HTTPClient returns the HTTPError as the error value itself when status >= 400.
	// If we were using ProcessJSONResponse, it would handle this check internally.
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		// Re-wrap the error for more context if needed, although HTTPClient already provides details
		httpErr := &httpClient.HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			URL:        resp.Request.URL.String(), // Get the actual URL used
			Method:     resp.Request.Method,
			Body:       string(bodyBytes),
		}
		logger.Error("CoinMarketCap API returned an error status", zap.Int("status", resp.StatusCode), zap.String("body", string(bodyBytes)))
		return nil, fmt.Errorf("CoinMarketCap API error: %w", httpErr)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read CoinMarketCap response body", zap.Error(err))
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for non-200 status code first
	if resp.StatusCode != http.StatusOK {
		// Attempt to parse the error message from CMC if possible
		var errResp CmcAPIResponse // Use the main struct to potentially capture status error
		_ = json.Unmarshal(body, &errResp)
		errMsg := fmt.Sprintf("status code %d", resp.StatusCode)
		if errResp.Status.ErrorMessage != "" {
			errMsg = errResp.Status.ErrorMessage
		}
		return nil, &Error{
			StatusCode: resp.StatusCode,
			Message:    errMsg,
		}
	}

	// Parse the successful response
	var apiResponse CmcAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w. Body: %s", err, string(body))
	}

	// Check for error code within the successful (200 OK) response status
	if apiResponse.Status.ErrorCode != 0 {
		return nil, &Error{
			StatusCode: resp.StatusCode, // Still 200, but logical error
			Message:    fmt.Sprintf("API Error %d: %s", apiResponse.Status.ErrorCode, apiResponse.Status.ErrorMessage),
		}
	}

	return &apiResponse, nil
}
