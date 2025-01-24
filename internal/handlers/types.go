package handlers

type GetUserResponse struct {
	Message string `json:"message"`
}
type UserLoginRegisterRequest struct {
	Address   string `json:"address"`
	Message   string `json:"message"`
	Signature string `json:"signature"`
	Nonce     string `json:"nonce"`
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

type SubscriptionRequest struct {
	Title     string     `json:"title"`
	Tokens    []string   `json:"tokens"`
	Plans     []Plan     `json:"plans"`
	Receivers []Receiver `json:"receivers"`
	Linktree  *string    `json:"linktree,omitempty"`
	TrialDays *int       `json:"trialDays,omitempty"`
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

type GetSubscriptionsResponse struct {
	Data []Subscription `json:"data"`
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

type SubscribersData struct {
	UserId         string       `json:"userid"`
	SubscriptionId string       `json:"subscriptionId"`
	Title          string       `json:"title"`
	Subscribers    []Subscriber `json:"subscribers"`
}

type GetSubscribersResponse struct {
	Data SubscribersData `json:"data"`
}

type DeleteSubscriptionRequest struct {
	SubscriptionId string `json:"subscriptionId" binding:"required"`
}

type GetTokensResponse struct {
	Data []Token `json:"data"`
}

type GetNetworksResponse struct {
	Data []Network `json:"data"`
}

type Network struct {
	ChainId  int    `json:"chain_id"`
	Name     string `json:"name"`
	Currency string `json:"currency"`
	Type     string `json:"type"`
}
