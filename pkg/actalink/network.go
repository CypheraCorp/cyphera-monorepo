package actalink

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

func (c *ActaLinkClient) GetNetworks() (*GetNetworksResponse, *int, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.billing.acta.link/api/ct/networks", nil)
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

	var networkResp GetNetworksResponse
	if err := json.Unmarshal(body, &networkResp); err != nil {
		return nil, nil, err
	}

	return &networkResp, &resp.StatusCode, nil
}
