package actalink

import (
	"encoding/json"
	"net/url"
)

func (c *ActaLinkClient) GetOperations(swAddress, subscriptionId, status string) (*OperationsResponse, *int, error) {
	params := url.Values{}
	params.Add("swaddress", swAddress)
	params.Add("subscriptionId", subscriptionId)
	params.Add("status", status)

	body, statusCode, err := c.doRequest("GET", "/api/operations", nil, params)
	if err != nil {
		return nil, statusCode, err
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

	return &operations, statusCode, nil
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
