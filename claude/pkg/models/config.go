package models

import (
	"time"
)

// Duration is a wrapper around time.Duration that supports both YAML and TOML
type Duration time.Duration

// UnmarshalText implements encoding.TextUnmarshaler for Duration
func (d *Duration) UnmarshalText(text []byte) error {
	dur, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// MarshalText implements encoding.TextMarshaler for Duration
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(time.Duration(d).String()), nil
}

// String returns the duration as a string
func (d Duration) String() string {
	return time.Duration(d).String()
}

// Seconds returns the duration as seconds
func (d Duration) Seconds() int {
	return int(time.Duration(d).Seconds())
}

// Config represents the application configuration
type Config struct {
	Hosts            []Host   `yaml:"hosts" toml:"hosts"`
	CheckInterval    Duration `yaml:"check_interval" toml:"check_interval"`
	WebServerPort    int      `yaml:"web_server_port" toml:"web_server_port"`
	EnableConsoleLog bool     `yaml:"enable_console_log" toml:"enable_console_log"`
}

// Host represents a host to monitor
type Host struct {
	Name    string  `yaml:"name" toml:"name"`
	Address string  `yaml:"address" toml:"address"`
	Checks  []Check `yaml:"checks" toml:"checks"`
}

// Check represents a health check configuration
type Check struct {
	Type             CheckType         `yaml:"type" toml:"type"`
	Enabled          bool              `yaml:"enabled" toml:"enabled"`
	Timeout          Duration          `yaml:"timeout" toml:"timeout"`
	HealthcheckIOURL string            `yaml:"healthcheck_io_url,omitempty" toml:"healthcheck_io_url,omitempty"`
	Options          map[string]string `yaml:"options,omitempty" toml:"options,omitempty"`
}

// CheckType represents the type of health check
type CheckType string

const (
	CheckTypePing CheckType = "ping"
	CheckTypeHTTP CheckType = "http"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Host      string
	CheckType CheckType
	Success   bool
	Message   string
	Timestamp time.Time
	Duration  time.Duration
}
