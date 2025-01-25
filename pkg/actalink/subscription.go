package actalink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

func (c *ActaLinkClient) GetAllSubscriptions() (*GetSubscriptionsResponse, *int, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.billing.acta.link/api/subscription", nil)
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

	var subscriptions GetSubscriptionsResponse
	if err := json.Unmarshal(body, &subscriptions); err != nil {
		return nil, nil, err
	}

	return &subscriptions, &resp.StatusCode, nil
}

func (c *ActaLinkClient) CreateSubscription(req SubscriptionRequest) (*CreateSubscriptionResponse, *int, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, nil, err
	}

	client := &http.Client{}
	request, err := http.NewRequest("POST", "https://api.billing.acta.link/api/newsubscription", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, nil, err
	}

	request.Header.Set("x-api-key", c.apiKey)
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
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

	return &CreateSubscriptionResponse{
		Message: "Subscription(s) created successfully",
	}, &resp.StatusCode, nil
}

func (c *ActaLinkClient) DeleteSubscription(req DeleteSubscriptionRequest) (DeleteSubscriptionResponse, *int, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return DeleteSubscriptionResponse{}, nil, err
	}

	client := &http.Client{}
	request, err := http.NewRequest("POST", "https://api.billing.acta.link/api/ct/subscription/delete", bytes.NewBuffer(jsonBody))
	if err != nil {
		return DeleteSubscriptionResponse{}, nil, err
	}

	request.Header.Set("x-api-key", c.apiKey)
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return DeleteSubscriptionResponse{}, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return DeleteSubscriptionResponse{}, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			return DeleteSubscriptionResponse{}, &resp.StatusCode, errors.New("unknown error occurred")
		}
		return DeleteSubscriptionResponse{}, &resp.StatusCode, errors.Wrap(errors.New(errResp.Error), "actalink api error")
	}

	return DeleteSubscriptionResponse{
		Message: fmt.Sprintf("Subscription %s deleted successfully", req.SubscriptionId),
	}, &resp.StatusCode, nil
}

func (c *ActaLinkClient) GetSubscribers(subscriptionId string) (*GetSubscribersResponse, *int, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.billing.acta.link/api/ct/subscribers", nil)
	if err != nil {
		return nil, nil, err
	}

	q := req.URL.Query()
	q.Add("subscriptionId", subscriptionId)
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

	var subscribers GetSubscribersResponse
	if err := json.Unmarshal(body, &subscribers); err != nil {
		return nil, nil, err
	}

	return &subscribers, &resp.StatusCode, nil
}
