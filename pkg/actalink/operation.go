package actalink

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

func (c *ActaLinkClient) GetOperations(swAddress, subscriptionId, status string) (*OperationsResponse, *int, error) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", "https://api.billing.acta.link/api/operations", nil)
	if err != nil {
		return nil, nil, err
	}

	q := request.URL.Query()
	q.Add("swaddress", swAddress)
	q.Add("subscriptionId", subscriptionId)
	q.Add("status", status)
	request.URL.RawQuery = q.Encode()

	request.Header.Set("x-api-key", c.apiKey)

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

	var rawResponse struct {
		Data []RawOperation `json:"data"`
	}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, nil, err
	}

	operations := OperationsResponse{
		Data: make([]Operation, 0, len(rawResponse.Data)),
	}

	for _, rawOp := range rawResponse.Data {
		op, err := transformOperation(rawOp)
		if err != nil {
			return nil, nil, err
		}
		operations.Data = append(operations.Data, op)
	}

	return &operations, &resp.StatusCode, nil
}

func transformOperation(raw RawOperation) (Operation, error) {
	var userOp UserOperation
	if err := json.Unmarshal([]byte(raw.UserOp), &userOp); err != nil {
		return Operation{}, err
	}

	var paymentParams PaymentTypeParams
	if err := json.Unmarshal([]byte(raw.PaymentTypeParams), &paymentParams); err != nil {
		return Operation{}, err
	}

	return Operation{
		UserOpHash:        raw.UserOpHash,
		UserOp:            userOp,
		EntryPoint:        raw.EntryPoint,
		ExecutionTime:     raw.ExecutionTime,
		PaymentType:       raw.PaymentType,
		PaymentTypeParams: paymentParams,
		Status:            raw.Status,
		TransactionHash:   raw.TransactionHash,
	}, nil
}
