package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	defaultConfigDir  = "mihotui"
	defaultConfigFile = "config.yaml"
)

var (
	loadedConfigMu   sync.RWMutex
	loadedConfigPath string
)

// LoadedConfigPath returns the path of the currently loaded config file,
// falling back to DefaultConfigPath() if not yet set.
func LoadedConfigPath() string {
	loadedConfigMu.RLock()
	defer loadedConfigMu.RUnlock()
	if loadedConfigPath != "" {
		return loadedConfigPath
	}
	return DefaultConfigPath()
}

// SetLoadedConfigPath sets the process-level loaded configuration path.
func SetLoadedConfigPath(path string) {
	loadedConfigMu.Lock()
	defer loadedConfigMu.Unlock()
	loadedConfigPath = path
}

// AppConfig represents user configuration and state persistence for mihotui.
type AppConfig struct {
	ApiURL        string         `yaml:"api_url"`        // external controller API URL
	Secret        string         `yaml:"secret"`         // external controller API secret
	ActiveProfile string         `yaml:"active_profile"` // active profile file name
	Subscriptions []Subscription `yaml:"subscriptions"`  // managed subscriptions list
}

// DefaultConfig returns reasonable defaults.
func DefaultConfig() *AppConfig {
	return &AppConfig{
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

// LoadConfig loads configuration from standard paths and tracks the loaded path.
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
		SetLoadedConfigPath(configPath)
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
		SetLoadedConfigPath(p)
		return cfg, nil
	}

	SetLoadedConfigPath(DefaultConfigPath())
	return cfg, nil
}

// SaveConfig saves the configuration back to disk atomically using the current LoadedConfigPath().
func SaveConfig(cfg *AppConfig) error {
	return SaveConfigTo(cfg, LoadedConfigPath())
}

// SaveConfigTo saves the configuration to targetPath atomically and updates LoadedConfigPath.
func SaveConfigTo(cfg *AppConfig, targetPath string) error {
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

	SetLoadedConfigPath(targetPath)
	return nil
}
