package actalink

type GetNonceResponse struct {
	Nonce string `json:"nonce"`
}

// GetNonce fetches a nonce from the Acta.link API
func (c *ActaLinkClient) GetNonce() (*GetNonceResponse, *int, error) {
	body, statusCode, err := c.doRequest("GET", "/api/ct/nonce", nil, nil, nil)
	if err != nil {
		return nil, statusCode, err
	}

	nonceResp := GetNonceResponse{
		Nonce: string(body),
	}

	return &nonceResp, statusCode, nil
}
