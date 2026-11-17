package pages

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jackie8tao/mihotui/internal/app/views"
	"github.com/jackie8tao/mihotui/internal/config"
	"github.com/jackie8tao/mihotui/internal/profile"
)

type (
	subUpdatedMsg struct {
		name string
		err  error
	}
	subAddedMsg struct {
		name string
		err  error
	}
	subDeletedMsg struct {
		name string
		err  error
	}
)

// SubscriptionsModel encapsulates state, interactions, and rendering for the Subscriptions tab.
type SubscriptionsModel struct {
	profileMgr    *profile.Manager
	keys          keyMap
	subscriptions []config.Subscription
	cursor        int

	modalActive views.ModalType
	nameInput   textinput.Model
	urlInput    textinput.Model
	inputFocus  int // 0: Name, 1: URL
}

// NewSubscriptionsModel initializes a self-contained SubscriptionsModel.
func NewSubscriptionsModel(mgr *profile.Manager) *SubscriptionsModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "e.g. My-Airport"
	nameInput.Focus()
	nameInput.Width = 46

	urlInput := textinput.New()
	urlInput.Placeholder = "https://sub.example.com/api/v1/..."
	urlInput.Width = 46

	m := &SubscriptionsModel{
		profileMgr:  mgr,
		keys:        defaultKeyMap(),
		modalActive: views.ModalNone,
		nameInput:   nameInput,
		urlInput:    urlInput,
		inputFocus:  0,
	}
	m.Reload()
	return m
}

func (m *SubscriptionsModel) Title() string {
	return "4. Subscriptions"
}

func (m *SubscriptionsModel) Help() string {
	return "[↑/↓] Move | [Enter/u] Update | [U] Update All | [a] Add Sub | [d] Delete"
}

func (m *SubscriptionsModel) CapturingInput() bool {
	return m.modalActive == views.ModalAddSub
}

func (m *SubscriptionsModel) Init() tea.Cmd {
	m.Reload()
	return nil
}

func (m *SubscriptionsModel) Reload() {
	if m.profileMgr == nil {
		m.subscriptions = nil
		m.cursor = 0
		return
	}
	m.subscriptions = m.profileMgr.ListSubscriptions()
	if len(m.subscriptions) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor >= len(m.subscriptions) {
		m.cursor = len(m.subscriptions) - 1
	}
}

func (m *SubscriptionsModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case subUpdatedMsg:
		if msg.err != nil {
			return statusCmd(fmt.Sprintf("Update sub failed: %v", msg.err), true)
		}
		m.Reload()
		return tea.Batch(
			statusCmd(fmt.Sprintf("Subscription [%s] updated successfully!", msg.name), false),
			subscriptionsChangedCmd(),
		)

	case subAddedMsg:
		if msg.err != nil {
			return statusCmd(fmt.Sprintf("Add subscription failed: %v", msg.err), true)
		}
		m.modalActive = views.ModalNone
		m.nameInput.SetValue("")
		m.urlInput.SetValue("")
		m.Reload()
		return tea.Batch(
			statusCmd(fmt.Sprintf("Subscription [%s] added and downloaded!", msg.name), false),
			subscriptionsChangedCmd(),
		)

	case subDeletedMsg:
		if msg.err != nil {
			return statusCmd(fmt.Sprintf("Delete sub failed: %v", msg.err), true)
		}
		m.Reload()
		return tea.Batch(
			statusCmd(fmt.Sprintf("Subscription [%s] deleted!", msg.name), false),
			subscriptionsChangedCmd(),
		)

	case tea.KeyMsg:
		if m.modalActive == views.ModalAddSub {
			return m.updateAddSubModal(msg)
		}
		return m.handleKeyMsg(msg)
	}

	return nil
}

func (m *SubscriptionsModel) View(width, height int) string {
	return views.RenderSubscriptions(views.SubscriptionsProps{
		Width:            width,
		Height:           height,
		Subscriptions:    m.subscriptions,
		SubCursor:        m.cursor,
		ModalActive:      m.modalActive,
		SubNameInputView: m.nameInput.View(),
		SubURLInputView:  m.urlInput.View(),
	})
}

func (m *SubscriptionsModel) currentSubscription() *config.Subscription {
	if len(m.subscriptions) == 0 || m.cursor < 0 || m.cursor >= len(m.subscriptions) {
		return nil
	}
	return &m.subscriptions[m.cursor]
}

func (m *SubscriptionsModel) updateAddSubModal(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.modalActive = views.ModalNone
		return nil

	case "tab", "down":
		if m.inputFocus == 0 {
			m.inputFocus = 1
			m.nameInput.Blur()
			m.urlInput.Focus()
		} else {
			m.inputFocus = 0
			m.urlInput.Blur()
			m.nameInput.Focus()
		}
		return nil

	case "up":
		if m.inputFocus == 1 {
			m.inputFocus = 0
			m.urlInput.Blur()
			m.nameInput.Focus()
		}
		return nil

	case "enter":
		name := strings.TrimSpace(m.nameInput.Value())
		subURL := strings.TrimSpace(m.urlInput.Value())
		if name == "" || subURL == "" {
			return statusCmd("Name and URL cannot be empty", true)
		}
		return tea.Batch(
			statusCmd(fmt.Sprintf("Downloading subscription [%s]...", name), false),
			m.addSubCmd(name, subURL),
		)

	default:
		var cmd tea.Cmd
		if m.inputFocus == 0 {
			m.nameInput, cmd = m.nameInput.Update(msg)
		} else {
			m.urlInput, cmd = m.urlInput.Update(msg)
		}
		return cmd
	}
}

func (m *SubscriptionsModel) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.subscriptions)-1 {
			m.cursor++
		}
	case key.Matches(msg, m.keys.UpdateSub), key.Matches(msg, m.keys.Enter):
		sub := m.currentSubscription()
		if sub != nil {
			return tea.Batch(
				statusCmd(fmt.Sprintf("Updating subscription [%s]...", sub.Name), false),
				m.updateSubCmd(sub.Name),
			)
		}
	case key.Matches(msg, m.keys.UpdateAll):
		return tea.Batch(
			statusCmd("Updating all subscriptions...", false),
			m.updateAllSubsCmd(),
		)
	case key.Matches(msg, m.keys.AddSub):
		m.modalActive = views.ModalAddSub
		m.inputFocus = 0
		m.nameInput.Focus()
		m.urlInput.Blur()
	case key.Matches(msg, m.keys.DeleteSub):
		sub := m.currentSubscription()
		if sub != nil {
			return tea.Batch(
				statusCmd(fmt.Sprintf("Deleting subscription [%s]...", sub.Name), false),
				m.deleteSubCmd(sub.Name),
			)
		}
	}

	return nil
}

func (m *SubscriptionsModel) updateSubCmd(name string) tea.Cmd {
	return func() tea.Msg {
		if m.profileMgr == nil {
			return subUpdatedMsg{name: name, err: errors.New("profile manager unavailable")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		_, err := m.profileMgr.UpdateSubscription(ctx, name)
		return subUpdatedMsg{name: name, err: err}
	}
}

func (m *SubscriptionsModel) updateAllSubsCmd() tea.Cmd {
	return func() tea.Msg {
		if m.profileMgr == nil {
			return subUpdatedMsg{name: "All", err: errors.New("profile manager unavailable")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		subs := m.profileMgr.ListSubscriptions()
		var errs []string
		for _, s := range subs {
			if _, err := m.profileMgr.UpdateSubscription(ctx, s.Name); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", s.Name, err))
			}
		}
		if len(errs) > 0 {
			return subUpdatedMsg{name: "All", err: fmt.Errorf("%s", strings.Join(errs, "; "))}
		}
		return subUpdatedMsg{name: "All", err: nil}
	}
}

func (m *SubscriptionsModel) addSubCmd(name, url string) tea.Cmd {
	return func() tea.Msg {
		if m.profileMgr == nil {
			return subAddedMsg{name: name, err: errors.New("profile manager unavailable")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		_, err := m.profileMgr.AddSubscription(ctx, name, url)
		return subAddedMsg{name: name, err: err}
	}
}

func (m *SubscriptionsModel) deleteSubCmd(name string) tea.Cmd {
	return func() tea.Msg {
		if m.profileMgr == nil {
			return subDeletedMsg{name: name, err: errors.New("profile manager unavailable")}
		}
		err := m.profileMgr.DeleteSubscription(name, true)
		return subDeletedMsg{name: name, err: err}
	}
}

func subscriptionsChangedCmd() tea.Cmd {
	return func() tea.Msg {
		return SubscriptionsChangedMsg{}
	}
}
