package actalink

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
)

type ActaLinkClient struct {
	apiKey string
}

func NewActaLinkClient(apiKey string) *ActaLinkClient {
	return &ActaLinkClient{
		apiKey: apiKey,
	}
}

// doRequest handles the common HTTP request/response logic used across all ActaLink API calls
func (c *ActaLinkClient) doRequest(method, endpoint string, body []byte, queryParams url.Values) ([]byte, *int, error) {
	client := &http.Client{}

	req, err := http.NewRequest(method, "https://api.billing.acta.link"+endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, &resp.StatusCode, errors.New("unknown error occurred")
		}
		return nil, &resp.StatusCode, errors.Wrap(errors.New(errResp.Error), "actalink api error")
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

	spew.Dump(resp.Header)
	spew.Dump(resp.Cookies())

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

	spew.Dump(respBody)
	spew.Dump(resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, "", &resp.StatusCode, errors.New("unknown error occurred")
		}
		return nil, "", &resp.StatusCode, errors.Wrap(errors.New(errResp.Error), "actalink api error")
	}

	return respBody, actalinkCookie, &resp.StatusCode, nil
}
