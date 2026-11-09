package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	defaultConfigDir  = "mihotui"
	defaultConfigFile = "config.yaml"
)

// AppConfig represents user configuration for mihomo-tui.
type AppConfig struct {
	MihomoDir string `yaml:"mihomo_dir"` // mihomo managed config directory
	ApiURL    string `yaml:"api_url"`
	Secret    string `yaml:"secret"`
}

// DefaultConfig returns reasonable defaults.
func DefaultConfig() *AppConfig {
	return &AppConfig{
		MihomoDir: "",
		ApiURL:    "http://127.0.0.1:9090",
		Secret:    "",
	}
}

// LoadConfig loads configuration from standard paths.
func LoadConfig(configPath string) (*AppConfig, error) {
	cfg := DefaultConfig()

	// 1. Explicitly specified configuration path (fail fast if invalid)
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read specified config %q: %w", configPath, err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse specified config %q: %w", configPath, err)
		}
		return cfg, nil
	}

	// 2. Standard distribution discovery paths: User Config (XDG)
	var candidates []string

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates, filepath.Join(xdg, defaultConfigDir, defaultConfigFile))
	}

	// User-level configuration ($XDG_CONFIG_HOME/mihotui/config.yaml or OS UserConfigDir)
	if userConfigDir, err := os.UserConfigDir(); err == nil {
		candidates = append(candidates, filepath.Join(userConfigDir, defaultConfigDir, defaultConfigFile))
	}

	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("failed to read config %q: %w", p, err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config %q: %w", p, err)
		}
		break
	}

	return cfg, nil
}
