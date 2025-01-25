package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func handleStatusCode(statusCode *int, defaultCode int) int {
	if statusCode == nil {
		return defaultCode
	}
	return *statusCode
}

// GetNonce godoc
// @Summary      Get authentication nonce
// @Description  Retrieves a nonce for wallet-based authentication
// @Tags         authentication
// @Accept       json
// @Produce      json
// @Success      200  {object}  GetNonceResponse   "Returns nonce"
// @Failure      400  {object}  ErrorResponse      "Bad request"
// @Failure      403  {object}  ErrorResponse      "Forbidden"
// @Failure      500  {object}  ErrorResponse      "Internal server error"
// @Router       /nonce [get]
func (h *HandlerClient) GetNonce(c *gin.Context) {
	nonceResp, statusCode, err := h.actalink.GetNonce()
	if err != nil {
		c.JSON(handleStatusCode(statusCode, http.StatusInternalServerError), gin.H{"error": fmt.Sprintf("Failed to get nonce: %v", err)})
		return
	}

	c.JSON(http.StatusOK, nonceResp)
}

// CheckUserAvailability godoc
// @Summary      Check username availability
// @Description  Verifies if a username is available for registration
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        address    query     string  true  "Address" example("0x1234567890abcdef")
// @Success      200  {object}  UserAvailabilityResponse
// @Failure      400  {object}  ErrorResponse      "Bad request"
// @Failure      403  {object}  ErrorResponse      "Forbidden"
// @Failure      500  {object}  ErrorResponse      "Internal server error"
// @Router       /users [get]
func (h *HandlerClient) CheckUserAvailability(c *gin.Context) {
	address := c.Query("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address parameter is required"})
		return
	}

	availResp, statusCode, err := h.actalink.CheckUserAvailability(address)
	if err != nil {
		c.JSON(handleStatusCode(statusCode, http.StatusInternalServerError), gin.H{"error": fmt.Sprintf("Failed to check user availability: %v", err)})
		return
	}

	exists := availResp.Message == UserExists
	c.JSON(http.StatusOK, UserAvailabilityResponse{
		Exists: exists,
	})
}

// RegisterUser godoc
// @Summary      Register new user
// @Description  Creates a new user account with wallet authentication
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request  body      UserLoginRegisterRequest  true  "User registration payload"
// @Success      200  {object}  RegisterUserResponse
// @Failure      400  {object}  ErrorResponse      "Bad request"
// @Failure      403  {object}  ErrorResponse      "Forbidden"
// @Failure      500  {object}  ErrorResponse      "Internal server error"
// @Router       /user/register [post]
func (h *HandlerClient) RegisterUser(c *gin.Context) {
	var req UserLoginRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request body: %v", err)})
		return
	}

	registerResp, statusCode, err := h.actalink.RegisterUser(req)
	if err != nil {
		c.JSON(handleStatusCode(statusCode, http.StatusInternalServerError), gin.H{"error": fmt.Sprintf("Failed to register user: %v", err)})
		return
	}

	c.JSON(http.StatusOK, registerResp)
}

// LoginUser godoc
// @Summary      Authenticate user
// @Description  Authenticates user using wallet signature and returns JWT token
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request  body      UserLoginRegisterRequest  true  "User login payload"
// @Success      200  {object}  LoginUserResponse
// @Failure      400  {object}  ErrorResponse      "Bad request"
// @Failure      403  {object}  ErrorResponse      "Forbidden"
// @Failure      500  {object}  ErrorResponse      "Internal server error"
// @Router       /user/login [post]
func (h *HandlerClient) LoginUser(c *gin.Context) {
	var req UserLoginRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request body: %v", err)})
		return
	}

	loginResp, statusCode, err := h.actalink.LoginUser(req)
	if err != nil {
		c.JSON(handleStatusCode(statusCode, http.StatusInternalServerError), gin.H{"error": fmt.Sprintf("Failed to login user: %v", err)})
		return
	}

	c.JSON(http.StatusOK, loginResp)
}

// GetNetworks godoc
// @Summary      List networks
// @Description  Retrieves all supported blockchain networks
// @Tags         networks
// @Accept       json
// @Produce      json
// @Success      200  {object}  GetNetworksResponse
// @Failure      403  {object}  ErrorResponse      "Forbidden"
// @Failure      500  {object}  ErrorResponse      "Internal server error"
// @Router       /networks [get]
func (h *HandlerClient) GetNetworks(c *gin.Context) {
	networks, statusCode, err := h.actalink.GetNetworks()
	if err != nil {
		c.JSON(handleStatusCode(statusCode, http.StatusInternalServerError), gin.H{"error": fmt.Sprintf("Failed to fetch networks: %v", err)})
		return
	}

	c.JSON(http.StatusOK, networks)
}

// GetOperations godoc
// @Summary      List operations
// @Description  Retrieves all operations for authenticated user
// @Accept       json
// @Produce      json
// @Param        swaddress  query     string  true  "Smart Wallet Address"  example("0x1234567890abcdef")
// @Param        subscriptionId  query     string  true  "Subscription ID"  example("1234567890")
// @Param        status  query     string  true  "Status"
// @Success      200  {object}  OperationsResponse
// @Failure      400  {object}  ErrorResponse      "Bad request"
// @Failure      403  {object}  ErrorResponse      "Forbidden"
// @Failure      500  {object}  ErrorResponse      "Internal server error"
// @Router       /operations [get]
func (h *HandlerClient) GetOperations(c *gin.Context) {
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

	operations, statusCode, err := h.actalink.GetOperations(swAddress, subId, status)
	if err != nil {
		c.JSON(handleStatusCode(statusCode, http.StatusInternalServerError), gin.H{"error": fmt.Sprintf("Failed to fetch operations: %v", err)})
		return
	}

	c.JSON(http.StatusOK, operations)
}

// GetAllSubscriptions godoc
// @Summary      List all subscriptions
// @Description  Retrieves all available subscription plans
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Success      200  {object}  GetSubscriptionsResponse
// @Failure      400  {object}  ErrorResponse      "Bad request"
// @Failure      403  {object}  ErrorResponse      "Forbidden"
// @Failure      500  {object}  ErrorResponse      "Internal server error"
// @Router       /subscriptions [get]
func (h *HandlerClient) GetAllSubscriptions(c *gin.Context) {
	subscriptions, statusCode, err := h.actalink.GetAllSubscriptions()
	if err != nil {
		c.JSON(handleStatusCode(statusCode, http.StatusInternalServerError), gin.H{"error": fmt.Sprintf("Failed to fetch subscriptions: %v", err)})
		return
	}

	c.JSON(http.StatusOK, subscriptions)
}

// CreateSubscription godoc
// @Summary      Create a new subscription
// @Description  Creates a new subscription plan
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        subscription  body  SubscriptionRequest  true  "Subscription details"
// @Success      200  {object}  CreateSubscriptionResponse
// @Failure      400  {object}  ErrorResponse      "Bad request"
// @Failure      403  {object}  ErrorResponse      "Forbidden"
// @Failure      500  {object}  ErrorResponse      "Internal server error"
// @Router       /subscriptions [post]
func (h *HandlerClient) CreateSubscription(c *gin.Context) {
	var req SubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request body: %v", err)})
		return
	}

	resp, statusCode, err := h.actalink.CreateSubscription(req)
	if err != nil {
		c.JSON(handleStatusCode(statusCode, http.StatusInternalServerError), gin.H{"error": fmt.Sprintf("Failed to create subscription: %v", err)})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteSubscription godoc
// @Summary      Delete a subscription
// @Description  Deletes a subscription plan
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        subscription  body  DeleteSubscriptionRequest  true  "Subscription details"
// @Success      200
// @Failure      400  {object}  ErrorResponse      "Bad request"
// @Failure      403  {object}  ErrorResponse      "Forbidden"
// @Failure      500  {object}  ErrorResponse      "Internal server error"
// @Router       /subscriptions [delete]
func (h *HandlerClient) DeleteSubscription(c *gin.Context) {
	var req DeleteSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request body: %v", err)})
		return
	}

	resp, statusCode, err := h.actalink.DeleteSubscription(req)
	if err != nil {
		c.JSON(handleStatusCode(statusCode, http.StatusInternalServerError), gin.H{"error": fmt.Sprintf("Failed to delete subscription: %v", err)})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetSubscribers godoc
// @Summary      List subscribers
// @Description  Retrieves all subscribers for authenticated user
// @Tags         subscribers
// @Accept       json
// @Produce      json
// @Param        subscriptionId  query     string  true  "Subscription ID"  example("1234567890")
// @Success      200  {object}  GetSubscribersResponse
// @Failure      400  {object}  ErrorResponse      "Bad request"
// @Failure      403  {object}  ErrorResponse      "Forbidden"
// @Failure      500  {object}  ErrorResponse      "Internal server error"
// @Router       /subscribers [get]
func (h *HandlerClient) GetSubscribers(c *gin.Context) {
	subId := c.Query("subscriptionId")
	if subId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subscription ID is required"})
		return
	}

	subscribers, statusCode, err := h.actalink.GetSubscribers(subId)
	if err != nil {
		c.JSON(handleStatusCode(statusCode, http.StatusInternalServerError), gin.H{"error": fmt.Sprintf("Failed to fetch subscribers: %v", err)})
		return
	}

	c.JSON(http.StatusOK, subscribers)
}

// GetTokens godoc
// @Summary      List tokens
// @Description  Retrieves all available tokens
// @Tags         tokens
// @Accept       json
// @Produce      json
// @Success      200  {object}  GetTokensResponse
// @Failure      400  {object}  ErrorResponse      "Bad request"
// @Failure      403  {object}  ErrorResponse      "Forbidden"
// @Failure      500  {object}  ErrorResponse      "Internal server error"
// @Router       /tokens [get]
func (h *HandlerClient) GetTokens(c *gin.Context) {
	tokens, statusCode, err := h.actalink.GetTokens()
	if err != nil {
		c.JSON(handleStatusCode(statusCode, http.StatusInternalServerError), gin.H{"error": fmt.Sprintf("Failed to fetch tokens: %v", err)})
		return
	}

	c.JSON(http.StatusOK, tokens)
}
