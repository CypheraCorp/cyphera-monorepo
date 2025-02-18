package actalink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
)

type ActaLinkClient struct {
	apiKey                  string
	CypheraWalletPrivateKey string
}

func NewActaLinkClient(apiKey string, cypheraWalletPrivateKey string) *ActaLinkClient {
	return &ActaLinkClient{
		apiKey:                  apiKey,
		CypheraWalletPrivateKey: cypheraWalletPrivateKey,
	}
}

// doRequest handles the common HTTP request/response logic used across all ActaLink API calls
func (c *ActaLinkClient) doRequest(method, endpoint string, body []byte, queryParams url.Values, headers map[string]string) ([]byte, *int, error) {
	client := &http.Client{}

	fullURL := "https://api.billing.acta.link" + endpoint
	fmt.Printf("Making request: %s %s\n", method, fullURL)

	req, err := http.NewRequest(method, fullURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	if queryParams != nil {
		req.URL.RawQuery = queryParams.Encode()
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)

	// Set custom headers after default headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	fmt.Printf("Final Request Headers: %+v\n", req.Header)
	fmt.Printf("Request Body: %s\n", string(body))

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error from Do(): %v\n", err)
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Printf("Response Headers: %+v\n", resp.Header)
	fmt.Printf("Response Body: %s\n", string(respBody))

	// Handle non-200 status codes
	if resp.StatusCode != http.StatusOK {
		statusCode := resp.StatusCode
		// Try to parse error response
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			// If we can't parse the error, return the raw response
			return respBody, &statusCode, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
		}
		return nil, &statusCode, fmt.Errorf("request failed: %s", errResp.Error)
	}

	return respBody, &resp.StatusCode, nil
}

// doRequest handles the common HTTP request/response logic used across all ActaLink API calls
func (c *ActaLinkClient) doRequestWithCookies(method, endpoint string, body []byte, queryParams url.Values) ([]byte, string, *int, error) {
	client := &http.Client{}

	req, err := http.NewRequest(method, "https://api.billing.acta.link"+endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, "", nil, err
	}

	if queryParams != nil {
		req.URL.RawQuery = queryParams.Encode()
	}

	req.Header.Set("x-api-key", c.apiKey)
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", nil, err
	}
	defer resp.Body.Close()

	cookies := resp.Cookies()
	actalinkCookie := ""
	for _, cookie := range cookies {
		if cookie.Name == "acta-session" {
			actalinkCookie = cookie.Value
		}
	}

	// if actalinkCookie == "" {
	// 	return nil, "", nil, errors.New("actalink cookie not found")
	// }

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, "", &resp.StatusCode, errors.New("unknown error occurred")
		}
		spew.Dump(errResp)
		return nil, "", &resp.StatusCode, errors.Wrap(errors.New(errResp.Error), "actalink api error")
	}

	return respBody, actalinkCookie, &resp.StatusCode, nil
}
