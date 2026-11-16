package app

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jackie8tao/mihotui/internal/profile"
)

func (m *AppModel) currentProfile() *profile.Profile {
	if len(m.profiles) == 0 || m.configsState.Cursor < 0 || m.configsState.Cursor >= len(m.profiles) {
		return nil
	}
	return &m.profiles[m.configsState.Cursor]
}

func (m *AppModel) fetchProfilesCmd() tea.Cmd {
	return func() tea.Msg {
		if m.profileMgr == nil {
			return nil
		}
		profiles, err := m.profileMgr.ListProfiles()
		return profilesLoadedMsg{profiles: profiles, err: err}
	}
}

func (m *AppModel) applyProfileCmd(p *profile.Profile) tea.Cmd {
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

func (m *AppModel) updateConfigsTab(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.configsState.Cursor > 0 {
			m.configsState.Cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.configsState.Cursor < len(m.profiles)-1 {
			m.configsState.Cursor++
		}
	case key.Matches(msg, m.keys.Enter):
		p := m.currentProfile()
		if p != nil {
			m.statusMsg = fmt.Sprintf("Applying profile [%s] to Mihomo...", p.Name)
			return m.applyProfileCmd(p)
		}
	}

	return nil
}
