package actalink

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/davecgh/go-spew/spew"
)

func (c *ActaLinkClient) CheckUserAvailability(address string) (*UserAvailabilityResponse, *int, error) {
	params := url.Values{}
	params.Add("address", address)

	resp, statusCode, err := c.doRequest("GET", "/api/ct/isuseravailable", nil, params, nil)
	if err != nil {
		return nil, statusCode, err
	}

	var respBody UserAvailabilityResponse
	if err := json.Unmarshal(resp, &respBody); err != nil {
		return nil, statusCode, err
	}

	return &respBody, statusCode, nil
}

func (c *ActaLinkClient) RegisterOrLoginUser(request UserLoginRegisterRequest, suffix string) (*RegisterOrLoginUserResponse, *int, error) {

	spew.Dump(request)
	jsonBody, err := json.Marshal(request)
	if err != nil {
		return nil, nil, err
	}

	body, cookie, statusCode, err := c.doRequestWithCookies("POST", fmt.Sprintf("/api/ct/%s", suffix), jsonBody, nil)
	if err != nil {
		return nil, statusCode, err
	}

	var resp RegisterOrLoginUserResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, nil, err
	}

	resp.Cookie = cookie

	return &resp, statusCode, nil
}
