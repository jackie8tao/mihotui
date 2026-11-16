package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

var (
	loadedConfigMu   sync.RWMutex
	loadedConfigPath string
)

// LoadedConfigPath returns the path of the currently loaded config file.
func LoadedConfigPath() string {
	loadedConfigMu.RLock()
	defer loadedConfigMu.RUnlock()
	return loadedConfigPath
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

// LoadConfig loads configuration from explicitPath if specified (failing fast if missing or invalid).
// Otherwise it loads from defaultPath (the suite-local config path <appRoot>/etc/mihotui/config.yaml),
// initializing in-memory defaults bound to defaultPath if the file does not yet exist.
func LoadConfig(explicitPath, defaultPath string) (*AppConfig, error) {
	if explicitPath != "" {
		return loadStrictConfigFile(explicitPath)
	}

	if defaultPath == "" {
		return nil, errors.New("config: default config path cannot be empty")
	}

	cfg, err := loadStrictConfigFile(defaultPath)
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	SetLoadedConfigPath(defaultPath)
	return DefaultConfig(), nil
}

func loadStrictConfigFile(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config %q: %w", path, err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config %q: %w", path, err)
	}

	SetLoadedConfigPath(path)
	return cfg, nil
}

// SaveConfig saves the configuration back to disk atomically using the current LoadedConfigPath().
func SaveConfig(cfg *AppConfig) error {
	return SaveConfigTo(cfg, LoadedConfigPath())
}

// SaveConfigTo saves the configuration to targetPath atomically and updates LoadedConfigPath.
func SaveConfigTo(cfg *AppConfig, targetPath string) error {
	if targetPath == "" {
		return errors.New("config: target config path cannot be empty")
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
