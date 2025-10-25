package checker

import (
	"context"
	"fmt"
	"time"

	"github.com/andrewsjg/simple-healthchecker/claude/pkg/models"
	"github.com/go-ping/ping"
)

// PingChecker implements ICMP ping health checks
type PingChecker struct{}

// NewPingChecker creates a new ping checker
func NewPingChecker() *PingChecker {
	return &PingChecker{}
}

// Type returns the checker type
func (p *PingChecker) Type() models.CheckType {
	return models.CheckTypePing
}

// Check performs a ping check on the host
func (p *PingChecker) Check(ctx context.Context, host models.Host, check models.Check) models.CheckResult {
	result := models.CheckResult{
		Host:      host.Name,
		CheckType: models.CheckTypePing,
		Timestamp: time.Now(),
	}

	if !check.Enabled {
		result.Success = true
		result.Message = "Check disabled"
		return result
	}

	start := time.Now()

	pinger, err := ping.NewPinger(host.Address)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Failed to create pinger: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	// Set privileged mode to false to use UDP instead of ICMP (works without root)
	pinger.SetPrivileged(false)
	pinger.Count = 1
	pinger.Timeout = time.Duration(check.Timeout)

	// Run ping with context
	done := make(chan bool)
	go func() {
		err = pinger.Run()
		done <- true
	}()

	select {
	case <-ctx.Done():
		pinger.Stop()
		result.Success = false
		result.Message = "Check cancelled"
		result.Duration = time.Since(start)
		return result
	case <-done:
		// Continue processing
	}

	result.Duration = time.Since(start)

	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Ping failed: %v", err)
		return result
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv > 0 {
		result.Success = true
		result.Message = fmt.Sprintf("Ping successful (rtt: %v)", stats.AvgRtt)
	} else {
		result.Success = false
		result.Message = "No packets received"
	}

	return result
}
