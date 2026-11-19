package pages

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jackie8tao/mihotui/internal/app/events"
	"github.com/jackie8tao/mihotui/internal/app/keymaps"
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
	subscriptions []config.Subscription
	selectedIdx   int

	modalActive views.ModalType
	nameInput   textinput.Model
	urlInput    textinput.Model
	inputFocus  int // 0: Name, 1: URL
}

// NewSubscriptionsModel initializes a self-contained SubscriptionsModel.
func NewSubscriptionsModel(mgr *profile.Manager) *SubscriptionsModel {
	if mgr == nil {
		panic("pages: nil profile manager")
	}

	nameInput := textinput.New()
	nameInput.Placeholder = "e.g. My-Airport"
	nameInput.Focus()
	nameInput.Width = 46

	urlInput := textinput.New()
	urlInput.Placeholder = "https://example.com/api/v1/..."
	urlInput.Width = 46

	return &SubscriptionsModel{
		profileMgr:  mgr,
		modalActive: views.ModalNone,
		nameInput:   nameInput,
		urlInput:    urlInput,
		inputFocus:  0,
	}
}

// Title returns the tab header title for the Subscriptions page.
func (m *SubscriptionsModel) Title() string {
	return "4. Subscriptions"
}

// Help returns the keybinding hints displayed in the footer for the Subscriptions page.
func (m *SubscriptionsModel) Help() string {
	return "[↑/↓] Move | [u] Update | [U] Update All | [a] Add Sub | [d] Delete"
}

// CapturingInput reports whether the add-subscription modal is currently active and capturing keyboard input.
func (m *SubscriptionsModel) CapturingInput() bool {
	return m.modalActive == views.ModalAddSub
}

// Init reloads the initial subscription list and resets modal input state.
func (m *SubscriptionsModel) Init() tea.Cmd {
	m.reload()
	return nil
}

func (m *SubscriptionsModel) reload() {
	m.modalActive = views.ModalNone
	m.inputFocus = 0
	m.nameInput.SetValue("")
	m.nameInput.Focus()
	m.urlInput.SetValue("")
	m.urlInput.Blur()

	m.subscriptions = m.profileMgr.ListSubscriptions()
	if len(m.subscriptions) == 0 {
		m.selectedIdx = 0
		return
	}
	if m.selectedIdx >= len(m.subscriptions) {
		m.selectedIdx = len(m.subscriptions) - 1
	}
}

// Update handles asynchronous subscription results and keyboard interactions for the Subscriptions page.
func (m *SubscriptionsModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case subUpdatedMsg:
		if msg.err != nil {
			return events.StatusCmd(fmt.Sprintf("Update sub failed: %v", msg.err), true)
		}
		m.reload()
		return tea.Batch(
			events.StatusCmd(fmt.Sprintf("Subscription [%s] updated successfully!", msg.name), false),
			events.SubChangedCmd(),
		)

	case subAddedMsg:
		if msg.err != nil {
			return events.StatusCmd(fmt.Sprintf("Add subscription failed: %v", msg.err), true)
		}
		m.reload()
		return tea.Batch(
			events.StatusCmd(fmt.Sprintf("Subscription [%s] added and downloaded!", msg.name), false),
			events.SubChangedCmd(),
		)

	case subDeletedMsg:
		if msg.err != nil {
			return events.StatusCmd(fmt.Sprintf("Delete sub failed: %v", msg.err), true)
		}
		m.reload()
		return tea.Batch(
			events.StatusCmd(fmt.Sprintf("Subscription [%s] deleted!", msg.name), false),
			events.SubChangedCmd(),
		)

	case tea.KeyMsg:
		if m.modalActive == views.ModalAddSub {
			return m.updateAddSubModal(msg)
		}
		return m.handleKeyMsg(msg)
	}

	return nil
}

// View renders the Subscriptions page within the given terminal dimensions.
func (m *SubscriptionsModel) View(width, height int) string {
	return views.RenderSubscriptions(views.SubscriptionsProps{
		Width:            width,
		Height:           height,
		Subscriptions:    m.subscriptions,
		SelectedIdx:      m.selectedIdx,
		ModalActive:      m.modalActive,
		SubNameInputView: m.nameInput.View(),
		SubURLInputView:  m.urlInput.View(),
	})
}

func (m *SubscriptionsModel) currentSubscription() *config.Subscription {
	if len(m.subscriptions) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.subscriptions) {
		return nil
	}
	return &m.subscriptions[m.selectedIdx]
}

func (m *SubscriptionsModel) updateAddSubModal(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, keymaps.Esc):
		m.modalActive = views.ModalNone
		return nil

	case key.Matches(msg, keymaps.Down):
		if m.inputFocus == 0 {
			m.inputFocus = 1
			m.nameInput.Blur()
			m.urlInput.Focus()
		}
		return nil

	case key.Matches(msg, keymaps.Up):
		if m.inputFocus == 1 {
			m.inputFocus = 0
			m.urlInput.Blur()
			m.nameInput.Focus()
		}
		return nil

	case key.Matches(msg, keymaps.Enter):
		name := strings.TrimSpace(m.nameInput.Value())
		subURL := strings.TrimSpace(m.urlInput.Value())
		if name == "" || subURL == "" {
			return events.StatusCmd("Name and URL cannot be empty", true)
		}
		return tea.Batch(
			events.StatusCmd(fmt.Sprintf("Downloading subscription [%s]...", name), false),
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
	case key.Matches(msg, keymaps.Up):
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
	case key.Matches(msg, keymaps.Down):
		if m.selectedIdx < len(m.subscriptions)-1 {
			m.selectedIdx++
		}
	case key.Matches(msg, keymaps.UpdateSub):
		sub := m.currentSubscription()
		if sub != nil {
			return tea.Batch(
				events.StatusCmd(fmt.Sprintf("Updating subscription [%s]...", sub.Name), false),
				m.updateSubCmd(sub.Name),
			)
		}
	case key.Matches(msg, keymaps.UpdateAll):
		return tea.Batch(
			events.StatusCmd("Updating all subscriptions...", false),
			m.updateAllSubsCmd(),
		)
	case key.Matches(msg, keymaps.AddSub):
		m.modalActive = views.ModalAddSub
		m.inputFocus = 0
		m.nameInput.Focus()
		m.urlInput.Blur()
	case key.Matches(msg, keymaps.DeleteSub):
		sub := m.currentSubscription()
		if sub != nil {
			return tea.Batch(
				events.StatusCmd(fmt.Sprintf("Deleting subscription [%s]...", sub.Name), false),
				m.deleteSubCmd(sub.Name),
			)
		}
	}

	return nil
}

func (m *SubscriptionsModel) updateSubCmd(name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		_, err := m.profileMgr.UpdateSubscription(ctx, name)
		return subUpdatedMsg{name: name, err: err}
	}
}

func (m *SubscriptionsModel) updateAllSubsCmd() tea.Cmd {
	return func() tea.Msg {
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
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		_, err := m.profileMgr.AddSubscription(ctx, name, url)
		return subAddedMsg{name: name, err: err}
	}
}

func (m *SubscriptionsModel) deleteSubCmd(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.profileMgr.DeleteSubscription(name, true)
		return subDeletedMsg{name: name, err: err}
	}
}
