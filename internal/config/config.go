package config

import (
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
	ApiURL        string `yaml:"api_url"`
	Secret        string `yaml:"secret"`
	ConfigDir     string `yaml:"config_dir"` // mihomo config directory (e.g. /etc/mihomo)
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

	// 1. Explicitly specified configuration path (fail fast if invalid)
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read specified config %q: %w", configPath, err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse specified config %q: %w", configPath, err)
		}
		applyEnvOverrides(cfg)
		return cfg, nil
	}

	// 2. Standard distribution discovery paths: User Config (XDG) > System-wide (/etc)
	var candidates []string

	// User-level configuration ($XDG_CONFIG_HOME/mihomo-tui/config.yaml or ~/.config/mihomo-tui/config.yaml)
	if userConfigDir, err := os.UserConfigDir(); err == nil {
		candidates = append(candidates, filepath.Join(userConfigDir, defaultConfigDir, defaultConfigFile))
	}

	// System-wide global configuration (/etc/mihomo-tui/config.yaml)
	candidates = append(candidates, filepath.Join("/etc", defaultConfigDir, defaultConfigFile))

	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			_ = yaml.Unmarshal(data, cfg)
			break
		}
	}

	// 3. Environment variable overrides
	applyEnvOverrides(cfg)

	return cfg, nil
}

func applyEnvOverrides(cfg *AppConfig) {
	if envAPI := os.Getenv("MIHOMO_API_URL"); envAPI != "" {
		cfg.ApiURL = envAPI
	}
	if envSecret := os.Getenv("MIHOMO_SECRET"); envSecret != "" {
		cfg.Secret = envSecret
	}
	if envDir := os.Getenv("MIHOMO_CONFIG_DIR"); envDir != "" {
		cfg.ConfigDir = envDir
	}
}
