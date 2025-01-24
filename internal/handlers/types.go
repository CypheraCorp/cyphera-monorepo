package handlers

// Consts
const (
	UserExists = "exists"
)

// Request Types
type UserLoginRegisterRequest struct {
	Address   string `json:"address"`
	Message   string `json:"message"`
	Signature string `json:"signature"`
	Nonce     string `json:"nonce"`
}

type SubscriptionRequest struct {
	Title     string     `json:"title"`
	Tokens    []string   `json:"tokens"`
	Plans     []Plan     `json:"plans"`
	Receivers []Receiver `json:"receivers"`
	Linktree  *string    `json:"linktree,omitempty"`
	TrialDays *int       `json:"trialDays,omitempty"`
}

type DeleteSubscriptionRequest struct {
	SubscriptionId string `json:"subscriptionId" binding:"required"`
}

// Response Types

type GetUserResponse struct {
	Message string `json:"message"`
}

type GetNonceResponse struct {
	Nonce string `json:"nonce"`
}

type GetSubscriptionsResponse struct {
	Data []Subscription `json:"data"`
}

type CreateSubscriptionResponse struct {
	Message string `json:"message"`
}

type UserAvailabilityResponse struct {
	Exists bool `json:"exists"`
}

type DeleteSubscriptionResponse struct {
	Message string `json:"message"`
}

type RegisterUserResponse struct {
	Message string `json:"message"`
}

type LoginUserResponse struct {
	Message string `json:"message"`
}

type GetSubscribersResponse struct {
	Data SubscribersData `json:"data"`
}

type GetTokensResponse struct {
	Data []Token `json:"data"`
}

type GetNetworksResponse struct {
	Data []Network `json:"data"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request parameters"`
}

// Model Types
type Subscription struct {
	Id          string      `json:"id"`
	Title       string      `json:"title"`
	Status      string      `json:"status"`
	Linktree    string      `json:"linktree"`
	TrialDays   int         `json:"trialDays"`
	CreatedAt   string      `json:"createdAt"`
	UserId      string      `json:"userId"`
	PaymentLink PaymentLink `json:"paymentlink"`
	Plans       []Plan      `json:"plans"`
	Receivers   []Receiver  `json:"receivers"`
	Tokens      []Token     `json:"tokens"`
}

type SubscribersData struct {
	UserId         string       `json:"userid"`
	SubscriptionId string       `json:"subscriptionId"`
	Title          string       `json:"title"`
	Subscribers    []Subscriber `json:"subscribers"`
}

type Subscriber struct {
	EoaAddress     string `json:"eoaaddress"`
	SwAddress      string `json:"swaddress"`
	PlanId         string `json:"planId"`
	Status         string `json:"status"`
	SubscribedAt   string `json:"subscribedAt"`
	SubscriptionId string `json:"subscriptionId"`
	Plan           Plan   `json:"plan"`
}

type Plan struct {
	Id             string  `json:"id,omitempty"`
	SubscriptionId string  `json:"subscriptionId,omitempty"`
	Name           string  `json:"name"`
	Frequency      string  `json:"frequency"`
	Volume         int     `json:"volume"`
	Price          float64 `json:"price"`
}

type Receiver struct {
	ReceiverId     string `json:"receiverId,omitempty"`
	Address        string `json:"address"`
	NetworkId      int    `json:"networkId"`
	SubscriptionId string `json:"subscriptionId,omitempty"`
}

type PaymentLink struct {
	Id             string `json:"id"`
	Title          string `json:"title"`
	SubscriptionId string `json:"subscriptionId"`
	CreatedAt      string `json:"createdAt"`
	ValidTill      string `json:"validTill"`
}

type Token struct {
	Id          int    `json:"id"`
	Address     string `json:"address"`
	Symbol      string `json:"symbol"`
	Name        string `json:"name"`
	Decimals    int    `json:"decimals"`
	ChainId     int    `json:"chainId"`
	LogoURI     string `json:"logoURI"`
	CoingeckoId string `json:"coingeckoId"`
}

type Network struct {
	ChainId  int    `json:"chain_id"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Type     string `json:"type"`
}
