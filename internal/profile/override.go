package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ValidTunStacks lists the supported Mihomo TUN network stack modes.
var ValidTunStacks = []string{"mixed", "system", "gvisor"}

// TunOverride represents user-customized TUN settings stored in override.yaml.
type TunOverride struct {
	Enable              bool
	Stack               string
	AutoRoute           bool
	AutoDetectInterface bool
	DNSHijack           bool
}

// DefaultTunOverride returns sensible default TUN settings when override.yaml does not specify them.
func DefaultTunOverride() TunOverride {
	return TunOverride{
		Enable:              false,
		Stack:               "mixed",
		AutoRoute:           true,
		AutoDetectInterface: true,
		DNSHijack:           true,
	}
}

// OverridePath returns the path to mihotui's override.yaml file.
func (m *Manager) OverridePath() string {
	return filepath.Join(m.appPaths.AppEtcDir(), "override.yaml")
}

// MihomoConfigPath returns the path to the final merged Mihomo config.yaml file.
func (m *Manager) MihomoConfigPath() string {
	return filepath.Join(m.appPaths.MihomoEtcDir(), "config.yaml")
}

// LoadTunOverride reads TUN settings from override.yaml, returning defaults if the file does not exist yet.
func (m *Manager) LoadTunOverride() (TunOverride, error) {
	tun := DefaultTunOverride()

	root, err := readYAMLMap(m.OverridePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return tun, nil
		}
		return tun, fmt.Errorf("profile: %w", err)
	}

	rawTun, ok := root["tun"]
	if !ok {
		return tun, nil
	}
	tunMap, ok := rawTun.(map[string]any)
	if !ok {
		return tun, fmt.Errorf("profile: invalid tun section in %q", m.OverridePath())
	}

	if v, ok := tunMap["enable"].(bool); ok {
		tun.Enable = v
	}
	if v, ok := tunMap["stack"].(string); ok && v != "" {
		tun.Stack = v
	}
	if v, ok := tunMap["auto-route"].(bool); ok {
		tun.AutoRoute = v
	}
	if v, ok := tunMap["auto-detect-interface"].(bool); ok {
		tun.AutoDetectInterface = v
	}
	if v, exists := tunMap["dns-hijack"]; exists {
		switch list := v.(type) {
		case []any:
			tun.DNSHijack = len(list) > 0
		case []string:
			tun.DNSHijack = len(list) > 0
		default:
			tun.DNSHijack = false
		}
	}

	return tun, nil
}

// SaveTunOverride updates the tun section in override.yaml while preserving all other user-defined fields.
func (m *Manager) SaveTunOverride(tun TunOverride) error {
	overridePath := m.OverridePath()

	root, err := readYAMLMap(overridePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("profile: %w", err)
		}
		root = make(map[string]any)
	}

	tunMap, ok := root["tun"].(map[string]any)
	if !ok || tunMap == nil {
		tunMap = make(map[string]any)
	}

	tunMap["enable"] = tun.Enable
	tunMap["stack"] = tun.Stack
	tunMap["auto-route"] = tun.AutoRoute
	tunMap["auto-detect-interface"] = tun.AutoDetectInterface

	if tun.DNSHijack {
		if _, exists := tunMap["dns-hijack"]; !exists {
			tunMap["dns-hijack"] = []string{"any:53"}
		}
	} else {
		delete(tunMap, "dns-hijack")
	}

	root["tun"] = tunMap

	if err := writeYAMLAtomically(overridePath, root); err != nil {
		return fmt.Errorf("profile: %w", err)
	}
	return nil
}

// BuildMihomoConfig merges the subscription YAML with override.yaml and atomically writes the final
// Mihomo config.yaml, returning its absolute path for core hot-reload.
func (m *Manager) BuildMihomoConfig(subName string) (string, error) {
	if subName == "" {
		subName = m.appCfg.ActiveSubName
	}
	if subName == "" {
		return "", errors.New("profile: no active subscription")
	}

	subPath := filepath.Join(m.SubsDir(), m.genSubFilename(subName))
	base, err := readYAMLMap(subPath)
	if err != nil {
		return "", fmt.Errorf("profile: load subscription %q: %w", subName, err)
	}

	override, err := readYAMLMap(m.OverridePath())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("profile: load override: %w", err)
	}
	if override != nil {
		deepMergeMaps(base, override)
	}

	mihomoCfgPath, err := filepath.Abs(m.MihomoConfigPath())
	if err != nil {
		return "", fmt.Errorf("profile: resolve mihomo config path: %w", err)
	}

	if err := writeYAMLAtomically(mihomoCfgPath, base); err != nil {
		return "", fmt.Errorf("profile: write mihomo config: %w", err)
	}
	return mihomoCfgPath, nil
}
