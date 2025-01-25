package actalink

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

func (c *ActaLinkClient) CheckUserAvailability(address string) (*UserAvailabilityResponse, *int, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.billing.acta.link/api/ct/isuseravailable", nil)
	if err != nil {
		return nil, nil, err
	}

	q := req.URL.Query()
	q.Add("address", address)
	req.URL.RawQuery = q.Encode()

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

	return &UserAvailabilityResponse{
		Message: string(body),
	}, &resp.StatusCode, nil
}

func (c *ActaLinkClient) RegisterUser(request UserLoginRegisterRequest) (*RegisterUserResponse, *int, error) {
	jsonBody, err := json.Marshal(request)
	if err != nil {
		return nil, nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.billing.acta.link/api/ct/register", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

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

	var registerResp RegisterUserResponse
	if err := json.Unmarshal(body, &registerResp); err != nil {
		return nil, nil, err
	}

	return &registerResp, &resp.StatusCode, nil
}

func (c *ActaLinkClient) LoginUser(request UserLoginRegisterRequest) (*LoginUserResponse, *int, error) {
	jsonBody, err := json.Marshal(request)
	if err != nil {
		return nil, nil, err
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.billing.acta.link/api/ct/login", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

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

	var loginResp LoginUserResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return nil, nil, err
	}

	return &loginResp, &resp.StatusCode, nil
}
