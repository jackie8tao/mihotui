package profile

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jackie8tao/mihotui/internal/config"
)

// Manager orchestrates local profiles and remote subscriptions.
type Manager struct {
	subsDir string
	appCfg  *config.AppConfig
}

// NewManager initializes a new Manager for the given Mihomo configuration directory and app config.
func NewManager(subsDir string, appCfg *config.AppConfig) (*Manager, error) {
	if appCfg == nil {
		panic("profile: nil app config")
	}
	subsDir = strings.TrimSpace(subsDir)
	if subsDir == "" {
		return nil, errors.New("profile: config directory cannot be empty")
	}
	if err := os.MkdirAll(subsDir, 0755); err != nil {
		return nil, fmt.Errorf("profile: invalid config directory %q: %w", subsDir, err)
	}

	return &Manager{
		subsDir: subsDir,
		appCfg:  appCfg,
	}, nil
}

func (m *Manager) ConfigDir() string {
	return m.subsDir
}

func (m *Manager) ActiveProfile() string {
	return m.appCfg.ActiveSubName
}

func (m *Manager) SetActiveProfile(name string) error {
	m.appCfg.ActiveSubName = name
	return config.SaveConfig(m.appCfg)
}

func (m *Manager) ListSubscriptions() []config.Subscription {
	return m.appCfg.Subscriptions
}

// ListProfiles scans all .yaml/.yml in ConfigDir and correlates with subscriptions.
func (m *Manager) ListProfiles() ([]Profile, error) {
	entries, err := os.ReadDir(m.subsDir)
	if err != nil {
		return nil, err
	}

	subMap := make(map[string]*config.Subscription)
	for i := range m.appCfg.Subscriptions {
		subMap[m.appCfg.Subscriptions[i].Name+".yaml"] = &m.appCfg.Subscriptions[i]
	}

	var profiles []Profile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		if strings.HasPrefix(name, ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		displayName := strings.TrimSuffix(name, ext)
		sub := subMap[name]
		if sub != nil {
			displayName = sub.Name
		}

		fullPath, _ := filepath.Abs(filepath.Join(m.subsDir, name))
		active := m.appCfg.ActiveSubName

		profiles = append(profiles, Profile{
			Name:         displayName,
			FileName:     name,
			FullPath:     fullPath,
			Size:         info.Size(),
			ModTime:      info.ModTime(),
			IsActive:     name == active || displayName == active || fullPath == active,
			Subscription: sub,
		})
	}

	return profiles, nil
}

var validSubNameRe = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

// AddSubscription registers a new subscription, downloads it, and saves metadata.
func (m *Manager) AddSubscription(ctx context.Context, name, rawURL string) (*config.Subscription, error) {
	if name == "" {
		return nil, errors.New("profile: subscription name cannot be empty")
	}
	if !validSubNameRe.MatchString(name) {
		return nil, fmt.Errorf("profile: invalid subscription name %q (only letters, digits, '-' and '_' are allowed)", name)
	}
	if rawURL == "" {
		return nil, errors.New("profile: subscription URL cannot be empty")
	}

	fileName := name + ".yaml"
	for _, existing := range m.appCfg.Subscriptions {
		if existing.Name == name {
			return nil, fmt.Errorf("profile: subscription %q already exists", name)
		}
	}

	targetPath := filepath.Join(m.subsDir, fileName)
	if _, err := os.Stat(targetPath); err == nil {
		return nil, fmt.Errorf("profile: profile file %q already exists", fileName)
	}

	userInfo, err := downloadAndValidate(ctx, rawURL, targetPath)
	if err != nil {
		_ = os.Remove(targetPath)
		return nil, fmt.Errorf("profile: download subscription %q: %w", name, err)
	}
	subscription := config.Subscription{
		Name:      name,
		URL:       rawURL,
		UpdatedAt: time.Now(),
		UserInfo:  userInfo,
	}

	m.appCfg.Subscriptions = append(m.appCfg.Subscriptions, subscription)
	if err := config.SaveConfig(m.appCfg); err != nil {
		return &subscription, fmt.Errorf("profile: save config: %w", err)
	}

	return &subscription, nil
}

// UpdateSubscription re-downloads an existing subscription.
func (m *Manager) UpdateSubscription(ctx context.Context, name string) (*config.Subscription, error) {
	var targetSub *config.Subscription
	for i := range m.appCfg.Subscriptions {
		if m.appCfg.Subscriptions[i].Name == name {
			targetSub = &m.appCfg.Subscriptions[i]
			break
		}
	}
	if targetSub == nil {
		return nil, fmt.Errorf("profile: subscription %q not found", name)
	}

	targetPath := filepath.Join(m.subsDir, targetSub.Name+".yaml")
	userInfo, err := downloadAndValidate(ctx, targetSub.URL, targetPath)
	if err != nil {
		return nil, fmt.Errorf("profile: update subscription %q: %w", targetSub.Name, err)
	}
	targetSub.UpdatedAt = time.Now()
	if userInfo != nil {
		targetSub.UserInfo = userInfo
	}

	if err := config.SaveConfig(m.appCfg); err != nil {
		return targetSub, fmt.Errorf("profile: save config: %w", err)
	}
	return targetSub, nil
}

// DeleteSubscription removes subscription record and optionally deletes local file.
func (m *Manager) DeleteSubscription(name string, deleteFile bool) error {
	var fileName string
	var newSubs []config.Subscription

	found := false
	for _, sub := range m.appCfg.Subscriptions {
		if sub.Name == name {
			found = true
			fileName = sub.Name + ".yaml"
			continue
		}
		newSubs = append(newSubs, sub)
	}
	if !found {
		return fmt.Errorf("subscription %q not found", name)
	}

	m.appCfg.Subscriptions = newSubs
	if err := config.SaveConfig(m.appCfg); err != nil {
		return fmt.Errorf("subscription deleted but failed to save config: %w", err)
	}
	if deleteFile && fileName != "" {
		return os.Remove(filepath.Join(m.subsDir, fileName))
	}
	return nil
}
