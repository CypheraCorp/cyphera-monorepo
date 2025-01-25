package actalink

import (
	"encoding/json"
)

func (c *ActaLinkClient) GetTokens() (*GetTokensResponse, *int, error) {
	body, statusCode, err := c.doRequest("GET", "/api/ct/tokens", nil, nil)
	if err != nil {
		return nil, statusCode, err
	}

	var tokenResp GetTokensResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, nil, err
	}

	return &tokenResp, statusCode, nil
}
