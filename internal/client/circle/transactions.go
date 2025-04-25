package circle

import (
	"context"
	httpClient "cyphera-api/internal/client/http"
	"fmt"
	"time"
)

// TransferChallengeRequest represents the request to create a transfer transaction challenge
type TransferChallengeRequest struct {
	IdempotencyKey     string   `json:"idempotencyKey"`
	WalletID           string   `json:"walletId,omitempty"`
	SourceAddress      string   `json:"sourceAddress,omitempty"`
	Blockchain         string   `json:"blockchain,omitempty"`
	DestinationAddress string   `json:"destinationAddress"`
	Amounts            []string `json:"amounts,omitempty"`
	TokenID            string   `json:"tokenId,omitempty"`
	TokenAddress       string   `json:"tokenAddress,omitempty"`
	FeeLevel           string   `json:"feeLevel,omitempty"` // LOW, MEDIUM, or HIGH
	GasLimit           string   `json:"gasLimit,omitempty"`
	GasPrice           string   `json:"gasPrice,omitempty"`
	MaxFee             string   `json:"maxFee,omitempty"`
	PriorityFee        string   `json:"priorityFee,omitempty"`
	NftTokenIds        []string `json:"nftTokenIds,omitempty"`
	RefID              string   `json:"refId,omitempty"`
}

// TransferChallengeResponse represents the response from creating a transfer transaction challenge
type TransferChallengeResponse struct {
	Data struct {
		ChallengeID string `json:"challengeId"`
	} `json:"data"`
}

// CreateTransferChallenge generates a challenge for initiating an on-chain digital asset transfer
// from a specified user-controlled wallet.
//
// The function accepts parameters for specifying the source wallet (via walletId or sourceAddress+blockchain),
// the destination address, and optional parameters for token information, fee settings, and additional metadata.
//
// For ERC721 token transfers, the amounts field should be ["1"] (array with "1" as the only element).
func (c *CircleClient) CreateTransferChallenge(ctx context.Context, request TransferChallengeRequest, userToken string) (*TransferChallengeResponse, error) {
	// Basic validation
	if request.IdempotencyKey == "" {
		return nil, fmt.Errorf("idempotencyKey is required")
	}

	if request.DestinationAddress == "" {
		return nil, fmt.Errorf("destinationAddress is required")
	}

	// Validate source identification parameters
	if request.WalletID == "" && (request.SourceAddress == "" || request.Blockchain == "") {
		return nil, fmt.Errorf("either walletId or both sourceAddress and blockchain must be provided")
	}

	// Validate token identification (if applicable)
	if request.TokenID != "" && request.TokenAddress != "" {
		return nil, fmt.Errorf("tokenId and tokenAddress are mutually exclusive")
	}

	// Validate fee parameters - these have complex rules about mutual exclusivity
	// but we'll just pass them along and let the API handle the validation

	// Make the API request
	resp, err := c.httpClient.Post(
		ctx,
		"user/transactions/transfer",
		request,
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer challenge: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	var response TransferChallengeResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process transfer challenge response: %w", err)
	}

	return &response, nil
}

// EstimatedFee represents the fee estimation for a transaction
type EstimatedFee struct {
	GasLimit    string `json:"gasLimit"`
	GasPrice    string `json:"gasPrice"`
	MaxFee      string `json:"maxFee"`
	PriorityFee string `json:"priorityFee"`
	BaseFee     string `json:"baseFee"`
	NetworkFee  string `json:"networkFee"`
}

// TransactionScreeningReason represents a reason for transaction screening
type TransactionScreeningReason struct {
	Source         string   `json:"source"`
	SourceValue    string   `json:"sourceValue"`
	RiskScore      string   `json:"riskScore"`
	RiskCategories []string `json:"riskCategories"`
	Type           string   `json:"type"`
}

// TransactionScreeningEvaluation represents the screening evaluation for a transaction
type TransactionScreeningEvaluation struct {
	RuleName      string                       `json:"ruleName"`
	Actions       []string                     `json:"actions"`
	ScreeningDate time.Time                    `json:"screeningDate"`
	Reasons       []TransactionScreeningReason `json:"reasons"`
}

// Transaction represents a Circle transaction
type Transaction struct {
	ID                             string                          `json:"id"`
	AbiFunctionSignature           string                          `json:"abiFunctionSignature"`
	AbiParameters                  []string                        `json:"abiParameters"`
	Amounts                        []string                        `json:"amounts"`
	AmountInUSD                    string                          `json:"amountInUSD"`
	BlockHash                      string                          `json:"blockHash"`
	BlockHeight                    int                             `json:"blockHeight"`
	Blockchain                     string                          `json:"blockchain"`
	ContractAddress                string                          `json:"contractAddress"`
	CreateDate                     time.Time                       `json:"createDate"`
	CustodyType                    string                          `json:"custodyType"`
	DestinationAddress             string                          `json:"destinationAddress"`
	ErrorReason                    string                          `json:"errorReason"`
	ErrorDetails                   string                          `json:"errorDetails"`
	EstimatedFee                   EstimatedFee                    `json:"estimatedFee"`
	FeeLevel                       string                          `json:"feeLevel"`
	FirstConfirmDate               string                          `json:"firstConfirmDate"`
	NetworkFee                     string                          `json:"networkFee"`
	NetworkFeeInUSD                string                          `json:"networkFeeInUSD"`
	Nfts                           []string                        `json:"nfts"`
	Operation                      string                          `json:"operation"`
	RefID                          string                          `json:"refId"`
	SourceAddress                  string                          `json:"sourceAddress"`
	State                          string                          `json:"state"`
	TokenID                        string                          `json:"tokenId"`
	TransactionType                string                          `json:"transactionType"`
	TxHash                         string                          `json:"txHash"`
	UpdateDate                     time.Time                       `json:"updateDate"`
	UserID                         string                          `json:"userId"`
	WalletID                       string                          `json:"walletId"`
	TransactionScreeningEvaluation *TransactionScreeningEvaluation `json:"transactionScreeningEvaluation"`
}

// TransactionListResponse represents the response from listing transactions
type TransactionListResponse struct {
	Data struct {
		Transactions []Transaction `json:"transactions"`
	} `json:"data"`
}

// ListTransactionsParams represents query parameters for listing transactions
type ListTransactionsParams struct {
	Blockchain         *string    `json:"blockchain,omitempty"`
	DestinationAddress *string    `json:"destinationAddress,omitempty"`
	IncludeAll         *bool      `json:"includeAll,omitempty"`
	Operation          *string    `json:"operation,omitempty"`
	State              *string    `json:"state,omitempty"`
	TxHash             *string    `json:"txHash,omitempty"`
	TxType             *string    `json:"txType,omitempty"`
	UserID             *string    `json:"userId,omitempty"`
	WalletIDs          *string    `json:"walletIds,omitempty"`
	From               *time.Time `json:"from,omitempty"`
	To                 *time.Time `json:"to,omitempty"`
	PageBefore         *string    `json:"pageBefore,omitempty"`
	PageAfter          *string    `json:"pageAfter,omitempty"`
	PageSize           *int       `json:"pageSize,omitempty"`
}

// ListTransactions retrieves a list of transactions that match the specified parameters
//
// This method supports querying transactions by various filters such as blockchain, address,
// transaction state, and more. It also supports pagination.
func (c *CircleClient) ListTransactions(ctx context.Context, userToken string, params *ListTransactionsParams) (*TransactionListResponse, error) {
	// Build the request with options
	options := []httpClient.RequestOption{
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	}

	// Add optional query parameters if provided
	if params != nil {
		if params.Blockchain != nil && *params.Blockchain != "" {
			options = append(options, httpClient.WithQueryParam("blockchain", *params.Blockchain))
		}
		if params.DestinationAddress != nil && *params.DestinationAddress != "" {
			options = append(options, httpClient.WithQueryParam("destinationAddress", *params.DestinationAddress))
		}
		if params.IncludeAll != nil && *params.IncludeAll {
			options = append(options, httpClient.WithQueryParam("includeAll", "true"))
		}
		if params.Operation != nil && *params.Operation != "" {
			options = append(options, httpClient.WithQueryParam("operation", *params.Operation))
		}
		if params.State != nil && *params.State != "" {
			options = append(options, httpClient.WithQueryParam("state", *params.State))
		}
		if params.TxHash != nil && *params.TxHash != "" {
			options = append(options, httpClient.WithQueryParam("txHash", *params.TxHash))
		}
		if params.TxType != nil && *params.TxType != "" {
			options = append(options, httpClient.WithQueryParam("txType", *params.TxType))
		}
		if params.UserID != nil && *params.UserID != "" {
			options = append(options, httpClient.WithQueryParam("userId", *params.UserID))
		}
		if params.WalletIDs != nil && *params.WalletIDs != "" {
			options = append(options, httpClient.WithQueryParam("walletIds", *params.WalletIDs))
		}
		if params.From != nil {
			options = append(options, httpClient.WithQueryParam("from", params.From.Format(time.RFC3339)))
		}
		if params.To != nil {
			options = append(options, httpClient.WithQueryParam("to", params.To.Format(time.RFC3339)))
		}
		if params.PageSize != nil {
			options = append(options, httpClient.WithQueryParam("pageSize", fmt.Sprintf("%d", *params.PageSize)))
		}
		if params.PageBefore != nil && *params.PageBefore != "" {
			options = append(options, httpClient.WithQueryParam("pageBefore", *params.PageBefore))
		}
		if params.PageAfter != nil && *params.PageAfter != "" {
			options = append(options, httpClient.WithQueryParam("pageAfter", *params.PageAfter))
		}
	}

	// Make the API request
	resp, err := c.httpClient.Get(
		ctx,
		"transactions",
		options...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	var response TransactionListResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process transaction list response: %w", err)
	}

	return &response, nil
}

// TransactionResponse represents the response from getting a single transaction
type TransactionResponse struct {
	Data struct {
		Transaction Transaction `json:"transaction"`
	} `json:"data"`
}

// GetTransaction retrieves details about a specific transaction by its ID
//
// This function fetches comprehensive information about a transaction, including
// status, amounts, blockchain details, fee information, and compliance screening data.
func (c *CircleClient) GetTransaction(ctx context.Context, transactionID string, userToken string) (*TransactionResponse, error) {
	if transactionID == "" {
		return nil, fmt.Errorf("transaction ID is required")
	}

	// Make the API request
	resp, err := c.httpClient.Get(
		ctx,
		fmt.Sprintf("transactions/%s", transactionID),
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	var response TransactionResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process transaction response: %w", err)
	}

	return &response, nil
}

// FeeEstimate represents the fee estimation for a specific fee level
type FeeEstimate struct {
	GasLimit    string `json:"gasLimit"`
	GasPrice    string `json:"gasPrice"`
	MaxFee      string `json:"maxFee"`
	PriorityFee string `json:"priorityFee"`
	BaseFee     string `json:"baseFee"`
	NetworkFee  string `json:"networkFee"`
}

// EstimateTransferFeeRequest represents the request to estimate transfer transaction fees
type EstimateTransferFeeRequest struct {
	DestinationAddress string   `json:"destinationAddress"`
	Amounts            []string `json:"amounts"`
	WalletID           string   `json:"walletId,omitempty"`
	SourceAddress      string   `json:"sourceAddress,omitempty"`
	Blockchain         string   `json:"blockchain,omitempty"`
	TokenID            string   `json:"tokenId,omitempty"`
	TokenAddress       string   `json:"tokenAddress,omitempty"`
	NftTokenIds        []string `json:"nftTokenIds,omitempty"`
}

// EstimateTransferFeeResponse represents the response from estimating transfer fees
type EstimateTransferFeeResponse struct {
	Data struct {
		High                 FeeEstimate `json:"high"`
		Medium               FeeEstimate `json:"medium"`
		Low                  FeeEstimate `json:"low"`
		CallGasLimit         string      `json:"callGasLimit"`
		VerificationGasLimit string      `json:"verificationGasLimit"`
		PreVerificationGas   string      `json:"preVerificationGas"`
	} `json:"data"`
}

// EstimateTransferFee estimates gas fees for a transfer transaction based on input parameters
//
// This function allows you to get fee estimates at different fee levels (low, medium, high)
// before creating an actual transfer transaction. This helps in determining the appropriate
// fee settings for subsequent transfers.
func (c *CircleClient) EstimateTransferFee(ctx context.Context, request EstimateTransferFeeRequest, userToken string) (*EstimateTransferFeeResponse, error) {
	// Basic validation
	if request.DestinationAddress == "" {
		return nil, fmt.Errorf("destinationAddress is required")
	}

	if len(request.Amounts) == 0 {
		return nil, fmt.Errorf("at least one amount is required")
	}

	// Validate source identification parameters (similar to CreateTransferChallenge)
	if request.WalletID == "" && (request.SourceAddress == "" || request.Blockchain == "") {
		return nil, fmt.Errorf("either walletId or both sourceAddress and blockchain must be provided")
	}

	// Validate token identification (if applicable)
	if request.TokenID != "" && request.TokenAddress != "" {
		return nil, fmt.Errorf("tokenId and tokenAddress are mutually exclusive")
	}

	// Make the API request
	resp, err := c.httpClient.Post(
		ctx,
		"transactions/transfer/estimateFee",
		request,
		httpClient.WithBearerToken(c.apiKey),
		httpClient.WithHeader("X-User-Token", userToken),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate transfer fee: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	var response EstimateTransferFeeResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process fee estimation response: %w", err)
	}

	return &response, nil
}

// ValidateAddressRequest represents the request to validate a blockchain address
type ValidateAddressRequest struct {
	Blockchain string `json:"blockchain"`
	Address    string `json:"address"`
}

// ValidateAddressResponse represents the response from validating an address
type ValidateAddressResponse struct {
	Data struct {
		IsValid bool `json:"isValid"`
	} `json:"data"`
}

// ValidateAddress confirms that a specified address is valid for a given token on a certain blockchain
//
// This function helps verify if an address is properly formatted and valid for use on the specified
// blockchain before attempting to send a transaction to it.
func (c *CircleClient) ValidateAddress(ctx context.Context, request ValidateAddressRequest) (*ValidateAddressResponse, error) {
	// Basic validation
	if request.Address == "" {
		return nil, fmt.Errorf("address is required")
	}

	if request.Blockchain == "" {
		return nil, fmt.Errorf("blockchain is required")
	}

	// Validate the blockchain is supported
	if err := ValidateBlockchains([]string{request.Blockchain}); err != nil {
		return nil, err
	}

	// Make the API request
	resp, err := c.httpClient.Post(
		ctx,
		"transactions/validateAddress",
		request,
		httpClient.WithBearerToken(c.apiKey),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to validate address: %w", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	var response ValidateAddressResponse
	if err := c.httpClient.ProcessJSONResponse(resp, &response); err != nil {
		return nil, fmt.Errorf("failed to process address validation response: %w", err)
	}

	return &response, nil
}
