package actalink

import (
	"encoding/json"
	"fmt"
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

func (c *ActaLinkClient) RegisterOrLoginUser(request UserLoginRegisterRequest, suffix string) (*RegisterOrLoginUserResponse, *int, error) {
	jsonBody, err := json.Marshal(request)
	if err != nil {
		return nil, nil, err
	}

	body, statusCode, err := c.doRequest("POST", fmt.Sprintf("/api/ct/%s", suffix), jsonBody, nil)
	if err != nil {
		return nil, statusCode, err
	}

	var resp RegisterOrLoginUserResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, err
	}

	return &resp, statusCode, nil
}
