package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/andrewsjg/simple-healthchecker/claude/pkg/models"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// LoadConfig loads configuration from a YAML or TOML file
func LoadConfig(path string) (*models.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	var cfg models.Config

	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".toml":
		if err := toml.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse TOML config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s (use .yaml, .yml, or .toml)", ext)
	}

	// Set defaults
	if cfg.CheckInterval == 0 {
		cfg.CheckInterval = models.Duration(60 * time.Second) // 60 seconds default
	}
	if cfg.WebServerPort == 0 {
		cfg.WebServerPort = 8080
	}
	// EnableConsoleLog defaults to false (zero value)

	// Validate configuration
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// validateConfig validates the configuration
func validateConfig(cfg *models.Config) error {
	if len(cfg.Hosts) == 0 {
		return fmt.Errorf("no hosts configured")
	}

	for i, host := range cfg.Hosts {
		if host.Name == "" {
			return fmt.Errorf("host at index %d has no name", i)
		}
		if host.Address == "" {
			return fmt.Errorf("host %s has no address", host.Name)
		}
		if len(host.Checks) == 0 {
			return fmt.Errorf("host %s has no checks configured", host.Name)
		}

		for j, check := range host.Checks {
			if check.Type == "" {
				return fmt.Errorf("host %s check at index %d has no type", host.Name, j)
			}
			if check.Timeout == 0 {
				cfg.Hosts[i].Checks[j].Timeout = models.Duration(5 * time.Second) // 5 seconds default
			}
		}
	}

	return nil
}

// SaveConfig saves the configuration to a file
func SaveConfig(path string, cfg *models.Config) error {
	ext := strings.ToLower(filepath.Ext(path))

	var data []byte
	var err error

	switch ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML config: %w", err)
		}
	case ".toml":
		data, err = toml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal TOML config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
