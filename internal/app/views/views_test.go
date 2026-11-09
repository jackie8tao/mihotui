package views

import (
	"strings"
	"testing"

	"mihomo-tui/pkg/mihomo"
)

func TestRenderOverview(t *testing.T) {
	props := OverviewProps{
		Width: 80,
		Traffic: mihomo.Traffic{
			Up:   1024 * 500,
			Down: 1024 * 1024 * 2,
		},
		TotalUp:   1024 * 1024 * 100,
		TotalDown: 1024 * 1024 * 500,
		Config: mihomo.Config{
			Mode:      "rule",
			MixedPort: 7890,
			LogLevel:  "info",
		},
		Version: mihomo.Version{
			Version: "1.18.0",
			Meta:    true,
		},
		Memory: mihomo.Memory{
			Inuse: 1024 * 1024 * 32,
		},
	}

	output := RenderOverview(props)
	if !strings.Contains(output, "Real-time Bandwidth") {
		t.Errorf("expected overview output to contain 'Real-time Bandwidth'")
	}
	if !strings.Contains(output, "RULE") {
		t.Errorf("expected overview output to contain mode 'RULE'")
	}
	if !strings.Contains(output, "1.18.0") {
		t.Errorf("expected overview output to contain version '1.18.0'")
	}
}

func TestRenderProxies_Empty(t *testing.T) {
	props := ProxiesProps{
		Width:       80,
		Height:      24,
		FocusPane:   PaneGroups,
		Groups:      nil,
		Proxies:     nil,
		GroupCursor: 0,
		NodeCursor:  0,
	}

	output := RenderProxies(props)
	if !strings.Contains(output, "No proxy groups found") {
		t.Errorf("expected output to contain 'No proxy groups found'")
	}
}

func TestRenderConfigs(t *testing.T) {
	props := ConfigsProps{
		Width:         80,
		Height:        24,
		ConfigDir:     "/etc/mihomo",
		Profiles:      nil,
		ProfileCursor: 0,
	}

	output := RenderConfigs(props)
	if !strings.Contains(output, "Local Profiles") {
		t.Errorf("expected output to contain 'Local Profiles'")
	}
	if !strings.Contains(output, "No config profiles found") {
		t.Errorf("expected output to contain 'No config profiles found'")
	}
}

func TestRenderSubscriptions(t *testing.T) {
	props := SubscriptionsProps{
		Width:         80,
		Height:        24,
		Subscriptions: nil,
		SubCursor:     0,
		ModalActive:   ModalNone,
	}

	output := RenderSubscriptions(props)
	if !strings.Contains(output, "Subscriptions") {
		t.Errorf("expected output to contain 'Subscriptions'")
	}
	if !strings.Contains(output, "No subscriptions added yet") {
		t.Errorf("expected output to contain 'No subscriptions added yet'")
	}
}

