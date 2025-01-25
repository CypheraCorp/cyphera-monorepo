package actalink

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

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
