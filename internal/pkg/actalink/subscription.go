package actalink

import (
	"encoding/json"
	"fmt"
	"net/url"
)

func (c *ActaLinkClient) GetAllSubscriptions() (*GetSubscriptionsResponse, *int, error) {
	body, statusCode, err := c.doRequest("GET", "/api/ct/subscriptions", nil, nil, nil)
	if err != nil {
		return nil, statusCode, err
	}

	var subscriptions GetSubscriptionsResponse
	if err := json.Unmarshal(body, &subscriptions); err != nil {
		return nil, nil, err
	}

	return &subscriptions, statusCode, nil
}

func (c *ActaLinkClient) CreateSubscription(req SubscriptionRequest, cookie string) (*CreateSubscriptionResponse, *int, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, nil, err
	}

	headers := map[string]string{
		"X-Api-Key":    c.apiKey,
		"Content-Type": "application/json",
		"Cookie":       fmt.Sprintf("acta-session=%s", cookie),
	}

	body, statusCode, err := c.doRequest("POST", "/api/ct/newsubscription", jsonBody, nil, headers)
	if err != nil {
		return nil, statusCode, err
	}

	var createSubscriptionResponse CreateSubscriptionResponse
	if err := json.Unmarshal(body, &createSubscriptionResponse); err != nil {
		return nil, statusCode, err
	}

	return &createSubscriptionResponse, statusCode, nil
}

func (c *ActaLinkClient) DeleteSubscription(req DeleteSubscriptionRequest) (DeleteSubscriptionResponse, *int, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return DeleteSubscriptionResponse{}, nil, err
	}

	_, statusCode, err := c.doRequest("POST", "/api/ct/subscription/delete", jsonBody, nil, nil)
	if err != nil {
		return DeleteSubscriptionResponse{}, statusCode, err
	}

	return DeleteSubscriptionResponse{
		Message: fmt.Sprintf("Subscription %s deleted successfully", req.SubscriptionId),
	}, statusCode, nil
}

func (c *ActaLinkClient) GetSubscribers(subscriptionId string) (*GetSubscribersResponse, *int, error) {
	params := url.Values{}
	params.Add("subscriptionId", subscriptionId)

	body, statusCode, err := c.doRequest("GET", "/api/ct/subscribers", nil, params, nil)
	if err != nil {
		return nil, statusCode, err
	}

	var subscribers GetSubscribersResponse
	if err := json.Unmarshal(body, &subscribers); err != nil {
		return nil, nil, err
	}

	return &subscribers, statusCode, nil
}
