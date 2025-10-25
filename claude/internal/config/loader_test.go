package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/andrewsjg/simple-healthchecker/claude/pkg/models"
)

func TestLoadConfigYAML(t *testing.T) {
	yamlContent := `
hosts:
  - name: "Google DNS"
    address: "8.8.8.8"
    checks:
      - type: "ping"
        enabled: true
        timeout: 5s
check_interval: 60s
web_server_port: 8080
`

	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Hosts) != 1 {
		t.Errorf("Expected 1 host, got %d", len(cfg.Hosts))
	}

	if cfg.Hosts[0].Name != "Google DNS" {
		t.Errorf("Expected host name 'Google DNS', got '%s'", cfg.Hosts[0].Name)
	}

	if cfg.Hosts[0].Address != "8.8.8.8" {
		t.Errorf("Expected address '8.8.8.8', got '%s'", cfg.Hosts[0].Address)
	}

	if cfg.CheckInterval != models.Duration(60*time.Second) {
		t.Errorf("Expected check interval 60s, got %v", cfg.CheckInterval)
	}

	if cfg.WebServerPort != 8080 {
		t.Errorf("Expected web server port 8080, got %d", cfg.WebServerPort)
	}
}

func TestLoadConfigTOML(t *testing.T) {
	tomlContent := `
check_interval = "60s"
web_server_port = 8080

[[hosts]]
name = "Google DNS"
address = "8.8.8.8"

[[hosts.checks]]
type = "ping"
enabled = true
timeout = "5s"
`

	tmpFile, err := os.CreateTemp("", "config-*.toml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(tomlContent); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	cfg, err := LoadConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Hosts) != 1 {
		t.Errorf("Expected 1 host, got %d", len(cfg.Hosts))
	}

	if cfg.Hosts[0].Name != "Google DNS" {
		t.Errorf("Expected host name 'Google DNS', got '%s'", cfg.Hosts[0].Name)
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *models.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &models.Config{
				Hosts: []models.Host{
					{
						Name:    "test",
						Address: "127.0.0.1",
						Checks: []models.Check{
							{Type: models.CheckTypePing, Enabled: true},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no hosts",
			config: &models.Config{
				Hosts: []models.Host{},
			},
			wantErr: true,
		},
		{
			name: "host without name",
			config: &models.Config{
				Hosts: []models.Host{
					{
						Address: "127.0.0.1",
						Checks: []models.Check{
							{Type: models.CheckTypePing},
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfigUnsupportedFormat(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("{}"); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	tmpFile.Close()

	_, err = LoadConfig(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for unsupported format, got nil")
	}
}

func TestSaveConfig(t *testing.T) {
	cfg := &models.Config{
		Hosts: []models.Host{
			{
				Name:    "Test Host",
				Address: "192.168.1.1",
				Checks: []models.Check{
					{
						Type:    models.CheckTypePing,
						Enabled: true,
						Timeout: models.Duration(5 * time.Second),
					},
				},
			},
		},
		CheckInterval: models.Duration(60 * time.Second),
		WebServerPort: 8080,
	}

	tmpDir, err := os.MkdirTemp("", "config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	yamlPath := filepath.Join(tmpDir, "config.yaml")
	if err := SaveConfig(yamlPath, cfg); err != nil {
		t.Fatalf("Failed to save YAML config: %v", err)
	}

	loadedCfg, err := LoadConfig(yamlPath)
	if err != nil {
		t.Fatalf("Failed to load saved YAML config: %v", err)
	}

	if loadedCfg.Hosts[0].Name != cfg.Hosts[0].Name {
		t.Errorf("Expected host name %s, got %s", cfg.Hosts[0].Name, loadedCfg.Hosts[0].Name)
	}
}
