package healthcheckio

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Client handles communication with healthcheck.io
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new healthcheck.io client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendSuccess sends a success signal to healthcheck.io for a specific check
func (c *Client) SendSuccess(ctx context.Context, healthcheckURL string) error {
	if healthcheckURL == "" {
		return nil // Healthcheck.io not configured for this check, skip
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthcheckURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send success: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("healthcheck.io returned error status: %d", resp.StatusCode)
	}

	return nil
}

// SendFailure sends a failure signal to healthcheck.io for a specific check
func (c *Client) SendFailure(ctx context.Context, healthcheckURL string) error {
	if healthcheckURL == "" {
		return nil // Healthcheck.io not configured for this check, skip
	}

	url := fmt.Sprintf("%s/fail", healthcheckURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send failure: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("healthcheck.io returned error status: %d", resp.StatusCode)
	}

	return nil
}
