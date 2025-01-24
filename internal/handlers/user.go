package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CheckUserAvailability godoc
// @Summary      Check username availability
// @Description  Verifies if a username is available for registration
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        address    query     string  true  "Address" example("0x1234567890abcdef")
// @Success      200  {object}  UserAvailabilityResponse
// @Failure      400  {object}  ErrorResponse      "Bad request"
// @Failure      401  {object}  ErrorResponse      "Unauthorized"
// @Failure      500  {object}  ErrorResponse      "Internal server error"
// @Router       /user [get]

func CheckUserAvailability(c *gin.Context) {
	apiKey := c.GetHeader("x-api-key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key is required"})
		return
	}

	address := c.Query("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address parameter is required"})
		return
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.billing.acta.link/api/ct/isuseravailable", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	q := req.URL.Query()
	q.Add("address", address)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("x-api-key", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user availability"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": "Failed to check user availability from upstream"})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	var getUserResp GetUserResponse
	if err := json.Unmarshal(body, &getUserResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	message := getUserResp.Message
	exists := message == UserExists
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
// @Failure      401  {object}  ErrorResponse      "Unauthorized"
// @Failure      500  {object}  ErrorResponse      "Internal server error"
// @Router       /user/register [post]

func RegisterUser(c *gin.Context) {
	apiKey := c.GetHeader("x-api-key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key is required"})
		return
	}

	var req UserLoginRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	client := &http.Client{}
	request, err := http.NewRequest("POST", "https://api.billing.acta.link/api/ct/register", bytes.NewBuffer(jsonBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	request.Header.Set("x-api-key", apiKey)
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, ErrorResponse{
			Error: string(body),
		})
		return
	}

	var registerUserResponse RegisterUserResponse
	if err := json.Unmarshal(body, &registerUserResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, registerUserResponse)
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
// @Failure      401  {object}  ErrorResponse      "Unauthorized"
// @Failure      500  {object}  ErrorResponse      "Internal server error"
// @Router       /user/login [post]

func LoginUser(c *gin.Context) {
	apiKey := c.GetHeader("x-api-key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API key is required"})
		return
	}

	var req UserLoginRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	client := &http.Client{}
	request, err := http.NewRequest("POST", "https://api.billing.acta.link/api/ct/register", bytes.NewBuffer(jsonBody))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	request.Header.Set("x-api-key", apiKey)
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.JSON(resp.StatusCode, gin.H{"error": string(body)})
		return
	}

	var loginUserResponse LoginUserResponse
	if err := json.Unmarshal(body, &loginUserResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse response"})
		return
	}

	c.JSON(http.StatusOK, loginUserResponse)
}
