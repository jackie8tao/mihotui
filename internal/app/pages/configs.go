package pages

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jackie8tao/mihotui/internal/app/events"
	"github.com/jackie8tao/mihotui/internal/app/keymaps"
	"github.com/jackie8tao/mihotui/internal/app/views"
	"github.com/jackie8tao/mihotui/internal/profile"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

const tunOptionCount = 5

type (
	overrideLoadedMsg struct {
		tun profile.TunOverride
		err error
	}
	overrideAppliedMsg struct {
		reloaded  bool
		activeSub string
		err       error
	}
)

// ConfigsModel manages user system configuration overrides stored in override.yaml.
type ConfigsModel struct {
	client      mihomo.Client
	profileMgr  *profile.Manager
	tun         profile.TunOverride
	selectedIdx int
	dirty       bool
}

// NewConfigsModel initializes a self-contained ConfigsModel.
func NewConfigsModel(cli mihomo.Client, mgr *profile.Manager) *ConfigsModel {
	if cli == nil {
		panic("pages: nil mihomo client")
	}
	if mgr == nil {
		panic("pages: nil profile manager")
	}
	return &ConfigsModel{
		client:     cli,
		profileMgr: mgr,
		tun:        profile.DefaultTunOverride(),
	}
}

// Title returns the tab header title for the Configs page.
func (m *ConfigsModel) Title() string {
	return "3. Configs"
}

// Help returns the keybinding hints displayed in the footer for the Configs page.
func (m *ConfigsModel) Help() string {
	return "[↑/↓] Move | [←/→] Change | [Enter] Save & Apply"
}

// CapturingInput reports whether the Configs page is currently capturing raw keyboard input.
func (m *ConfigsModel) CapturingInput() bool {
	return false
}

// Init loads the current override.yaml settings from disk.
func (m *ConfigsModel) Init() tea.Cmd {
	return m.loadOverrideCmd()
}

// Update handles override loading, save & merge results, and keyboard navigation.
func (m *ConfigsModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case overrideLoadedMsg:
		if msg.err != nil {
			return events.StatusCmd(fmt.Sprintf("Load override.yaml failed: %v", msg.err), true)
		}
		m.tun = msg.tun
		m.dirty = false
		return nil

	case overrideAppliedMsg:
		if msg.err != nil {
			return events.StatusCmd(fmt.Sprintf("Apply override failed: %v", msg.err), true)
		}
		m.dirty = false
		if !msg.reloaded {
			return events.StatusCmd("Saved override.yaml (no active subscription to reload)", false)
		}
		return tea.Batch(
			events.StatusCmd(fmt.Sprintf("Saved override.yaml & hot-reloaded with [%s]!", msg.activeSub), false),
			events.ProfileAppliedCmd(),
		)

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return nil
}

// View renders the Configs page within the given terminal dimensions.
func (m *ConfigsModel) View(width, height int) string {
	return views.RenderConfigs(views.ConfigsProps{
		Width:            width,
		Height:           height,
		OverridePath:     m.profileMgr.OverridePath(),
		MihomoConfigPath: m.profileMgr.MihomoConfigPath(),
		ActiveSubName:    m.profileMgr.ActiveProfile(),
		Tun:              m.tun,
		SelectedIdx:      m.selectedIdx,
		Dirty:            m.dirty,
	})
}

func (m *ConfigsModel) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, keymaps.Up):
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
	case key.Matches(msg, keymaps.Down):
		if m.selectedIdx < tunOptionCount-1 {
			m.selectedIdx++
		}
	case key.Matches(msg, keymaps.Left):
		m.adjustSelected(-1)
	case key.Matches(msg, keymaps.Right):
		m.adjustSelected(1)
	case key.Matches(msg, keymaps.Enter):
		return tea.Batch(
			events.StatusCmd("Saving override.yaml and merging Mihomo config...", false),
			m.saveAndApplyCmd(m.tun),
		)
	}

	return nil
}

func (m *ConfigsModel) adjustSelected(delta int) {
	switch m.selectedIdx {
	case 0:
		m.tun.Enable = !m.tun.Enable
	case 1:
		m.tun.Stack = cycleStack(m.tun.Stack, delta)
	case 2:
		m.tun.AutoRoute = !m.tun.AutoRoute
	case 3:
		m.tun.AutoDetectInterface = !m.tun.AutoDetectInterface
	case 4:
		m.tun.DNSHijack = !m.tun.DNSHijack
	default:
		return
	}
	m.dirty = true
}

func cycleStack(current string, delta int) string {
	stacks := profile.ValidTunStacks
	idx := 0
	for i, s := range stacks {
		if s == current {
			idx = i
			break
		}
	}
	next := (idx + delta) % len(stacks)
	if next < 0 {
		next += len(stacks)
	}
	return stacks[next]
}

func (m *ConfigsModel) loadOverrideCmd() tea.Cmd {
	return func() tea.Msg {
		tun, err := m.profileMgr.LoadTunOverride()
		return overrideLoadedMsg{tun: tun, err: err}
	}
}

func (m *ConfigsModel) saveAndApplyCmd(tun profile.TunOverride) tea.Cmd {
	return func() tea.Msg {
		if err := m.profileMgr.SaveTunOverride(tun); err != nil {
			return overrideAppliedMsg{err: err}
		}

		activeSub := m.profileMgr.ActiveProfile()
		if activeSub == "" {
			return overrideAppliedMsg{reloaded: false}
		}

		mihomoCfgPath, err := m.profileMgr.BuildMihomoConfig(activeSub)
		if err != nil {
			return overrideAppliedMsg{activeSub: activeSub, err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := m.client.ReloadConfig(ctx, mihomoCfgPath, true); err != nil {
			return overrideAppliedMsg{activeSub: activeSub, err: err}
		}

		return overrideAppliedMsg{reloaded: true, activeSub: activeSub}
	}
}
