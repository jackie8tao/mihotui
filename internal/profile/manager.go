package profile

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackie8tao/mihotui/internal/config"
	"gopkg.in/yaml.v3"
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

// SetActiveProfile persists the active subscription name in appCfg.
func (m *Manager) SetActiveProfile(subName string) error {
	prevActive := m.appCfg.ActiveSubName
	m.appCfg.ActiveSubName = subName
	if err := config.SaveConfig(m.appCfg); err != nil {
		m.appCfg.ActiveSubName = prevActive
		return fmt.Errorf("profile: %w", err)
	}
	return nil
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

	if err := os.MkdirAll(m.SubsDir(), 0755); err != nil {
		return nil, fmt.Errorf("profile: create subscriptions dir %q: %w", m.SubsDir(), err)
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

	targetFilepath := filepath.Join(m.SubsDir(), m.genSubFilename(targetSub.Name))
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
	prevActive := m.appCfg.ActiveSubName
	m.appCfg.Subscriptions = newSubs
	if m.appCfg.ActiveSubName == subName {
		m.appCfg.ActiveSubName = ""
	}
	if err := config.SaveConfig(m.appCfg); err != nil {
		m.appCfg.Subscriptions = prevSubs
		m.appCfg.ActiveSubName = prevActive
		return fmt.Errorf("profile: %w", err)
	}

	targetFilepath := filepath.Join(m.SubsDir(), m.genSubFilename(targetSub.Name))
	if err := os.Remove(targetFilepath); err != nil {
		return fmt.Errorf("profile: remove file %q: %w", targetFilepath, err)
	}
	return nil
}

// parseUserinfo parses the Subscription-Userinfo header.
func parseUserinfo(raw string) *config.SubUserInfo {
	if raw == "" {
		return nil
	}

	info := &config.SubUserInfo{}
	parts := strings.Split(raw, ";")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(kv[0]))
		val, _ := strconv.ParseInt(strings.TrimSpace(kv[1]), 10, 64)

		switch k {
		case "upload":
			info.Upload = val
		case "download":
			info.Download = val
		case "total":
			info.Total = val
		case "expire":
			if val > 0 {
				info.Expire = time.Unix(val, 0)
			}
		}
	}

	return info
}

// downloadAndValidate downloads config from URL, verifies YAML, and writes atomically.
func downloadAndValidate(ctx context.Context, rawURL, targetPath string) (*config.SubUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("User-Agent", "Clash.Meta; mihomo/1.18.0; mihotui")

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	var userInfo *config.SubUserInfo
	if rawUI := resp.Header.Get("subscription-userinfo"); rawUI != "" {
		userInfo = parseUserinfo(rawUI)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var testMap map[string]any
	if err := yaml.Unmarshal(content, &testMap); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	tmpPath := targetPath + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return nil, fmt.Errorf("write temp file %q: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("replace file %q: %w", targetPath, err)
	}

	return userInfo, nil
}
