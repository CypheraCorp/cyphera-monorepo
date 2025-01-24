package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserOperation struct {
	Sender               string `json:"sender"`
	Nonce                string `json:"nonce"`
	InitCode             string `json:"initCode"`
	CallData             string `json:"callData"`
	CallGasLimit         string `json:"callGasLimit"`
	VerificationGasLimit string `json:"verificationGasLimit"`
	PreVerificationGas   string `json:"preVerificationGas"`
	MaxFeePerGas         string `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas"`
	PaymasterAndData     string `json:"paymasterAndData"`
	Signature            string `json:"signature"`
}

type PaymentTypeParams struct {
	SubscriptionId string `json:"subscriptionId"`
	PaylinkUrl     string `json:"paylinkUrl"`
}

type RawOperation struct {
	UserOpHash        string `json:"userOpHash"`
	UserOp            string `json:"userOp"`
	EntryPoint        string `json:"entryPoint"`
	ExecutionTime     int64  `json:"executionTime"`
	PaymentType       string `json:"paymentType"`
	PaymentTypeParams string `json:"paymentTypeParams"`
	Status            string `json:"status"`
	TransactionHash   string `json:"transactionHash"`
}

type Operation struct {
	UserOpHash        string            `json:"userOpHash"`
	UserOp            UserOperation     `json:"userOp"`
	EntryPoint        string            `json:"entryPoint"`
	ExecutionTime     int64             `json:"executionTime"`
	PaymentType       string            `json:"paymentType"`
	PaymentTypeParams PaymentTypeParams `json:"paymentTypeParams"`
	Status            string            `json:"status"`
	TransactionHash   string            `json:"transactionHash"`
}

type OperationsResponse struct {
	Data []Operation `json:"data"`
}

type OperationsRequest struct {
	SwAddress      string `json:"swaddress" binding:"required"`
	SubscriptionId string `json:"subscription_id" binding:"required"`
	Status         string `json:"status" binding:"required"`
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

// GetOperations godoc
// @Summary      List operations
// @Description  Retrieves all operations for authenticated user
// @Tags         operations
// @Accept       json
// @Produce      json
// @Security     Bearer
// @Param        page     query  int     false  "Page number"         default(1)
// @Param        limit    query  int     false  "Items per page"      default(10)
// @Success      200  {object}  OperationsResponse
// @Failure      401  {object}  ErrorResponse
// @Router       /operations [get]
func GetOperations(c *gin.Context) {
	apiKey := c.GetHeader("x-api-key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key is required"})
		return
	}

	swAddress := c.Query("swaddress")
	if swAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Smart Wallet Address is required"})
		return
	}

	subId := c.Query("subscriptionId")
	if subId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subscription ID is required"})
		return
	}

	status := c.Query("status")
	if status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status is required"})
		return
	}

	client := &http.Client{}
	request, err := http.NewRequest("GET", "https://api.billing.acta.link/api/operations", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	q := request.URL.Query()
	q.Add("swaddress", swAddress)
	q.Add("subscriptionId", subId)
	q.Add("status", status)
	request.URL.RawQuery = q.Encode()

	request.Header.Set("x-api-key", apiKey)

	resp, err := client.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch operations"})
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	var rawResponse struct {
		Data []RawOperation `json:"data"`
	}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	operations := OperationsResponse{
		Data: make([]Operation, 0, len(rawResponse.Data)),
	}

	for _, rawOp := range rawResponse.Data {
		op, err := transformOperation(rawOp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to transform operation"})
			return
		}
		operations.Data = append(operations.Data, op)
	}

	c.JSON(http.StatusOK, operations)
}
