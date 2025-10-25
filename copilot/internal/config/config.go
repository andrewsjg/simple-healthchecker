package config

import (
	"fmt"
	"path/filepath"

	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type CheckType string

const (
	CheckPing CheckType = "ping"
	CheckHTTP CheckType = "http"
)

type Check struct {
	Type    CheckType `koanf:"type" json:"type" yaml:"type" toml:"type"`
	Enabled bool      `koanf:"enabled" json:"enabled" yaml:"enabled" toml:"enabled"`
	URL     string    `koanf:"url" json:"url" yaml:"url" toml:"url"`
	Expect  int       `koanf:"expect" json:"expect" yaml:"expect" toml:"expect"`
}

type Host struct {
	Name                string  `koanf:"name" json:"name" yaml:"name" toml:"name"`
	Address             string  `koanf:"address" json:"address" yaml:"address" toml:"address"`
	Checks              []Check `koanf:"checks" json:"checks" yaml:"checks" toml:"checks"`
	HealthchecksPingURL string  `koanf:"healthchecks_ping_url" json:"healthchecks_ping_url" yaml:"healthchecks_ping_url" toml:"healthchecks_ping_url"`
}

type Config struct {
	Hosts []Host `koanf:"hosts" json:"hosts" yaml:"hosts" toml:"hosts"`
}

func Load(path string) (*Config, error) {
	k := koanf.New("")
	ext := filepath.Ext(path)
	var parser koanf.Parser
	switch ext {
	case ".yaml", ".yml":
		parser = yaml.Parser()
	case ".toml":
		parser = toml.Parser()
	default:
		return nil, fmt.Errorf("unsupported config extension: %s", ext)
	}
	if err := k.Load(file.Provider(path), parser); err != nil {
		return nil, err
	}
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, err
	}
	// ensure at least one ping check if none provided
	for i := range cfg.Hosts {
		if len(cfg.Hosts[i].Checks) == 0 {
			cfg.Hosts[i].Checks = []Check{{Type: CheckPing, Enabled: true}}
		}
	}
	return &cfg, nil
}
