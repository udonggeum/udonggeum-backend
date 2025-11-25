package kakaopay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a Kakao Pay API client
type Client struct {
	config     Config
	httpClient *http.Client
}

// NewClient creates a new Kakao Pay client with the given configuration
func NewClient(config Config) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Create HTTP client with reasonable timeout
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &Client{
		config:     config,
		httpClient: httpClient,
	}, nil
}

// GetConfig returns the client configuration
func (c *Client) GetConfig() Config {
	return c.config
}

// Ready initiates a payment process
func (c *Client) Ready(ctx context.Context, req ReadyRequest) (*ReadyResponse, error) {
	// Set CID from config
	req.CID = c.config.CID

	// Use callback URLs from request if provided, otherwise use config defaults
	if req.ApprovalURL == "" {
		req.ApprovalURL = c.config.ApprovalURL
	}
	if req.FailURL == "" {
		req.FailURL = c.config.FailURL
	}
	if req.CancelURL == "" {
		req.CancelURL = c.config.CancelURL
	}

	resp, err := c.doRequest(ctx, "ready", req)
	if err != nil {
		return nil, fmt.Errorf("failed to make ready request: %w", err)
	}

	var readyResp ReadyResponse
	if err := json.Unmarshal(resp, &readyResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ready response: %w", err)
	}

	return &readyResp, nil
}

// Approve approves a payment process
func (c *Client) Approve(ctx context.Context, req ApproveRequest) (*ApproveResponse, error) {
	// Set CID from config
	req.CID = c.config.CID

	resp, err := c.doRequest(ctx, "approve", req)
	if err != nil {
		return nil, fmt.Errorf("failed to make approve request: %w", err)
	}

	var approveResp ApproveResponse
	if err := json.Unmarshal(resp, &approveResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal approve response: %w", err)
	}

	return &approveResp, nil
}

// Cancel cancels a payment
func (c *Client) Cancel(ctx context.Context, req CancelRequest) (*CancelResponse, error) {
	// Set CID from config
	req.CID = c.config.CID

	resp, err := c.doRequest(ctx, "cancel", req)
	if err != nil {
		return nil, fmt.Errorf("failed to make cancel request: %w", err)
	}

	var cancelResp CancelResponse
	if err := json.Unmarshal(resp, &cancelResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cancel response: %w", err)
	}

	return &cancelResp, nil
}

// doRequest performs an HTTP request to the Kakao Pay API
func (c *Client) doRequest(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("%s/%s", c.config.BaseURL, endpoint)

	// Log the request for debugging (mask sensitive data)
	fmt.Printf("[KakaoPay] Request to %s: %s\n", url, string(reqBody))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers for Kakao Pay API
	// Always use SECRET_KEY for both production and development
	authHeader := "SECRET_KEY " + c.config.AdminKey
	fmt.Printf("[KakaoPay] Authorization: SECRET_KEY [REDACTED] (key length: %d)\n", len(c.config.AdminKey))
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetworkError, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err != nil {
			// If we can't parse the error response, return a generic error
			return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
		}

		// Log the full error for debugging
		errorMsg := fmt.Sprintf("Kakao Pay API error - Status: %d, Code: %d, Message: %s, Body: %s",
			resp.StatusCode, errResp.Code, errResp.Message, string(body))

		// Map common error codes to custom errors
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return nil, fmt.Errorf("%w: %s", ErrUnauthorized, errorMsg)
		case http.StatusBadRequest:
			return nil, fmt.Errorf("%w: %s", ErrInvalidRequest, errorMsg)
		default:
			return nil, fmt.Errorf("%w: %s", ErrPaymentFailed, errorMsg)
		}
	}

	return body, nil
}
