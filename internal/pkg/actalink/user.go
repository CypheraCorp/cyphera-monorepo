package actalink

import (
	"encoding/json"
	"net/url"
)

func (c *ActaLinkClient) CheckUserAvailability(address string) (*UserAvailabilityResponse, *int, error) {
	params := url.Values{}
	params.Add("address", address)

	body, statusCode, err := c.doRequest("GET", "/api/ct/isuseravailable", nil, params)
	if err != nil {
		return nil, statusCode, err
	}

	return &UserAvailabilityResponse{
		Message: string(body),
	}, statusCode, nil
}

func (c *ActaLinkClient) RegisterUser(request UserLoginRegisterRequest) (*RegisterUserResponse, *int, error) {
	jsonBody, err := json.Marshal(request)
	if err != nil {
		return nil, nil, err
	}

	body, statusCode, err := c.doRequest("POST", "/api/ct/register", jsonBody, nil)
	if err != nil {
		return nil, statusCode, err
	}

	var registerResp RegisterUserResponse
	if err := json.Unmarshal(body, &registerResp); err != nil {
		return nil, nil, err
	}

	return &registerResp, statusCode, nil
}

func (c *ActaLinkClient) LoginUser(request UserLoginRegisterRequest) (*LoginUserResponse, *int, error) {
	jsonBody, err := json.Marshal(request)
	if err != nil {
		return nil, nil, err
	}

	body, statusCode, err := c.doRequest("POST", "/api/ct/login", jsonBody, nil)
	if err != nil {
		return nil, statusCode, err
	}

	var loginResp LoginUserResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return nil, nil, err
	}

	return &loginResp, statusCode, nil
}
