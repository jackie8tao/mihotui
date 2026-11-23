package pages

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// FieldKind defines the input and serialization behavior of an override.yaml field.
type FieldKind int

const (
	// KindBool represents a boolean toggle (true / false).
	KindBool FieldKind = iota
	// KindSelect represents a fixed set of string options.
	KindSelect
	// KindInt represents an integer input field.
	KindInt
	// KindString represents a free-form single-line string input field.
	KindString
	// KindStringList represents a comma-separated list of strings ([]string in YAML).
	KindStringList
)

// FieldSpec declaratively describes a single overrideable Mihomo configuration field.
type FieldSpec struct {
	Path        []string
	Label       string
	Description string
	Kind        FieldKind
	Options     []string
	Placeholder string
	Validate    func(raw string) (any, error)
}

// SectionSpec groups related overrideable fields under a top-level configuration category.
type SectionSpec struct {
	Title  string
	Fields []FieldSpec
}

// YAMLPath returns the dotted YAML key path for display (e.g. "tun.enable").
func (f FieldSpec) YAMLPath() string {
	return strings.Join(f.Path, ".")
}

// KindHint returns a short human-readable type label for the UI.
func (f FieldSpec) KindHint() string {
	switch f.Kind {
	case KindBool:
		return "bool (Enter to toggle)"
	case KindSelect:
		return fmt.Sprintf("enum [%s] (Enter to cycle)", strings.Join(f.Options, " / "))
	case KindInt:
		return "int (Enter to edit)"
	case KindString:
		return "string (Enter to edit)"
	case KindStringList:
		return "list (Enter to edit, comma-separated)"
	default:
		return ""
	}
}

// FormatDisplay formats a typed YAML value for display in the fields list.
func (f FieldSpec) FormatDisplay(val any) string {
	switch f.Kind {
	case KindBool:
		if b, ok := val.(bool); ok && b {
			return "[ON]"
		}
		return "[OFF]"
	case KindSelect:
		return fmt.Sprintf("< %v >", val)
	case KindStringList:
		items := toStringSlice(val)
		if len(items) == 0 {
			return "[]"
		}
		return "[" + strings.Join(items, ", ") + "]"
	default:
		return fmt.Sprintf("%v", val)
	}
}

// FormatEditValue formats a typed YAML value as the initial text in the edit modal.
func (f FieldSpec) FormatEditValue(val any) string {
	if val == nil {
		return ""
	}
	if f.Kind == KindStringList {
		return strings.Join(toStringSlice(val), ", ")
	}
	return fmt.Sprintf("%v", val)
}

// ParseInput parses and validates raw text entered by the user into its typed YAML value.
func (f FieldSpec) ParseInput(raw string) (any, error) {
	if f.Validate != nil {
		return f.Validate(raw)
	}

	trimmed := strings.TrimSpace(raw)
	switch f.Kind {
	case KindInt:
		n, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q", trimmed)
		}
		return n, nil

	case KindString:
		if trimmed == "" {
			return nil, errors.New("value cannot be empty (press 'x' to reset to inherit)")
		}
		return trimmed, nil

	case KindStringList:
		var items []string
		for _, part := range strings.Split(trimmed, ",") {
			if s := strings.TrimSpace(part); s != "" {
				items = append(items, s)
			}
		}
		if len(items) == 0 {
			return nil, errors.New("list cannot be empty (press 'x' to reset to inherit)")
		}
		return items, nil

	default:
		return trimmed, nil
	}
}

func toStringSlice(val any) []string {
	switch list := val.(type) {
	case []string:
		return list
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			out = append(out, fmt.Sprintf("%v", item))
		}
		return out
	default:
		return nil
	}
}

// defaultConfigSections defines the registered override.yaml configuration sections.
// Adding new modules (e.g. DNS, General, Sniffer) only requires appending a SectionSpec here.
func defaultConfigSections() []SectionSpec {
	return []SectionSpec{
		{
			Title: "TUN Mode",
			Fields: []FieldSpec{
				{
					Path:        []string{"tun", "enable"},
					Label:       "TUN Enable",
					Description: "Enable or disable Mihomo virtual TUN interface to capture system-wide traffic.",
					Kind:        KindBool,
				},
				{
					Path:        []string{"tun", "stack"},
					Label:       "Network Stack",
					Description: "TCP/IP stack implementation: mixed (recommended), system (native OS), or gvisor (userspace).",
					Kind:        KindSelect,
					Options:     []string{"mixed", "system", "gvisor"},
				},
				{
					Path:        []string{"tun", "device"},
					Label:       "Device Name",
					Description: "Custom virtual network interface name (e.g. mihomo or utun).",
					Kind:        KindString,
					Placeholder: "e.g. mihomo",
				},
				{
					Path:        []string{"tun", "auto-route"},
					Label:       "Auto Route",
					Description: "Automatically configure system routing table to direct traffic into the TUN interface.",
					Kind:        KindBool,
				},
				{
					Path:        []string{"tun", "auto-detect-interface"},
					Label:       "Auto Detect Iface",
					Description: "Automatically bind outbound traffic to the default physical network interface.",
					Kind:        KindBool,
				},
				{
					Path:        []string{"tun", "dns-hijack"},
					Label:       "DNS Hijack",
					Description: "Intercept outbound DNS queries through Mihomo DNS (comma-separated, e.g. any:53, tcp://any:53).",
					Kind:        KindStringList,
					Placeholder: "any:53, tcp://any:53",
				},
			},
		},
	}
}
