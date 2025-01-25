package actalink

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

type GetNonceResponse struct {
	Nonce string `json:"nonce"`
}

// GetNonce fetches a nonce from the Acta.link API
func (c *ActaLinkClient) GetNonce() (*GetNonceResponse, *int, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.billing.acta.link/api/ct/nonce", nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("x-api-key", c.apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, &resp.StatusCode, errors.New("unknown error occurred")
		}
		return nil, &resp.StatusCode, errors.Wrap(errors.New(errResp.Error), "actalink api error")
	}

	var nonceResp GetNonceResponse
	if err := json.Unmarshal(body, &nonceResp); err != nil {
		return nil, nil, err
	}

	return &nonceResp, &resp.StatusCode, nil
}
