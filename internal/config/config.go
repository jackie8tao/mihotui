package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// AppConfig represents user configuration for mihomo-tui.
type AppConfig struct {
	ApiURL        string `yaml:"api_url"`
	Secret        string `yaml:"secret"`
	ConfigDir     string `yaml:"config_dir"`      // mihomo config directory (e.g. /etc/mihomo)
	TestURL       string `yaml:"test_url"`
	TestTimeout   int    `yaml:"test_timeout"`    // in milliseconds
	RefreshRateMs int    `yaml:"refresh_rate_ms"` // traffic/stats polling in ms
}

// DefaultConfig returns reasonable defaults.
func DefaultConfig() *AppConfig {
	return &AppConfig{
		ApiURL:        "http://127.0.0.1:9090",
		Secret:        "",
		ConfigDir:     "",
		TestURL:       "http://www.gstatic.com/generate_204",
		TestTimeout:   3000,
		RefreshRateMs: 1000,
	}
}

// LoadConfig loads configuration from standard paths and environment variables.
func LoadConfig(configPath string) (*AppConfig, error) {
	cfg := DefaultConfig()

	// Try reading file if specified or found in default locations
	candidates := []string{}
	if configPath != "" {
		candidates = append(candidates, configPath)
	} else {
		candidates = append(candidates, "config.yaml", filepath.Join("configs", "config.yaml"))
		if home, err := os.UserHomeDir(); err == nil {
			candidates = append(candidates, filepath.Join(home, ".config", "mihomo-tui", "config.yaml"))
		}
	}

	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			_ = yaml.Unmarshal(data, cfg)
			break
		}
	}

	// Environment variable overrides
	if envAPI := os.Getenv("MIHOMO_API_URL"); envAPI != "" {
		cfg.ApiURL = envAPI
	}
	if envSecret := os.Getenv("MIHOMO_SECRET"); envSecret != "" {
		cfg.Secret = envSecret
	}
	if envDir := os.Getenv("MIHOMO_CONFIG_DIR"); envDir != "" {
		cfg.ConfigDir = envDir
	}

	return cfg, nil
}
