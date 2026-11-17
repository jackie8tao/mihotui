package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	appConfigMu   sync.RWMutex
	appConfigPath string
)

// ConfigPath returns the path of the currently loaded app config file.
func ConfigPath() string {
	appConfigMu.RLock()
	defer appConfigMu.RUnlock()
	return appConfigPath
}

// SetConfigPath sets the process-level loaded configuration path.
func SetConfigPath(path string) {
	appConfigMu.Lock()
	defer appConfigMu.Unlock()
	appConfigPath = path
}

// AppConfig represents user configuration and state persistence for mihotui.
type AppConfig struct {
	ApiURL                 string         `yaml:"api_url"`                  // external controller API URL
	Secret                 string         `yaml:"secret"`                   // external controller API secret
	ActiveSubscriptionName string         `yaml:"active_subscription_name"` // active subscription name
	Subscriptions          []Subscription `yaml:"subscriptions"`            // managed subscriptions list
}

// SubUserInfo represents subscription traffic quota & expiration header.
type SubUserInfo struct {
	Upload   int64     `json:"upload" yaml:"upload"`
	Download int64     `json:"download" yaml:"download"`
	Total    int64     `json:"total" yaml:"total"`
	Expire   time.Time `json:"expire" yaml:"expire"`
}

// Subscription represents a managed subscription source.
type Subscription struct {
	Name      string       `json:"name" yaml:"name"`
	URL       string       `json:"url" yaml:"url"`
	UpdatedAt time.Time    `json:"updated_at" yaml:"updated_at"`
	UserInfo  *SubUserInfo `json:"user_info,omitempty" yaml:"user_info,omitempty"`
}

// LoadConfig loads configuration from config path.
func LoadConfig(cfgPath string) (*AppConfig, error) {
	if cfgPath == "" {
		return nil, errors.New("config: config path cannot be empty")
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config %q: %w", cfgPath, err)
	}

	cfg := &AppConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config %q: %w", cfgPath, err)
	}

	SetConfigPath(cfgPath)
	return cfg, nil
}

// SaveConfig saves the configuration back to disk atomically using the current ConfigPath().
func SaveConfig(cfg *AppConfig) error {
	targetPath := ConfigPath()
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
	return nil
}
