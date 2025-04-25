package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cyphera-api/internal/logger"

	"github.com/cenkalti/backoff/v4"
	"go.uber.org/zap"
)

// RequestOption represents a function that can modify an HTTP request
type RequestOption func(*http.Request)

// ClientOption represents a function that can modify the HTTP client
type ClientOption func(*HTTPClient)

// ResponseProcessor processes HTTP responses
type ResponseProcessor func(*http.Response) error

// Middleware represents a function that wraps an http.RoundTripper
type Middleware func(http.RoundTripper) http.RoundTripper

// HTTPError represents an error returned from an HTTP request
type HTTPError struct {
	StatusCode int
	Status     string
	URL        string
	Method     string
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%s %s failed with status %d %s: %s", e.Method, e.URL, e.StatusCode, e.Status, e.Body)
}

// HTTPClient is a robust HTTP client with advanced features
type HTTPClient struct {
	httpClient     *http.Client
	baseURL        string
	defaultHeaders map[string]string
	retryConfig    *RetryConfig
	middlewares    []Middleware
	metrics        MetricsCollector
}

// RetryConfig configures the retry behavior
type RetryConfig struct {
	MaxRetries           int
	InitialInterval      time.Duration
	MaxInterval          time.Duration
	Multiplier           float64
	MaxElapsedTime       time.Duration
	RetryableStatusCodes []int
}

// MetricsCollector defines an interface for collecting metrics
type MetricsCollector interface {
	RecordRequestDuration(method, path string, statusCode int, duration time.Duration)
	RecordRequestCount(method, path string, statusCode int)
	RecordRequestError(method, path string)
}

// DefaultRetryConfig provides sensible defaults for retries
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:           3,
		InitialInterval:      100 * time.Millisecond,
		MaxInterval:          10 * time.Second,
		Multiplier:           2.0,
		MaxElapsedTime:       30 * time.Second,
		RetryableStatusCodes: []int{408, 429, 500, 502, 503, 504},
	}
}

// NewHTTPClient creates a new HTTPClient with the given options
func NewHTTPClient(options ...ClientOption) *HTTPClient {
	client := &HTTPClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		defaultHeaders: map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
		},
		retryConfig: DefaultRetryConfig(),
		metrics:     &NoopMetricsCollector{},
	}

	// Apply all client options
	for _, option := range options {
		option(client)
	}

	// Apply middlewares to the transport
	if len(client.middlewares) > 0 {
		transport := client.httpClient.Transport
		if transport == nil {
			transport = http.DefaultTransport
		}

		// Apply middlewares in reverse order so the first one is outermost
		for i := len(client.middlewares) - 1; i >= 0; i-- {
			transport = client.middlewares[i](transport)
		}
		client.httpClient.Transport = transport
	}

	return client
}

// WithBaseURL sets the base URL for all requests
func WithBaseURL(baseURL string) ClientOption {
	return func(c *HTTPClient) {
		c.baseURL = baseURL
	}
}

// WithDefaultHeader adds a default header to all requests
func WithDefaultHeader(key, value string) ClientOption {
	return func(c *HTTPClient) {
		c.defaultHeaders[key] = value
	}
}

// WithTimeout sets the timeout for all requests
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *HTTPClient) {
		c.httpClient.Timeout = timeout
	}
}

// WithRetryConfig sets the retry configuration
func WithRetryConfig(config *RetryConfig) ClientOption {
	return func(c *HTTPClient) {
		c.retryConfig = config
	}
}

// WithMiddleware adds a middleware to the client
func WithMiddleware(middleware Middleware) ClientOption {
	return func(c *HTTPClient) {
		c.middlewares = append(c.middlewares, middleware)
	}
}

// WithMetricsCollector sets the metrics collector
func WithMetricsCollector(collector MetricsCollector) ClientOption {
	return func(c *HTTPClient) {
		c.metrics = collector
	}
}

// WithHeader adds a header to the request
func WithHeader(key, value string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

// WithQueryParam adds a query parameter to the request
func WithQueryParam(key, value string) RequestOption {
	return func(req *http.Request) {
		q := req.URL.Query()
		q.Add(key, value)
		req.URL.RawQuery = q.Encode()
	}
}

// WithBasicAuth adds basic authentication to the request
func WithBasicAuth(username, password string) RequestOption {
	return func(req *http.Request) {
		req.SetBasicAuth(username, password)
	}
}

// WithBearerToken adds bearer token authentication to the request
func WithBearerToken(token string) RequestOption {
	return func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

// Get performs an HTTP GET request
func (c *HTTPClient) Get(ctx context.Context, path string, options ...RequestOption) (*http.Response, error) {
	return c.DoRequest(ctx, http.MethodGet, path, nil, options...)
}

// Post performs an HTTP POST request with a JSON body
func (c *HTTPClient) Post(ctx context.Context, path string, body interface{}, options ...RequestOption) (*http.Response, error) {
	return c.DoRequest(ctx, http.MethodPost, path, body, options...)
}

// Put performs an HTTP PUT request with a JSON body
func (c *HTTPClient) Put(ctx context.Context, path string, body interface{}, options ...RequestOption) (*http.Response, error) {
	return c.DoRequest(ctx, http.MethodPut, path, body, options...)
}

// Patch performs an HTTP PATCH request with a JSON body
func (c *HTTPClient) Patch(ctx context.Context, path string, body interface{}, options ...RequestOption) (*http.Response, error) {
	return c.DoRequest(ctx, http.MethodPatch, path, body, options...)
}

// Delete performs an HTTP DELETE request
func (c *HTTPClient) Delete(ctx context.Context, path string, options ...RequestOption) (*http.Response, error) {
	return c.DoRequest(ctx, http.MethodDelete, path, nil, options...)
}

// DoRequest is the generic method that performs all HTTP requests
func (c *HTTPClient) DoRequest(ctx context.Context, method, path string, body interface{}, options ...RequestOption) (*http.Response, error) {
	start := time.Now()

	// Build the full URL by directly concatenating base URL and path
	fullURL := path
	if c.baseURL != "" {
		// Ensure the base URL does not have a trailing slash
		trimmedBaseURL := strings.TrimSuffix(c.baseURL, "/")
		// Ensure the path has a leading slash
		trimmedPath := path
		if !strings.HasPrefix(trimmedPath, "/") {
			trimmedPath = "/" + trimmedPath
		}
		fullURL = trimmedBaseURL + trimmedPath
	} else {
		// If no base URL, use the path directly (ensure it's a valid URL)
		// This branch might need more robust URL validation depending on usage
		_, err := url.ParseRequestURI(path)
		if err != nil {
			return nil, fmt.Errorf("invalid path used without base URL: %s, error: %w", path, err)
		}
	}

	// Prepare the request body
	var bodyReader io.Reader
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyJSON)
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add default headers
	for key, value := range c.defaultHeaders {
		req.Header.Set(key, value)
	}

	// Apply all request options
	for _, option := range options {
		option(req)
	}

	// Execute the request with retries if configured
	var resp *http.Response
	var requestErr error

	if c.retryConfig != nil && c.retryConfig.MaxRetries > 0 {
		operation := func() error {
			// nolint:bodyclose // Body is closed conditionally for retries or handled by caller/later checks
			resp, requestErr = c.httpClient.Do(req)

			// Check if we should retry based on the status code
			if requestErr == nil && resp != nil {
				for _, code := range c.retryConfig.RetryableStatusCodes {
					if resp.StatusCode == code {
						// Read and close the body to avoid connection leaks
						if resp.Body != nil {
							_, _ = io.Copy(io.Discard, resp.Body)
							_ = resp.Body.Close()
						}
						return fmt.Errorf("retryable status code: %d", resp.StatusCode)
					}
				}
			}

			return requestErr
		}

		expBackoff := backoff.NewExponentialBackOff()
		expBackoff.InitialInterval = c.retryConfig.InitialInterval
		expBackoff.MaxInterval = c.retryConfig.MaxInterval
		expBackoff.Multiplier = c.retryConfig.Multiplier
		expBackoff.MaxElapsedTime = c.retryConfig.MaxElapsedTime

		// Execute the retry operation
		requestErr = backoff.Retry(operation, backoff.WithMaxRetries(expBackoff, uint64(c.retryConfig.MaxRetries)))
	} else {
		resp, requestErr = c.httpClient.Do(req)
	}

	// Record metrics
	duration := time.Since(start)
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
	}
	c.metrics.RecordRequestDuration(method, path, statusCode, duration)
	c.metrics.RecordRequestCount(method, path, statusCode)

	// Handle request errors
	if requestErr != nil {
		c.metrics.RecordRequestError(method, path)
		logger.Error("HTTP request failed",
			zap.String("method", method),
			zap.String("url", fullURL),
			zap.Error(requestErr),
			zap.Duration("duration", duration))
		return nil, fmt.Errorf("http request failed: %w", requestErr)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		c.metrics.RecordRequestError(method, path)

		// Read the error body
		var bodyBytes []byte
		if resp.Body != nil {
			bodyBytes, _ = io.ReadAll(resp.Body)
			resp.Body.Close()
			// Recreate the body for further processing
			resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		httpErr := &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			URL:        fullURL,
			Method:     method,
			Body:       string(bodyBytes),
		}

		logger.Warn("HTTP error response",
			zap.String("method", method),
			zap.String("url", fullURL),
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(bodyBytes)),
			zap.Duration("duration", duration))

		return resp, httpErr
	}

	// Log successful requests
	logger.Info("HTTP request successful",
		zap.String("method", method),
		zap.String("url", fullURL),
		zap.Int("status", resp.StatusCode),
		zap.Duration("duration", duration))

	return resp, nil
}

// ProcessJSONResponse decodes a JSON response into the provided target
func (c *HTTPClient) ProcessJSONResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			URL:        resp.Request.URL.String(),
			Method:     resp.Request.Method,
			Body:       string(bodyBytes),
		}
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

// NoopMetricsCollector is a metrics collector that does nothing
type NoopMetricsCollector struct{}

func (n *NoopMetricsCollector) RecordRequestDuration(method, path string, statusCode int, duration time.Duration) {
}
func (n *NoopMetricsCollector) RecordRequestCount(method, path string, statusCode int) {}
func (n *NoopMetricsCollector) RecordRequestError(method, path string)                 {}

// LoggingMiddleware creates a middleware that logs requests and responses
func LoggingMiddleware() Middleware {
	return func(next http.RoundTripper) http.RoundTripper {
		return &loggingRoundTripper{next: next}
	}
}

type loggingRoundTripper struct {
	next http.RoundTripper
}

func (l *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	logger.Debug("HTTP request started",
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.Any("headers", req.Header))

	resp, err := l.next.RoundTrip(req)

	duration := time.Since(start)
	if err != nil {
		logger.Error("HTTP request failed",
			zap.String("method", req.Method),
			zap.String("url", req.URL.String()),
			zap.Error(err),
			zap.Duration("duration", duration))
		return resp, err
	}

	logger.Debug("HTTP response received",
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.Int("status", resp.StatusCode),
		zap.Duration("duration", duration))

	return resp, nil
}

func (c *HTTPClient) GetBaseURL() string {
	return c.baseURL
}

// Example usage:
// client := NewHTTPClient(
//     WithBaseURL("https://api.example.com"),
//     WithTimeout(10 * time.Second),
//     WithDefaultHeader("X-API-Key", "your-api-key"),
//     WithRetryConfig(DefaultRetryConfig()),
//     WithMiddleware(LoggingMiddleware()),
// )
//
// GET EXAMPLE
// resp, err := client.Get(ctx, "/users", WithQueryParam("page", "1"))
// if err != nil {
//     // Handle error
// }
//
// var users []User
// if err := client.ProcessJSONResponse(resp, &users); err != nil {
//     // Handle error
// }
//
// POST EXAMPLE
// resp, err := client.Post(ctx, "/users", WithBody(User{Name: "John", Email: "john@example.com"}))
// if err != nil {
//     // Handle error
// }
//
