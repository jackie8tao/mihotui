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

// SaveFunc is invoked whenever the active profile or subscriptions list changes.
type SaveFunc func(activeProfile string, subs []config.Subscription) error

// ManagerOption configures a Manager instance.
type ManagerOption func(*Manager)

// WithActiveProfile sets the initial active profile name.
func WithActiveProfile(name string) ManagerOption {
	return func(m *Manager) {
		m.activeProfile = name
	}
}

// WithSubscriptions sets the initial subscriptions list.
func WithSubscriptions(subs []config.Subscription) ManagerOption {
	return func(m *Manager) {
		m.subscriptions = append([]config.Subscription(nil), subs...)
	}
}

// WithSaveFunc configures the persistence callback.
func WithSaveFunc(fn SaveFunc) ManagerOption {
	return func(m *Manager) {
		m.saveFn = fn
	}
}

// Manager orchestrates local profiles and remote subscriptions.
type Manager struct {
	configDir     string
	activeProfile string
	subscriptions []config.Subscription
	saveFn        SaveFunc
}

// NewManager initializes a new Manager for the given Mihomo configuration directory.
func NewManager(configDir string, opts ...ManagerOption) (*Manager, error) {
	configDir = strings.TrimSpace(configDir)
	if configDir == "" {
		return nil, errors.New("profile: config directory cannot be empty")
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("profile: invalid config directory %q: %w", configDir, err)
	}

	m := &Manager{
		configDir: configDir,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m, nil
}

func (m *Manager) ConfigDir() string {
	return m.configDir
}

func (m *Manager) ActiveProfile() string {
	return m.activeProfile
}

func (m *Manager) SetActiveProfile(name string) error {
	m.activeProfile = name
	return m.save()
}

func (m *Manager) save() error {
	if m.saveFn != nil {
		return m.saveFn(m.activeProfile, m.subscriptions)
	}
	return nil
}

func (m *Manager) ListSubscriptions() []config.Subscription {
	return m.subscriptions
}

// ListProfiles scans all .yaml/.yml in ConfigDir and correlates with subscriptions.
func (m *Manager) ListProfiles() ([]Profile, error) {
	entries, err := os.ReadDir(m.configDir)
	if err != nil {
		return nil, err
	}

	subMap := make(map[string]*config.Subscription)
	for i := range m.subscriptions {
		subMap[m.subscriptions[i].Name+".yaml"] = &m.subscriptions[i]
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

		fullPath, _ := filepath.Abs(filepath.Join(m.configDir, name))

		profiles = append(profiles, Profile{
			Name:         displayName,
			FileName:     name,
			FullPath:     fullPath,
			Size:         info.Size(),
			ModTime:      info.ModTime(),
			IsActive:     name == m.activeProfile || displayName == m.activeProfile || fullPath == m.activeProfile,
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
	for _, existing := range m.subscriptions {
		if existing.Name == name {
			return nil, fmt.Errorf("profile: subscription %q already exists", name)
		}
	}

	targetPath := filepath.Join(m.configDir, fileName)
	if _, err := os.Stat(targetPath); err == nil {
		return nil, fmt.Errorf("profile: profile file %q already exists", fileName)
	}

	sub := config.Subscription{
		Name:      name,
		URL:       rawURL,
		UpdatedAt: time.Now(),
	}

	// Perform initial download
	userInfo, err := downloadAndValidate(ctx, rawURL, targetPath)
	if err != nil {
		_ = os.Remove(targetPath)
		return nil, fmt.Errorf("profile: download subscription %q: %w", name, err)
	}
	sub.UserInfo = userInfo

	m.subscriptions = append(m.subscriptions, sub)
	if err := m.save(); err != nil {
		return &sub, fmt.Errorf("profile: save config: %w", err)
	}

	return &sub, nil
}

// UpdateSubscription re-downloads an existing subscription.
func (m *Manager) UpdateSubscription(ctx context.Context, name string) (*config.Subscription, error) {
	var targetSub *config.Subscription
	var subIndex int
	for i := range m.subscriptions {
		if m.subscriptions[i].Name == name {
			targetSub = &m.subscriptions[i]
			subIndex = i
			break
		}
	}
	if targetSub == nil {
		return nil, fmt.Errorf("profile: subscription %q not found", name)
	}

	targetPath := filepath.Join(m.configDir, targetSub.Name+".yaml")
	userInfo, err := downloadAndValidate(ctx, targetSub.URL, targetPath)
	if err != nil {
		return nil, fmt.Errorf("profile: update subscription %q: %w", targetSub.Name, err)
	}

	m.subscriptions[subIndex].UpdatedAt = time.Now()
	if userInfo != nil {
		m.subscriptions[subIndex].UserInfo = userInfo
	}

	if err := m.save(); err != nil {
		return &m.subscriptions[subIndex], fmt.Errorf("profile: save config: %w", err)
	}

	return &m.subscriptions[subIndex], nil
}

// DeleteSubscription removes subscription record and optionally deletes local file.
func (m *Manager) DeleteSubscription(name string, deleteFile bool) error {
	var fileName string
	found := false
	var newSubs []config.Subscription

	for _, sub := range m.subscriptions {
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

	m.subscriptions = newSubs
	if err := m.save(); err != nil {
		return fmt.Errorf("subscription deleted but failed to save config: %w", err)
	}

	if deleteFile && fileName != "" {
		_ = os.Remove(filepath.Join(m.configDir, fileName))
	}
	return nil
}
