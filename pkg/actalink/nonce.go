package actalink

import (
	"encoding/json"
)

type GetNonceResponse struct {
	Nonce string `json:"nonce"`
}

// GetNonce fetches a nonce from the Acta.link API
func (c *ActaLinkClient) GetNonce() (*GetNonceResponse, *int, error) {
	body, statusCode, err := c.doRequest("GET", "/api/ct/nonce", nil, nil)
	if err != nil {
		return nil, statusCode, err
	}

	var nonceResp GetNonceResponse
	if err := json.Unmarshal(body, &nonceResp); err != nil {
		return nil, nil, err
	}

	return &nonceResp, statusCode, nil
}
