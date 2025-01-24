package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

func CreateSubscription(c *gin.Context) {
	apiKey := c.GetHeader("x-api-key")
	if apiKey == "" {
		c.Status(http.StatusUnauthorized)
		return
	}

	var req SubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	client := &http.Client{}
	request, err := http.NewRequest("POST", "https://api.billing.acta.link/api/newsubscription", bytes.NewBuffer(jsonBody))
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	request.Header.Set("x-api-key", apiKey)
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
}

func GetAllSubscriptions(c *gin.Context) {
	apiKey := c.GetHeader("x-api-key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key is required"})
		return
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.billing.acta.link/api/subscription", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	req.Header.Set("x-api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subscriptions"})
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	var subscriptions GetSubscriptionsResponse
	if err := json.Unmarshal(body, &subscriptions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, subscriptions)
}

func GetSubscribers(c *gin.Context) {
	apiKey := c.GetHeader("x-api-key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key is required"})
		return
	}

	subId := c.Query("subscriptionId")
	if subId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Subscription ID is required"})
		return
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.billing.acta.link/api/ct/subscribers", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	q := req.URL.Query()
	q.Add("subscriptionId", subId)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("x-api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subscribers"})
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	var subscribers GetSubscribersResponse
	if err := json.Unmarshal(body, &subscribers); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, subscribers)
}

func DeleteSubscription(c *gin.Context) {
	apiKey := c.GetHeader("x-api-key")
	if apiKey == "" {
		c.Status(http.StatusUnauthorized)
		return
	}

	var req DeleteSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	client := &http.Client{}
	request, err := http.NewRequest("POST", "https://api.billing.acta.link/api/ct/subscription/delete", bytes.NewBuffer(jsonBody))
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	request.Header.Set("x-api-key", apiKey)
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)
}
