package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jackie8tao/mihotui/internal/config"
)

func (m *AppModel) currentSubscription() *config.Subscription {
	if len(m.subscriptions) == 0 || m.subsState.Cursor < 0 || m.subsState.Cursor >= len(m.subscriptions) {
		return nil
	}
	return &m.subscriptions[m.subsState.Cursor]
}

func (m *AppModel) updateSubCmd(subID, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		_, err := m.profileMgr.UpdateSubscription(ctx, subID)
		return subUpdatedMsg{name: name, err: err}
	}
}

func (m *AppModel) updateAllSubsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		subs := m.profileMgr.ListSubscriptions()
		var errs []string
		for _, s := range subs {
			if _, err := m.profileMgr.UpdateSubscription(ctx, s.ID); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", s.Name, err))
			}
		}
		if len(errs) > 0 {
			return subUpdatedMsg{name: "All", err: fmt.Errorf("%s", strings.Join(errs, "; "))}
		}
		return subUpdatedMsg{name: "All", err: nil}
	}
}

func (m *AppModel) addSubCmd(name, url string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		_, err := m.profileMgr.AddSubscription(ctx, name, url)
		return subAddedMsg{name: name, err: err}
	}
}

func (m *AppModel) deleteSubCmd(subID, name string) tea.Cmd {
	return func() tea.Msg {
		err := m.profileMgr.DeleteSubscription(subID, true)
		return subDeletedMsg{name: name, err: err}
	}
}

func (m *AppModel) updateAddSubModal(msg tea.KeyMsg) (tea.Cmd, bool) {
	if m.subsState.ModalActive != ModalAddSub {
		return nil, false
	}

	switch msg.String() {
	case "esc":
		m.subsState.ModalActive = ModalNone
		return nil, true

	case "tab", "down":
		if m.subsState.InputFocus == 0 {
			m.subsState.InputFocus = 1
			m.subsState.NameInput.Blur()
			m.subsState.URLInput.Focus()
		} else {
			m.subsState.InputFocus = 0
			m.subsState.URLInput.Blur()
			m.subsState.NameInput.Focus()
		}
		return nil, true

	case "up":
		if m.subsState.InputFocus == 1 {
			m.subsState.InputFocus = 0
			m.subsState.URLInput.Blur()
			m.subsState.NameInput.Focus()
		}
		return nil, true

	case "enter":
		name := strings.TrimSpace(m.subsState.NameInput.Value())
		subURL := strings.TrimSpace(m.subsState.URLInput.Value())
		if name == "" || subURL == "" {
			m.statusMsg = "Name and URL cannot be empty"
			m.isError = true
			return nil, true
		}
		m.statusMsg = fmt.Sprintf("Downloading subscription [%s]...", name)
		return m.addSubCmd(name, subURL), true

	default:
		var cmd tea.Cmd
		if m.subsState.InputFocus == 0 {
			m.subsState.NameInput, cmd = m.subsState.NameInput.Update(msg)
		} else {
			m.subsState.URLInput, cmd = m.subsState.URLInput.Update(msg)
		}
		return cmd, true
	}
}

func (m *AppModel) updateSubscriptionsTab(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.subsState.Cursor > 0 {
			m.subsState.Cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.subsState.Cursor < len(m.subscriptions)-1 {
			m.subsState.Cursor++
		}
	case key.Matches(msg, m.keys.UpdateSub), key.Matches(msg, m.keys.Enter):
		sub := m.currentSubscription()
		if sub != nil {
			m.statusMsg = fmt.Sprintf("Updating subscription [%s]...", sub.Name)
			return m.updateSubCmd(sub.ID, sub.Name)
		}
	case key.Matches(msg, m.keys.UpdateAll):
		m.statusMsg = "Updating all subscriptions..."
		return m.updateAllSubsCmd()
	case key.Matches(msg, m.keys.AddSub):
		m.subsState.ModalActive = ModalAddSub
		m.subsState.InputFocus = 0
		m.subsState.NameInput.Focus()
		m.subsState.URLInput.Blur()
		return nil
	case key.Matches(msg, m.keys.DeleteSub):
		sub := m.currentSubscription()
		if sub != nil {
			m.statusMsg = fmt.Sprintf("Deleting subscription [%s]...", sub.Name)
			return m.deleteSubCmd(sub.ID, sub.Name)
		}
	}

	return nil
}
