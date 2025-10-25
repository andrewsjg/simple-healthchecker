package checker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/andrewsjg/simple-healthchecker/claude/pkg/models"
)

// HTTPChecker implements HTTP health checks
type HTTPChecker struct{}

// NewHTTPChecker creates a new HTTP checker
func NewHTTPChecker() *HTTPChecker {
	return &HTTPChecker{}
}

// Type returns the checker type
func (h *HTTPChecker) Type() models.CheckType {
	return models.CheckTypeHTTP
}

// Check performs an HTTP check on the host
func (h *HTTPChecker) Check(ctx context.Context, host models.Host, check models.Check) models.CheckResult {
	result := models.CheckResult{
		Host:      host.Name,
		CheckType: models.CheckTypeHTTP,
		Timestamp: time.Now(),
	}

	if !check.Enabled {
		result.Success = true
		result.Message = "Check disabled"
		return result
	}

	start := time.Now()

	// Get URL from options, or construct from address
	url := check.Options["url"]
	if url == "" {
		// If no URL specified, construct one from the address
		url = "http://" + host.Address
	}

	// Get expected status code (default: 200)
	expectedStatus := 200
	if statusStr, ok := check.Options["expected_status"]; ok {
		fmt.Sscanf(statusStr, "%d", &expectedStatus)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(check.Timeout),
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Failed to create request: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	// Set User-Agent header
	req.Header.Set("User-Agent", "HealthChecker/1.0")

	// Perform the request
	resp, err := client.Do(req)
	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("HTTP request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode == expectedStatus {
		result.Success = true
		result.Message = fmt.Sprintf("HTTP %d OK (response time: %v)", resp.StatusCode, result.Duration)
	} else {
		result.Success = false
		result.Message = fmt.Sprintf("HTTP %d (expected %d)", resp.StatusCode, expectedStatus)
	}

	return result
}
