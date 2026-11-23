package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// OverrideDoc wraps the parsed override.yaml document tree and provides path-based
// reads and mutations while preserving any hand-edited user keys untouched.
type OverrideDoc struct {
	data map[string]any
}

// NewOverrideDoc initializes an empty OverrideDoc.
func NewOverrideDoc() *OverrideDoc {
	return &OverrideDoc{
		data: make(map[string]any),
	}
}

// Get retrieves the value at the given nested YAML path.
// It returns (nil, false) when the path is not set (inherited from subscription).
func (d *OverrideDoc) Get(path ...string) (any, bool) {
	if d == nil || d.data == nil || len(path) == 0 {
		return nil, false
	}

	cur := d.data
	for i := 0; i < len(path)-1; i++ {
		next, ok := cur[path[i]].(map[string]any)
		if !ok || next == nil {
			return nil, false
		}
		cur = next
	}

	val, exists := cur[path[len(path)-1]]
	return val, exists
}

// Set writes val at the given nested YAML path, creating intermediate maps as needed.
func (d *OverrideDoc) Set(val any, path ...string) {
	if d == nil || len(path) == 0 {
		return
	}
	if d.data == nil {
		d.data = make(map[string]any)
	}

	cur := d.data
	for i := 0; i < len(path)-1; i++ {
		next, ok := cur[path[i]].(map[string]any)
		if !ok || next == nil {
			next = make(map[string]any)
			cur[path[i]] = next
		}
		cur = next
	}
	cur[path[len(path)-1]] = val
}

// Delete removes the key at the given nested YAML path and prunes any parent maps
// that become empty, returning true if a key was removed.
func (d *OverrideDoc) Delete(path ...string) bool {
	if d == nil || d.data == nil || len(path) == 0 {
		return false
	}
	return deleteNestedKey(d.data, path)
}

func deleteNestedKey(cur map[string]any, path []string) bool {
	key := path[0]
	if len(path) == 1 {
		if _, exists := cur[key]; !exists {
			return false
		}
		delete(cur, key)
		return true
	}

	child, ok := cur[key].(map[string]any)
	if !ok || child == nil {
		return false
	}
	deleted := deleteNestedKey(child, path[1:])
	if deleted && len(child) == 0 {
		delete(cur, key)
	}
	return deleted
}

// Clone returns a deep copy of the document safe for concurrent serialization.
func (d *OverrideDoc) Clone() *OverrideDoc {
	if d == nil || d.data == nil {
		return NewOverrideDoc()
	}
	return &OverrideDoc{
		data: deepCopyMap(d.data),
	}
}

// MarshalSection renders the YAML snippet for a specific section key.
// If sectionKey is non-empty (e.g. "tun", "dns"), only that top-level block is rendered.
// If sectionKey is empty, all top-level non-map keys are rendered.
func (d *OverrideDoc) MarshalSection(sectionKey string) string {
	if d == nil || len(d.data) == 0 {
		return ""
	}

	var subset map[string]any
	if sectionKey != "" {
		val, exists := d.data[sectionKey]
		if !exists {
			return ""
		}
		subset = map[string]any{sectionKey: val}
	} else {
		subset = make(map[string]any)
		for k, v := range d.data {
			if _, isMap := v.(map[string]any); !isMap {
				subset[k] = v
			}
		}
		if len(subset) == 0 {
			return ""
		}
	}

	raw, err := yaml.Marshal(subset)
	if err != nil {
		return ""
	}
	return string(raw)
}

func deepCopyMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = deepCopyValue(v)
	}
	return dst
}

func deepCopyValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return deepCopyMap(val)
	case []any:
		cp := make([]any, len(val))
		for i, item := range val {
			cp[i] = deepCopyValue(item)
		}
		return cp
	case []string:
		cp := make([]string, len(val))
		copy(cp, val)
		return cp
	default:
		return v
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

// LoadOverride reads override.yaml into an OverrideDoc, returning an empty document if the file does not exist yet.
func (m *Manager) LoadOverride() (*OverrideDoc, error) {
	root, err := readYAMLMap(m.OverridePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewOverrideDoc(), nil
		}
		return nil, fmt.Errorf("profile: %w", err)
	}
	return &OverrideDoc{data: root}, nil
}

// SaveOverride atomically writes the OverrideDoc to override.yaml.
func (m *Manager) SaveOverride(doc *OverrideDoc) error {
	data := make(map[string]any)
	if doc != nil && doc.data != nil {
		data = doc.data
	}
	if err := writeYAMLAtomically(m.OverridePath(), data); err != nil {
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
