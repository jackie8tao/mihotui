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

	"gopkg.in/yaml.v3"
)

type SubStore struct {
	ConfigDir     string         `yaml:"config_dir"`
	ActiveProfile string         `yaml:"active_profile"`
	Subscriptions []Subscription `yaml:"subscriptions"`
}

type Manager struct {
	configDir     string
	metaPath      string
	activeProfile string
	subscriptions []Subscription
	httpClient    *http.Client
}

func NewManager(customConfigDir string) (*Manager, error) {
	configDir := resolveMihomoConfigDir(customConfigDir)
	metaPath := resolveMetaPath()

	m := &Manager{
		configDir: configDir,
		metaPath:  metaPath,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	if err := m.loadMeta(); err != nil {
		// If load fails because not exists, that's fine
	}

	return m, nil
}

// resolveMihomoConfigDir ensures and returns the configuration directory for Mihomo.
// If custom is specified in configuration, it is used directly without probing system paths.
// Otherwise, it defaults deterministically to the application-managed Mihomo directory.
func resolveMihomoConfigDir(custom string) string {
	if strings.TrimSpace(custom) != "" {
		_ = os.MkdirAll(custom, 0755)
		return custom
	}

	// Deterministic fallback: application-managed directory, avoiding heuristic system guesses
	if userConfigDir, err := os.UserConfigDir(); err == nil {
		dir := filepath.Join(userConfigDir, "mihotui", "mihomo")
		_ = os.MkdirAll(dir, 0755)
		return dir
	}

	defaultDir := filepath.Join(".", "mihomo")
	_ = os.MkdirAll(defaultDir, 0755)
	return defaultDir
}

func resolveMetaPath() string {
	if userConfigDir, err := os.UserConfigDir(); err == nil {
		dir := filepath.Join(userConfigDir, "mihotui")
		_ = os.MkdirAll(dir, 0755)
		return filepath.Join(dir, "subscriptions.yaml")
	}
	return "subscriptions.yaml"
}

func (m *Manager) ConfigDir() string {
	return m.configDir
}

func (m *Manager) ActiveProfile() string {
	return m.activeProfile
}

func (m *Manager) SetActiveProfile(name string) error {
	m.activeProfile = name
	return m.saveMeta()
}

func (m *Manager) loadMeta() error {
	data, err := os.ReadFile(m.metaPath)
	if err != nil {
		return err
	}
	var store SubStore
	if err := yaml.Unmarshal(data, &store); err != nil {
		return err
	}

	if store.ConfigDir != "" {
		m.configDir = store.ConfigDir
	}
	m.activeProfile = store.ActiveProfile
	m.subscriptions = store.Subscriptions
	return nil
}

func (m *Manager) saveMeta() error {
	store := SubStore{
		ConfigDir:     m.configDir,
		ActiveProfile: m.activeProfile,
		Subscriptions: m.subscriptions,
	}
	data, err := yaml.Marshal(store)
	if err != nil {
		return err
	}
	return os.WriteFile(m.metaPath, data, 0644)
}

func (m *Manager) ListSubscriptions() []Subscription {
	return m.subscriptions
}

// ListProfiles scans all .yaml/.yml in ConfigDir and correlates with subscriptions.
func (m *Manager) ListProfiles() ([]Profile, error) {
	entries, err := os.ReadDir(m.configDir)
	if err != nil {
		return nil, err
	}

	subMap := make(map[string]*Subscription)
	for i := range m.subscriptions {
		subMap[m.subscriptions[i].FileName] = &m.subscriptions[i]
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
		if name == "subscriptions.yaml" || strings.HasPrefix(name, ".") {
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
			IsActive:     name == m.activeProfile || fullPath == m.activeProfile,
			Subscription: sub,
		})
	}

	return profiles, nil
}

// ParseUserinfo parses the Subscription-Userinfo header.
// e.g. upload=1073741824; download=21474836480; total=107374182400; expire=1735689600
func ParseUserinfo(raw string) *SubUserInfo {
	if raw == "" {
		return nil
	}

	info := &SubUserInfo{}
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

var safeFileNameRe = regexp.MustCompile(`[^a-zA-Z0-9_\-]+`)

// sanitizeFileName converts arbitrary name to safe alphanumeric file name.
func sanitizeFileName(name string) string {
	clean := safeFileNameRe.ReplaceAllString(name, "_")
	clean = strings.Trim(clean, "_")
	if clean == "" {
		clean = fmt.Sprintf("sub_%d", time.Now().Unix())
	}
	return clean + ".yaml"
}

// AddSubscription registers a new subscription, downloads it, and saves metadata.
func (m *Manager) AddSubscription(ctx context.Context, name, rawURL string) (*Subscription, error) {
	if name == "" {
		return nil, errors.New("subscription name cannot be empty")
	}
	if rawURL == "" {
		return nil, errors.New("subscription URL cannot be empty")
	}

	fileName := sanitizeFileName(name)
	targetPath := filepath.Join(m.configDir, fileName)

	// Avoid duplicate file collision
	counter := 1
	for {
		if _, err := os.Stat(targetPath); errors.Is(err, os.ErrNotExist) {
			break
		}
		fileName = fmt.Sprintf("%s_%d.yaml", strings.TrimSuffix(sanitizeFileName(name), ".yaml"), counter)
		targetPath = filepath.Join(m.configDir, fileName)
		counter++
	}

	sub := Subscription{
		ID:        fmt.Sprintf("sub_%d", time.Now().UnixNano()),
		Name:      name,
		URL:       rawURL,
		FileName:  fileName,
		UpdatedAt: time.Now(),
	}

	// Perform initial download
	userInfo, err := m.downloadAndValidate(ctx, rawURL, targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to download subscription: %w", err)
	}
	sub.UserInfo = userInfo
	sub.UpdatedAt = time.Now()

	m.subscriptions = append(m.subscriptions, sub)
	_ = m.saveMeta()

	return &sub, nil
}

// UpdateSubscription re-downloads an existing subscription.
func (m *Manager) UpdateSubscription(ctx context.Context, subID string) (*Subscription, error) {
	var targetSub *Subscription
	for i := range m.subscriptions {
		if m.subscriptions[i].ID == subID || m.subscriptions[i].FileName == subID {
			targetSub = &m.subscriptions[i]
			break
		}
	}
	if targetSub == nil {
		return nil, fmt.Errorf("subscription not found: %s", subID)
	}

	targetPath := filepath.Join(m.configDir, targetSub.FileName)
	userInfo, err := m.downloadAndValidate(ctx, targetSub.URL, targetPath)
	if err != nil {
		return nil, err
	}

	targetSub.UpdatedAt = time.Now()
	if userInfo != nil {
		targetSub.UserInfo = userInfo
	}
	_ = m.saveMeta()

	return targetSub, nil
}

// DeleteSubscription removes subscription record and optionally deletes local file.
func (m *Manager) DeleteSubscription(subID string, deleteFile bool) error {
	idx := -1
	var fileName string
	for i, sub := range m.subscriptions {
		if sub.ID == subID || sub.FileName == subID {
			idx = i
			fileName = sub.FileName
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("subscription not found: %s", subID)
	}

	m.subscriptions = append(m.subscriptions[:idx], m.subscriptions[idx+1:]...)
	_ = m.saveMeta()

	if deleteFile && fileName != "" {
		_ = os.Remove(filepath.Join(m.configDir, fileName))
	}
	return nil
}

// downloadAndValidate downloads config from URL, verifies YAML, and writes atomically.
func (m *Manager) downloadAndValidate(ctx context.Context, rawURL, targetPath string) (*SubUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	// Use Clash.Meta / mihomo UA so subscription providers return standard mihomo YAML
	req.Header.Set("User-Agent", "Clash.Meta; mihomo/1.18.0; mihomo-tui")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with HTTP status %d", resp.StatusCode)
	}

	// Parse Userinfo if header present
	var userInfo *SubUserInfo
	if rawUI := resp.Header.Get("subscription-userinfo"); rawUI != "" {
		userInfo = ParseUserinfo(rawUI)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Validate basic YAML structure to prevent saving 404 HTML pages
	var testMap map[string]any
	if err := yaml.Unmarshal(content, &testMap); err != nil {
		return nil, fmt.Errorf("downloaded content is not valid YAML: %w", err)
	}

	// Write atomically via temporary file
	tmpPath := targetPath + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return nil, err
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, err
	}

	return userInfo, nil
}
