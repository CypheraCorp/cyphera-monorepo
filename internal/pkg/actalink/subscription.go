package actalink

import (
	"encoding/json"
	"fmt"
	"net/url"
)

func (c *ActaLinkClient) GetAllSubscriptions() (*GetSubscriptionsResponse, *int, error) {
	body, statusCode, err := c.doRequest("GET", "/api/subscription", nil, nil)
	if err != nil {
		return nil, statusCode, err
	}

	var subscriptions GetSubscriptionsResponse
	if err := json.Unmarshal(body, &subscriptions); err != nil {
		return nil, nil, err
	}

	return &subscriptions, statusCode, nil
}

func (c *ActaLinkClient) CreateSubscription(req SubscriptionRequest) (*CreateSubscriptionResponse, *int, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, nil, err
	}

	_, statusCode, err := c.doRequest("POST", "/api/newsubscription", jsonBody, nil)
	if err != nil {
		return nil, statusCode, err
	}

	return &CreateSubscriptionResponse{
		Message: "Subscription(s) created successfully",
	}, statusCode, nil
}

func (c *ActaLinkClient) DeleteSubscription(req DeleteSubscriptionRequest) (DeleteSubscriptionResponse, *int, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return DeleteSubscriptionResponse{}, nil, err
	}

	_, statusCode, err := c.doRequest("POST", "/api/ct/subscription/delete", jsonBody, nil)
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

	body, statusCode, err := c.doRequest("GET", "/api/ct/subscribers", nil, params)
	if err != nil {
		return nil, statusCode, err
	}

	var subscribers GetSubscribersResponse
	if err := json.Unmarshal(body, &subscribers); err != nil {
		return nil, nil, err
	}

	return &subscribers, statusCode, nil
}
