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

type (
	profilesLoadedMsg struct {
		profiles []profile.Profile
		err      error
	}
	profileAppliedMsg struct {
		profileName string
		err         error
	}
)

// ConfigsModel encapsulates state, interactions, and rendering for the Configs tab.
type ConfigsModel struct {
	client      mihomo.Client
	profileMgr  *profile.Manager
	profiles    []profile.Profile
	selectedIdx int
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
	}
}

// Title returns the tab header title for the Configs page.
func (m *ConfigsModel) Title() string {
	return "3. Configs"
}

// Help returns the keybinding hints displayed in the footer for the Configs page.
func (m *ConfigsModel) Help() string {
	return "[↑/↓] Move | [Enter] Apply Profile"
}

// CapturingInput reports whether the Configs page is currently capturing raw keyboard input.
func (m *ConfigsModel) CapturingInput() bool {
	return false
}

// Init triggers the initial scan of local configuration profiles.
func (m *ConfigsModel) Init() tea.Cmd {
	return m.fetchProfilesCmd()
}

// Update handles profile list loading, hot-reload results, subscription change events, and keyboard navigation.
func (m *ConfigsModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case profilesLoadedMsg:
		if msg.err != nil {
			return events.StatusCmd(fmt.Sprintf("Load profiles error: %v", msg.err), true)
		}
		m.profiles = msg.profiles
		if len(m.profiles) == 0 {
			m.selectedIdx = 0
		} else if m.selectedIdx >= len(m.profiles) {
			m.selectedIdx = len(m.profiles) - 1
		}
		return nil

	case profileAppliedMsg:
		if msg.err != nil {
			return events.StatusCmd(fmt.Sprintf("Hot-reload profile failed: %v", msg.err), true)
		}
		return tea.Batch(
			events.StatusCmd(fmt.Sprintf("Applied profile [%s] & hot-reloaded successfully!", msg.profileName), false),
			m.fetchProfilesCmd(),
			events.ProfileAppliedCmd(),
		)

	case events.SubChangedMsg:
		return m.fetchProfilesCmd()

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return nil
}

// View renders the Configs page within the given terminal dimensions.
func (m *ConfigsModel) View(width, height int) string {
	return views.RenderConfigs(views.ConfigsProps{
		Width:       width,
		Height:      height,
		ConfigDir:   m.profileMgr.ConfigDir(),
		Profiles:    m.profiles,
		SelectedIdx: m.selectedIdx,
	})
}

func (m *ConfigsModel) currentProfile() *profile.Profile {
	if len(m.profiles) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.profiles) {
		return nil
	}
	return &m.profiles[m.selectedIdx]
}

func (m *ConfigsModel) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, keymaps.Up):
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
	case key.Matches(msg, keymaps.Down):
		if m.selectedIdx < len(m.profiles)-1 {
			m.selectedIdx++
		}
	case key.Matches(msg, keymaps.Enter):
		p := m.currentProfile()
		if p == nil {
			return nil
		}
		return tea.Batch(
			events.StatusCmd(fmt.Sprintf("Applying profile [%s] to Mihomo...", p.Name), false),
			m.applyProfileCmd(p),
		)
	}

	return nil
}

func (m *ConfigsModel) fetchProfilesCmd() tea.Cmd {
	return func() tea.Msg {
		profiles, err := m.profileMgr.ListProfiles()
		return profilesLoadedMsg{profiles: profiles, err: err}
	}
}

func (m *ConfigsModel) applyProfileCmd(p *profile.Profile) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		err := m.client.ReloadConfig(ctx, p.FullPath, true)
		if err == nil {
			_ = m.profileMgr.SetActiveProfile(p.FileName)
		}
		return profileAppliedMsg{profileName: p.Name, err: err}
	}
}
