package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/jackie8tao/mihotui/internal/profile"
)

const (
	defaultConfigDir  = "mihotui"
	defaultConfigFile = "config.yaml"
)

// AppConfig represents user configuration and state persistence for mihotui.
type AppConfig struct {
	MihomoDir     string                 `yaml:"mihomo_dir"`     // mihomo managed config directory
	ApiURL        string                 `yaml:"api_url"`        // external controller API URL
	Secret        string                 `yaml:"secret"`         // external controller API secret
	ActiveProfile string                 `yaml:"active_profile"` // active profile file name
	Subscriptions []profile.Subscription `yaml:"subscriptions"`  // managed subscriptions list

	LoadedPath string `yaml:"-"` // file path from which this config was loaded or will be saved
}

// DefaultConfig returns reasonable defaults.
func DefaultConfig() *AppConfig {
	return &AppConfig{
		MihomoDir:     "",
		ApiURL:        "http://127.0.0.1:9090",
		Secret:        "",
		ActiveProfile: "",
		Subscriptions: nil,
	}
}

// DefaultConfigPath returns the canonical path for config.yaml.
func DefaultConfigPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, defaultConfigDir, defaultConfigFile)
	}
	if userConfigDir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(userConfigDir, defaultConfigDir, defaultConfigFile)
	}
	return filepath.Join(".", defaultConfigFile)
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
		cfg.LoadedPath = configPath
		return cfg, nil
	}

	// 2. Standard distribution discovery paths: User Config (XDG)
	var candidates []string

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates, filepath.Join(xdg, defaultConfigDir, defaultConfigFile))
	}

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
		cfg.LoadedPath = p
		return cfg, nil
	}

	cfg.LoadedPath = DefaultConfigPath()
	return cfg, nil
}

// SaveConfig saves the configuration back to disk atomically.
func SaveConfig(cfg *AppConfig) error {
	targetPath := cfg.LoadedPath
	if targetPath == "" {
		targetPath = DefaultConfigPath()
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	tmpPath := targetPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp config file: %w", err)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to replace config file: %w", err)
	}

	cfg.LoadedPath = targetPath
	return nil
}
