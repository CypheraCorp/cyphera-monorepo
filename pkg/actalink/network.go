package actalink

import (
	"encoding/json"
)

func (c *ActaLinkClient) GetNetworks() (*GetNetworksResponse, *int, error) {
	body, statusCode, err := c.doRequest("GET", "/api/ct/networks", nil, nil)
	if err != nil {
		return nil, statusCode, err
	}

	var networkResp GetNetworksResponse
	if err := json.Unmarshal(body, &networkResp); err != nil {
		return nil, nil, err
	}

	return &networkResp, statusCode, nil
}
