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
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

type (
	subUpdatedMsg struct {
		name     string
		reloaded bool
		err      error
	}
	subAddedMsg struct {
		name string
		err  error
	}
	subDeletedMsg struct {
		name string
		err  error
	}
	subAppliedMsg struct {
		name string
		err  error
	}
)

// SubscriptionsModel encapsulates state, interactions, and rendering for the Subscriptions tab.
type SubscriptionsModel struct {
	client        mihomo.Client
	profileMgr    *profile.Manager
	subscriptions []config.Subscription
	selectedIdx   int

	modalActive views.ModalType
	nameInput   textinput.Model
	urlInput    textinput.Model
	inputFocus  int // 0: Name, 1: URL
}

// NewSubscriptionsModel initializes a self-contained SubscriptionsModel.
func NewSubscriptionsModel(cli mihomo.Client, mgr *profile.Manager) *SubscriptionsModel {
	if cli == nil {
		panic("pages: nil mihomo client")
	}
	if mgr == nil {
		panic("pages: nil profile manager")
	}

	nameInput := textinput.New()
	nameInput.Placeholder = "e.g. My-Airport (a-z, A-Z, 0-9, -, _)"
	nameInput.Focus()
	nameInput.Width = 46

	urlInput := textinput.New()
	urlInput.Placeholder = "https://example.com/api/v1/..."
	urlInput.Width = 46

	return &SubscriptionsModel{
		client:      cli,
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
	if m.modalActive == views.ModalAddSub {
		return "[↑/↓] Switch Field | [Enter] Confirm & Download | [Esc] Cancel"
	}
	return "[↑/↓] Move | [Enter] Apply | [u] Update | [U] Update All | [a] Add Sub | [d] Delete"
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

// Update handles asynchronous subscription results and keyboard interactions for the Subscriptions page.
func (m *SubscriptionsModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case subUpdatedMsg:
		if msg.err != nil {
			return events.StatusCmd(fmt.Sprintf("Update sub failed: %v", msg.err), true)
		}
		m.reload()
		if msg.reloaded {
			return tea.Batch(
				events.StatusCmd(fmt.Sprintf("Subscription [%s] updated & hot-reloaded!", msg.name), false),
				events.ProfileAppliedCmd(),
			)
		}
		return events.StatusCmd(fmt.Sprintf("Subscription [%s] updated successfully!", msg.name), false)

	case subAddedMsg:
		if msg.err != nil {
			return events.StatusCmd(fmt.Sprintf("Add subscription failed: %v", msg.err), true)
		}
		m.reload()
		return events.StatusCmd(fmt.Sprintf("Subscription [%s] added and downloaded!", msg.name), false)

	case subDeletedMsg:
		if msg.err != nil {
			return events.StatusCmd(fmt.Sprintf("Delete sub failed: %v", msg.err), true)
		}
		m.reload()
		return events.StatusCmd(fmt.Sprintf("Subscription [%s] deleted!", msg.name), false)

	case subAppliedMsg:
		if msg.err != nil {
			return events.StatusCmd(fmt.Sprintf("Apply subscription failed: %v", msg.err), true)
		}
		m.reload()
		return tea.Batch(
			events.StatusCmd(fmt.Sprintf("Applied subscription [%s] & hot-reloaded successfully!", msg.name), false),
			events.ProfileAppliedCmd(),
		)

	case tea.KeyMsg:
		if m.modalActive == views.ModalAddSub {
			return m.updateAddSubModal(msg)
		}
		return m.handleKeyMsg(msg)
	}

	return nil
}

// reload closes and clears the add-subscription modal, resets input focus to the name field,
// refreshes m.subscriptions from profileMgr, and resets the list cursor to the first item.
func (m *SubscriptionsModel) reload() {
	m.modalActive = views.ModalNone
	m.inputFocus = 0
	m.nameInput.SetValue("")
	m.nameInput.Focus()
	m.urlInput.SetValue("")
	m.urlInput.Blur()

	m.selectedIdx = 0 // select first subscription
	m.subscriptions = m.profileMgr.ListSubscriptions()
	if len(m.subscriptions) == 0 {
		m.selectedIdx = 0
		return
	}
	if m.selectedIdx >= len(m.subscriptions) {
		m.selectedIdx = len(m.subscriptions) - 1
	}
}

// View renders the Subscriptions page within the given terminal dimensions.
func (m *SubscriptionsModel) View(width, height int) string {
	return views.RenderSubscriptions(views.SubscriptionsProps{
		Width:            width,
		Height:           height,
		Subscriptions:    m.subscriptions,
		ActiveSubName:    m.profileMgr.ActiveProfile(),
		SelectedIdx:      m.selectedIdx,
		ModalActive:      m.modalActive,
		SubNameInputView: m.nameInput.View(),
		SubURLInputView:  m.urlInput.View(),
	})
}

// currentSubscription returns the currently selected subscription, or nil if the list is empty.
func (m *SubscriptionsModel) currentSubscription() *config.Subscription {
	if len(m.subscriptions) == 0 || m.selectedIdx < 0 || m.selectedIdx >= len(m.subscriptions) {
		return nil
	}
	return &m.subscriptions[m.selectedIdx]
}

// updateAddSubModal handles keyboard input while the add-subscription modal is open.
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

// handleKeyMsg processes subscription list navigation and CRUD/apply shortcuts.
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
	case key.Matches(msg, keymaps.AddSub):
		m.modalActive = views.ModalAddSub
		m.inputFocus = 0
		m.nameInput.Focus()
		m.urlInput.Blur()
	case key.Matches(msg, keymaps.UpdateSub):
		sub := m.currentSubscription()
		if sub == nil {
			return nil
		}
		return tea.Batch(
			events.StatusCmd(fmt.Sprintf("Updating subscription [%s]...", sub.Name), false),
			m.updateSubCmd(sub.Name),
		)
	case key.Matches(msg, keymaps.UpdateAll):
		return tea.Batch(
			events.StatusCmd("Updating all subscriptions...", false),
			m.updateAllSubsCmd(),
		)
	case key.Matches(msg, keymaps.DeleteSub):
		sub := m.currentSubscription()
		if sub == nil {
			return nil
		}
		return tea.Batch(
			events.StatusCmd(fmt.Sprintf("Deleting subscription [%s]...", sub.Name), false),
			m.deleteSubCmd(sub.Name),
		)
	case key.Matches(msg, keymaps.ApplySub):
		sub := m.currentSubscription()
		if sub == nil {
			return nil
		}
		return tea.Batch(
			events.StatusCmd(fmt.Sprintf("Applying subscription [%s]...", sub.Name), false),
			m.applySubCmd(sub.Name),
		)
	}

	return nil
}

// updateSubCmd re-downloads a single subscription and reloads Mihomo if it is currently active.
func (m *SubscriptionsModel) updateSubCmd(name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		defer cancel()
		if _, err := m.profileMgr.UpdateSubscription(ctx, name); err != nil {
			return subUpdatedMsg{name: name, err: err}
		}

		if m.profileMgr.ActiveProfile() == name {
			if err := m.mergeAndReload(ctx, name); err != nil {
				return subUpdatedMsg{name: name, err: err}
			}
			return subUpdatedMsg{name: name, reloaded: true}
		}
		return subUpdatedMsg{name: name}
	}
}

// updateAllSubsCmd sequentially updates all configured subscriptions and reloads Mihomo if one is active.
func (m *SubscriptionsModel) updateAllSubsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		for _, s := range m.profileMgr.ListSubscriptions() {
			if _, err := m.profileMgr.UpdateSubscription(ctx, s.Name); err != nil {
				return subUpdatedMsg{name: "All", err: err}
			}
		}

		if active := m.profileMgr.ActiveProfile(); active != "" {
			if err := m.mergeAndReload(ctx, active); err != nil {
				return subUpdatedMsg{name: "All", err: err}
			}
			return subUpdatedMsg{name: "All", reloaded: true}
		}
		return subUpdatedMsg{name: "All"}
	}
}

// addSubCmd asynchronously downloads and registers a new subscription.
func (m *SubscriptionsModel) addSubCmd(name, url string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		_, err := m.profileMgr.AddSubscription(ctx, name, url)
		return subAddedMsg{
			name: name,
			err:  err,
		}
	}
}

// deleteSubCmd asynchronously deletes a subscription record and its local YAML file.
func (m *SubscriptionsModel) deleteSubCmd(name string) tea.Cmd {
	return func() tea.Msg {
		return subDeletedMsg{
			name: name,
			err:  m.profileMgr.DeleteSubscription(name),
		}
	}
}

// applySubCmd merges the subscription with override.yaml, reloads Mihomo, and persists it as active.
func (m *SubscriptionsModel) applySubCmd(name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		if err := m.mergeAndReload(ctx, name); err != nil {
			return subAppliedMsg{name: name, err: err}
		}
		if err := m.profileMgr.SetActiveProfile(name); err != nil {
			return subAppliedMsg{name: name, err: err}
		}
		return subAppliedMsg{name: name}
	}
}

// mergeAndReload builds the merged Mihomo config.yaml for subName and triggers a forced core reload.
func (m *SubscriptionsModel) mergeAndReload(ctx context.Context, subName string) error {
	mihomoCfgPath, err := m.profileMgr.BuildMihomoConfig(subName)
	if err != nil {
		return err
	}
	return m.client.ReloadConfig(ctx, mihomoCfgPath, true)
}
