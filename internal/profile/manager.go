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
	subsDir   string
	mihomoDir string
	appCfg    *config.AppConfig
}

// NewManager initializes a new Manager for the given subscriptions directory, Mihomo config directory, and app config.
func NewManager(subsDir string, mihomoDir string, appCfg *config.AppConfig) (*Manager, error) {
	if appCfg == nil {
		panic("profile: nil app config")
	}
	subsDir = strings.TrimSpace(subsDir)
	if subsDir == "" {
		return nil, errors.New("profile: subscriptions directory cannot be empty")
	}
	if err := os.MkdirAll(subsDir, 0755); err != nil {
		return nil, fmt.Errorf("profile: create subscriptions dir %q: %w", subsDir, err)
	}

	mihomoDir = strings.TrimSpace(mihomoDir)
	if mihomoDir == "" {
		return nil, errors.New("profile: mihomo directory cannot be empty")
	}
	if err := os.MkdirAll(mihomoDir, 0755); err != nil {
		return nil, fmt.Errorf("profile: create mihomo dir %q: %w", mihomoDir, err)
	}

	return &Manager{
		subsDir:   subsDir,
		appCfg:    appCfg,
		mihomoDir: mihomoDir,
	}, nil
}

// SubsDir returns the subscriptions directory.
func (m *Manager) SubsDir() string {
	return m.subsDir
}

// ActiveProfile returns the active profile name.
func (m *Manager) ActiveProfile() string {
	return m.appCfg.ActiveSubName
}

// SetActiveProfile updates the mihomo config.yaml symlink to point to subName and persists the configuration.
func (m *Manager) SetActiveProfile(subName string) error {
	subFilePath, err := filepath.Abs(filepath.Join(m.subsDir, genSubFilePath(subName)))
	if err != nil {
		return fmt.Errorf("profile: resolve subscription path: %w", err)
	}
	if _, err := os.Stat(subFilePath); err != nil {
		return fmt.Errorf("profile: stat subscription %q: %w", subFilePath, err)
	}

	mihomoFilePath, err := filepath.Abs(filepath.Join(m.mihomoDir, "config.yaml"))
	if err != nil {
		return fmt.Errorf("profile: resolve mihomo config path: %w", err)
	}
	if err := replaceSymlink(subFilePath, mihomoFilePath); err != nil {
		return fmt.Errorf("profile: %w", err)
	}

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
	entries, err := os.ReadDir(m.subsDir)
	if err != nil {
		return nil, fmt.Errorf("profile: read subscriptions dir %q: %w", m.subsDir, err)
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
		filename := entry.Name()
		ext := strings.ToLower(filepath.Ext(filename))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		if strings.HasPrefix(filename, ".") {
			continue
		}
		fileInfo, err := entry.Info()
		if err != nil {
			continue
		}

		subName := strings.TrimSuffix(filename, ext)
		sub := subMap[filename]
		if sub != nil {
			subName = sub.Name
		}

		fullPath, _ := filepath.Abs(filepath.Join(m.subsDir, filename))
		isActive := m.appCfg.ActiveSubName == subName

		profiles = append(profiles, Profile{
			SubName:      subName,
			FileName:     filename,
			FullPath:     fullPath,
			Size:         fileInfo.Size(),
			ModTime:      fileInfo.ModTime(),
			IsActive:     isActive,
			Subscription: sub,
		})
	}

	return profiles, nil
}

// ListSubscriptions returns the configured remote subscriptions from appCfg.
// Note: it returns the backing slice directly; callers must treat it as read-only.
func (m *Manager) ListSubscriptions() []config.Subscription {
	return m.appCfg.Subscriptions
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
		return &subscription, fmt.Errorf("profile: %w", err)
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
		return targetSub, fmt.Errorf("profile: %w", err)
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
		return fmt.Errorf("profile: subscription %q not found", name)
	}

	m.appCfg.Subscriptions = newSubs
	if err := config.SaveConfig(m.appCfg); err != nil {
		return fmt.Errorf("profile: %w", err)
	}
	if deleteFile && fileName != "" {
		targetPath := filepath.Join(m.subsDir, fileName)
		if err := os.Remove(targetPath); err != nil {
			return fmt.Errorf("profile: remove file %q: %w", targetPath, err)
		}
	}
	return nil
}
