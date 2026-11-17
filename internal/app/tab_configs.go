package app

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

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
	profileAppliedEventMsg struct{}
)

// ConfigsModel encapsulates state, interactions, and rendering for the Configs tab.
type ConfigsModel struct {
	client     mihomo.Client
	profileMgr *profile.Manager
	keys       KeyMap
	profiles   []profile.Profile
	cursor     int
}

// NewConfigsModel initializes a self-contained ConfigsModel.
func NewConfigsModel(cli mihomo.Client, mgr *profile.Manager, keys KeyMap) *ConfigsModel {
	return &ConfigsModel{
		client:     cli,
		profileMgr: mgr,
		keys:       keys,
	}
}

func (m *ConfigsModel) Title() string {
	return "3. Configs"
}

func (m *ConfigsModel) Help() string {
	return "[↑/↓] Move | [Enter] Apply Profile"
}

func (m *ConfigsModel) CapturingInput() bool {
	return false
}

func (m *ConfigsModel) Init() tea.Cmd {
	return m.fetchProfilesCmd()
}

func (m *ConfigsModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case profilesLoadedMsg:
		if msg.err != nil {
			return statusCmd(fmt.Sprintf("Load profiles error: %v", msg.err), true)
		}
		m.profiles = msg.profiles
		if len(m.profiles) == 0 {
			m.cursor = 0
		} else if m.cursor >= len(m.profiles) {
			m.cursor = len(m.profiles) - 1
		}
		return nil

	case profileAppliedMsg:
		if msg.err != nil {
			return statusCmd(fmt.Sprintf("Hot-reload profile failed: %v", msg.err), true)
		}
		return tea.Batch(
			statusCmd(fmt.Sprintf("Applied profile [%s] & hot-reloaded successfully!", msg.profileName), false),
			m.fetchProfilesCmd(),
			profileAppliedEventCmd(),
		)

	case subscriptionsChangedMsg:
		return m.fetchProfilesCmd()

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return nil
}

func (m *ConfigsModel) View(width, height int) string {
	configDir := ""
	if m.profileMgr != nil {
		configDir = m.profileMgr.ConfigDir()
	}
	return views.RenderConfigs(views.ConfigsProps{
		Width:         width,
		Height:        height,
		ConfigDir:     configDir,
		Profiles:      m.profiles,
		ProfileCursor: m.cursor,
	})
}

func (m *ConfigsModel) currentProfile() *profile.Profile {
	if len(m.profiles) == 0 || m.cursor < 0 || m.cursor >= len(m.profiles) {
		return nil
	}
	return &m.profiles[m.cursor]
}

func (m *ConfigsModel) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.profiles)-1 {
			m.cursor++
		}
	case key.Matches(msg, m.keys.Enter):
		p := m.currentProfile()
		if p == nil {
			return nil
		}
		return tea.Batch(
			statusCmd(fmt.Sprintf("Applying profile [%s] to Mihomo...", p.Name), false),
			m.applyProfileCmd(p),
		)
	}

	return nil
}

func (m *ConfigsModel) fetchProfilesCmd() tea.Cmd {
	return func() tea.Msg {
		if m.profileMgr == nil {
			return nil
		}
		profiles, err := m.profileMgr.ListProfiles()
		return profilesLoadedMsg{profiles: profiles, err: err}
	}
}

func (m *ConfigsModel) applyProfileCmd(p *profile.Profile) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		err := m.client.ReloadConfig(ctx, p.FullPath, true)
		if err == nil && m.profileMgr != nil {
			_ = m.profileMgr.SetActiveProfile(p.FileName)
		}
		return profileAppliedMsg{profileName: p.Name, err: err}
	}
}

func profileAppliedEventCmd() tea.Cmd {
	return func() tea.Msg {
		return profileAppliedEventMsg{}
	}
}
