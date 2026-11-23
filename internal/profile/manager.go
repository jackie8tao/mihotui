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
	appPaths *config.AppPaths
	appCfg   *config.AppConfig
}

// NewManager initializes a new Manager for the given app config and app paths.
func NewManager(appCfg *config.AppConfig, appPaths *config.AppPaths) (*Manager, error) {
	if appCfg == nil {
		panic("profile: nil app config")
	}
	if appPaths == nil {
		panic("profile: nil app paths")
	}
	return &Manager{
		appCfg:   appCfg,
		appPaths: appPaths,
	}, nil
}

// SubsDir returns the subscriptions directory.
func (m *Manager) SubsDir() string {
	return filepath.Join(m.appPaths.AppEtcDir(), "subscriptions")
}

// ActiveProfile returns the active profile name.
func (m *Manager) ActiveProfile() string {
	return m.appCfg.ActiveSubName
}

// SetActiveProfile updates the mihomo config.yaml symlink to point to subName and persists the configuration.
func (m *Manager) SetActiveProfile(subName string) error {
	prevActive := m.appCfg.ActiveSubName
	m.appCfg.ActiveSubName = subName
	if err := config.SaveConfig(m.appCfg); err != nil {
		m.appCfg.ActiveSubName = prevActive
		return fmt.Errorf("profile: %w", err)
	}
	return nil
}

// ListProfiles scans all .yaml/.yml in subsDir and correlates with subscriptions.
func (m *Manager) ListProfiles() ([]Profile, error) {
	return []Profile{}, nil
}

// ListSubscriptions returns the configured remote subscriptions from appCfg.
// Note: it returns the backing slice directly; callers must treat it as read-only.
func (m *Manager) ListSubscriptions() []config.Subscription {
	return m.appCfg.Subscriptions
}

var validSubNameRe = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

// AddSubscription registers a new subscription, downloads it, and saves metadata.
func (m *Manager) AddSubscription(ctx context.Context, subName, rawURL string) (*config.Subscription, error) {
	if subName == "" {
		return nil, errors.New("profile: subscription name cannot be empty")
	}
	if !validSubNameRe.MatchString(subName) {
		return nil, fmt.Errorf("profile: invalid subscription name %q (only letters, digits, '-' and '_' are allowed)", subName)
	}
	if rawURL == "" {
		return nil, errors.New("profile: subscription URL cannot be empty")
	}

	for _, existing := range m.appCfg.Subscriptions {
		if existing.Name == subName {
			return nil, fmt.Errorf("profile: subscription %q already exists", subName)
		}
	}

	targetFilename := m.genSubFilename(subName)
	targetFilepath := filepath.Join(m.SubsDir(), targetFilename)
	if _, err := os.Stat(targetFilepath); err == nil {
		return nil, fmt.Errorf("profile: profile file %q already exists", targetFilename)
	}

	userInfo, err := downloadAndValidate(ctx, rawURL, targetFilepath)
	if err != nil {
		_ = os.Remove(targetFilepath)
		return nil, fmt.Errorf("profile: download subscription %q: %w", subName, err)
	}
	subscription := config.Subscription{
		Name:      subName,
		URL:       rawURL,
		UpdatedAt: time.Now(),
		UserInfo:  userInfo,
	}
	m.appCfg.Subscriptions = append(m.appCfg.Subscriptions, subscription)
	if err := config.SaveConfig(m.appCfg); err != nil {
		return nil, fmt.Errorf("profile: %w", err)
	}

	return &subscription, nil
}

func (m *Manager) genSubFilename(subName string) string {
	return fmt.Sprintf("%s.yaml", strings.TrimSpace(subName))
}

// UpdateSubscription re-downloads an existing subscription.
func (m *Manager) UpdateSubscription(ctx context.Context, subName string) (*config.Subscription, error) {
	var targetSub *config.Subscription
	for i := range m.appCfg.Subscriptions {
		if m.appCfg.Subscriptions[i].Name == subName {
			targetSub = &m.appCfg.Subscriptions[i]
			break
		}
	}
	if targetSub == nil {
		return nil, fmt.Errorf("profile: subscription %q not found", subName)
	}

	targetFilepath := filepath.Join(m.SubsDir(), targetSub.Name)
	userInfo, err := downloadAndValidate(ctx, targetSub.URL, targetFilepath)
	if err != nil {
		return nil, fmt.Errorf("profile: update subscription %q: %w", targetSub.Name, err)
	}

	prevUserInfo := targetSub.UserInfo

	targetSub.UpdatedAt = time.Now()
	if userInfo != nil {
		targetSub.UserInfo = userInfo
	}
	if err := config.SaveConfig(m.appCfg); err != nil {
		targetSub.UserInfo = prevUserInfo
		return nil, fmt.Errorf("profile: %w", err)
	}
	return targetSub, nil
}

// DeleteSubscription removes subscription record and deletes local file.
func (m *Manager) DeleteSubscription(subName string) error {
	var newSubs []config.Subscription
	var targetSub *config.Subscription
	for i := range m.appCfg.Subscriptions {
		if m.appCfg.Subscriptions[i].Name == subName {
			targetSub = &m.appCfg.Subscriptions[i]
			continue
		}
		newSubs = append(newSubs, m.appCfg.Subscriptions[i])
	}
	if targetSub == nil {
		return fmt.Errorf("profile: subscription %q not found", subName)
	}

	prevSubs := m.appCfg.Subscriptions
	m.appCfg.Subscriptions = newSubs
	if err := config.SaveConfig(m.appCfg); err != nil {
		m.appCfg.Subscriptions = prevSubs
		return fmt.Errorf("profile: %w", err)
	}

	targetFilename := m.genSubFilename(targetSub.Name)
	targetFilepath := filepath.Join(m.SubsDir(), targetFilename)
	if err := os.Remove(targetFilepath); err != nil {
		m.appCfg.Subscriptions = prevSubs
		return fmt.Errorf("profile: remove file %q: %w", targetFilepath, err)
	}
	return nil
}
