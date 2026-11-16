package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jackie8tao/mihotui/internal/app/theme"
	"github.com/jackie8tao/mihotui/internal/app/views"
	"github.com/jackie8tao/mihotui/internal/config"
	"github.com/jackie8tao/mihotui/internal/profile"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

type TabIndex int

const (
	TabOverview TabIndex = iota
	TabProxies
	TabConfigs
	TabSubscriptions
)

type FocusPane = views.FocusPane

const (
	PaneGroups = views.PaneGroups
	PaneNodes  = views.PaneNodes
)

type ModalType = views.ModalType

const (
	ModalNone   = views.ModalNone
	ModalAddSub = views.ModalAddSub

	defaultTestURL       = "http://www.gstatic.com/generate_204"
	defaultTestTimeoutMs = 3000
	defaultRefreshRateMs = 1000
)

// Messages
type (
	initialDataMsg struct {
		version *mihomo.Version
		config  *mihomo.Config
		proxies *mihomo.ProxiesResponse
		memory  *mihomo.Memory
		err     error
	}
	trafficMsg     mihomo.Traffic
	delayResultMsg struct {
		proxyName string
		delay     int
		err       error
	}
	modeUpdatedMsg struct {
		newMode string
		err     error
	}
	proxySelectedMsg struct {
		groupName string
		proxyName string
		err       error
	}
	profilesLoadedMsg struct {
		profiles []profile.Profile
		err      error
	}
	profileAppliedMsg struct {
		profileName string
		err         error
	}
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
	tickMsg time.Time
)

// AppModel is the main Bubble Tea model.
type AppModel struct {
	client     mihomo.Client
	cfg        *config.AppConfig
	profileMgr *profile.Manager
	keys       KeyMap

	// Global Layout & Navigation
	width     int
	height    int
	activeTab TabIndex

	// Tab-specific States
	proxiesState ProxiesTabState
	configsState ConfigsTabState
	subsState    SubscriptionsTabState

	// Shared Domain Data
	version       mihomo.Version
	config        mihomo.Config
	memory        mihomo.Memory
	proxies       map[string]mihomo.ProxyItem
	groups        []mihomo.ProxyItem
	profiles      []profile.Profile
	subscriptions []profile.Subscription
	traffic       mihomo.Traffic
	totalUp       int64
	totalDown     int64

	// Channels & Async Lifecycle
	trafficChan chan mihomo.Traffic
	cancelCtx   context.CancelFunc

	// Status / Notifications
	statusMsg string
	isError   bool
}

// New initializes and returns a new AppModel.
func New(cli mihomo.Client, cfg *config.AppConfig, mgr *profile.Manager) *AppModel {
	return NewAppModel(cli, cfg, mgr)
}

// NewAppModel initializes the AppModel.
func NewAppModel(cli mihomo.Client, cfg *config.AppConfig, mgr *profile.Manager) *AppModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "e.g. My-Airport"
	nameInput.Focus()
	nameInput.Width = 46

	urlInput := textinput.New()
	urlInput.Placeholder = "https://sub.example.com/api/v1/..."
	urlInput.Width = 46

	return &AppModel{
		client:     cli,
		cfg:        cfg,
		profileMgr: mgr,
		keys:       DefaultKeyMap(),
		activeTab:  TabOverview,
		width:      80,
		height:     24,
		proxiesState: ProxiesTabState{
			FocusPane: PaneGroups,
		},
		subsState: SubscriptionsTabState{
			ModalActive: ModalNone,
			NameInput:   nameInput,
			URLInput:    urlInput,
			InputFocus:  0,
		},
		proxies:     make(map[string]mihomo.ProxyItem),
		trafficChan: make(chan mihomo.Traffic, 50),
	}
}

func (m *AppModel) Init() tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelCtx = cancel

	// Start WebSocket traffic listener in background
	go func() {
		_ = m.client.SubscribeTraffic(ctx, m.trafficChan)
	}()

	return tea.Batch(
		m.fetchDataCmd(),
		m.fetchProfilesCmd(),
		m.waitForTrafficCmd(),
		m.tickCmd(),
	)
}

func (m *AppModel) tickCmd() tea.Cmd {
	return tea.Tick(time.Duration(defaultRefreshRateMs)*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *AppModel) waitForTrafficCmd() tea.Cmd {
	return func() tea.Msg {
		t, ok := <-m.trafficChan
		if !ok {
			return nil
		}
		return trafficMsg(t)
	}
}

func (m *AppModel) fetchDataCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		v, err := m.client.GetVersion(ctx)
		if err != nil {
			return initialDataMsg{err: err}
		}
		c, err := m.client.GetConfig(ctx)
		if err != nil {
			return initialDataMsg{err: err}
		}
		p, err := m.client.GetProxies(ctx)
		if err != nil {
			return initialDataMsg{err: err}
		}
		mem, _ := m.client.GetMemory(ctx)

		return initialDataMsg{
			version: v,
			config:  c,
			proxies: p,
			memory:  mem,
		}
	}
}

func (m *AppModel) updateModeCmd(mode string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := m.client.UpdateMode(ctx, mode)
		return modeUpdatedMsg{newMode: mode, err: err}
	}
}

func (m *AppModel) handleGlobalKeys(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		if m.cancelCtx != nil {
			m.cancelCtx()
		}
		return tea.Quit, true

	case key.Matches(msg, m.keys.Tab):
		switch m.activeTab {
		case TabOverview:
			m.activeTab = TabProxies
		case TabProxies:
			m.activeTab = TabConfigs
		case TabConfigs:
			m.activeTab = TabSubscriptions
		case TabSubscriptions:
			m.activeTab = TabOverview
		}
		return nil, true

	case key.Matches(msg, m.keys.Refresh):
		m.statusMsg = "Refreshing data..."
		return tea.Batch(m.fetchDataCmd(), m.fetchProfilesCmd()), true

	case key.Matches(msg, m.keys.SwitchMode):
		curr := strings.ToLower(m.config.Mode)
		next := "rule"
		switch curr {
		case "rule":
			next = "global"
		case "global":
			next = "direct"
		default:
			next = "rule"
		}
		return m.updateModeCmd(next), true
	}

	return nil, false
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		cmds = append(cmds, m.tickCmd())

	case trafficMsg:
		m.traffic = mihomo.Traffic(msg)
		m.totalUp += m.traffic.Up
		m.totalDown += m.traffic.Down
		cmds = append(cmds, m.waitForTrafficCmd())

	case initialDataMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("API Connection Error: %v", msg.err)
			m.isError = true
			return m, nil
		}

		m.version = *msg.version
		m.config = *msg.config
		if msg.memory != nil {
			m.memory = *msg.memory
		}
		m.proxies = msg.proxies.Proxies

		// Filter and sort proxy groups (Selector, URLTest, Fallback, etc.)
		var groups []mihomo.ProxyItem
		for _, item := range m.proxies {
			if len(item.All) > 0 {
				groups = append(groups, item)
			}
		}
		sort.Slice(groups, func(i, j int) bool {
			if groups[i].Name == "GLOBAL" {
				return true
			}
			if groups[j].Name == "GLOBAL" {
				return false
			}
			return groups[i].Name < groups[j].Name
		})
		m.groups = groups
		if m.proxiesState.GroupCursor >= len(m.groups) {
			m.proxiesState.GroupCursor = 0
		}
		m.statusMsg = "Connected to Mihomo"
		m.isError = false
		return m, nil

	case profilesLoadedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Load profiles error: %v", msg.err)
			m.isError = true
			return m, nil
		}

		m.profiles = msg.profiles
		if m.configsState.Cursor >= len(m.profiles) {
			m.configsState.Cursor = 0
		}
		if m.profileMgr != nil {
			m.subscriptions = m.profileMgr.ListSubscriptions()
			if m.subsState.Cursor >= len(m.subscriptions) {
				m.subsState.Cursor = 0
			}
		}
		return m, nil

	case profileAppliedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Hot-reload profile failed: %v", msg.err)
			m.isError = true
			return m, nil
		}

		m.statusMsg = fmt.Sprintf("Applied profile [%s] & hot-reloaded successfully!", msg.profileName)
		m.isError = false
		return m, tea.Batch(m.fetchProfilesCmd(), m.fetchDataCmd())

	case subUpdatedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Update sub failed: %v", msg.err)
			m.isError = true
			return m, nil
		}

		m.statusMsg = fmt.Sprintf("Subscription [%s] updated successfully!", msg.name)
		m.isError = false
		return m, m.fetchProfilesCmd()

	case subAddedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Add subscription failed: %v", msg.err)
			m.isError = true
			return m, nil
		}

		m.statusMsg = fmt.Sprintf("Subscription [%s] added and downloaded!", msg.name)
		m.isError = false
		m.subsState.ModalActive = ModalNone
		m.subsState.NameInput.SetValue("")
		m.subsState.URLInput.SetValue("")
		return m, m.fetchProfilesCmd()

	case subDeletedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Delete sub failed: %v", msg.err)
			m.isError = true
			return m, nil
		}

		m.statusMsg = fmt.Sprintf("Subscription [%s] deleted!", msg.name)
		m.isError = false
		return m, m.fetchProfilesCmd()

	case delayResultMsg:
		item, ok := m.proxies[msg.proxyName]
		if !ok {
			return m, nil
		}
		item.TestingDelay = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Latency test failed for %s: %v", msg.proxyName, msg.err)
			m.isError = true
			m.proxies[msg.proxyName] = item
			return m, nil
		}

		item.History = append(item.History, mihomo.DelayHistory{
			Time:  time.Now(),
			Delay: msg.delay,
		})
		m.statusMsg = fmt.Sprintf("Latency for %s: %s", msg.proxyName, mihomo.FormatDelay(msg.delay))
		m.isError = false
		m.proxies[msg.proxyName] = item
		return m, nil

	case proxySelectedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Select proxy failed: %v", msg.err)
			m.isError = true
			return m, nil
		}

		m.statusMsg = fmt.Sprintf("[%s] => Switched to: %s", msg.groupName, msg.proxyName)
		m.isError = false
		return m, m.fetchDataCmd()

	case modeUpdatedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Update mode failed: %v", msg.err)
			m.isError = true
			return m, nil
		}

		m.config.Mode = msg.newMode
		m.statusMsg = fmt.Sprintf("Mode switched to: %s", strings.ToUpper(msg.newMode))
		m.isError = false
		return m, nil

	case tea.KeyMsg:
		// 1. Modal Interception
		if cmd, handled := m.updateAddSubModal(msg); handled {
			return m, cmd
		}

		// 2. Global Hotkeys
		if cmd, handled := m.handleGlobalKeys(msg); handled {
			return m, cmd
		}

		// 3. Tab-specific Updates
		switch m.activeTab {
		case TabProxies:
			if cmd := m.updateProxiesTab(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		case TabConfigs:
			if cmd := m.updateConfigsTab(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		case TabSubscriptions:
			if cmd := m.updateSubscriptionsTab(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *AppModel) View() string {
	// Top Header with Tabs
	title := theme.TitleStyle.Render(" MihoTui ")

	tab4Name := "4. Subs"
	if m.width >= 90 {
		tab4Name = "4. Subscriptions"
	}

	tab1 := theme.TabInactiveStyle.Render("1. Overview")
	tab2 := theme.TabInactiveStyle.Render("2. Proxies")
	tab3 := theme.TabInactiveStyle.Render("3. Configs")
	tab4 := theme.TabInactiveStyle.Render(tab4Name)

	switch m.activeTab {
	case TabOverview:
		tab1 = theme.TabActiveStyle.Render("1. Overview")
	case TabProxies:
		tab2 = theme.TabActiveStyle.Render("2. Proxies")
	case TabConfigs:
		tab3 = theme.TabActiveStyle.Render("3. Configs")
	case TabSubscriptions:
		tab4 = theme.TabActiveStyle.Render(tab4Name)
	}

	tabs := lipgloss.JoinHorizontal(lipgloss.Top, tab1, " ", tab2, " ", tab3, " ", tab4)

	modeBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorDark).
		Background(theme.ColorSecondary).
		Padding(0, 1).
		Render(strings.ToUpper(m.config.Mode))

	header := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", tabs, "  ", modeBadge)

	// Main Body
	var body string
	switch m.activeTab {
	case TabOverview:
		body = views.RenderOverview(views.OverviewProps{
			Width:     m.width,
			Traffic:   m.traffic,
			TotalUp:   m.totalUp,
			TotalDown: m.totalDown,
			Config:    m.config,
			Version:   m.version,
			Memory:    m.memory,
		})
	case TabProxies:
		body = views.RenderProxies(views.ProxiesProps{
			Width:       m.width,
			Height:      m.height,
			FocusPane:   m.proxiesState.FocusPane,
			Groups:      m.groups,
			Proxies:     m.proxies,
			GroupCursor: m.proxiesState.GroupCursor,
			NodeCursor:  m.proxiesState.NodeCursor,
		})
	case TabConfigs:
		body = views.RenderConfigs(views.ConfigsProps{
			Width:         m.width,
			Height:        m.height,
			ConfigDir:     m.profileMgr.ConfigDir(),
			Profiles:      m.profiles,
			ProfileCursor: m.configsState.Cursor,
		})
	case TabSubscriptions:
		body = views.RenderSubscriptions(views.SubscriptionsProps{
			Width:            m.width,
			Height:           m.height,
			Subscriptions:    m.subscriptions,
			SubCursor:        m.subsState.Cursor,
			ModalActive:      m.subsState.ModalActive,
			SubNameInputView: m.subsState.NameInput.View(),
			SubURLInputView:  m.subsState.URLInput.View(),
		})
	}

	// Bottom Status & Help
	statusColor := theme.ColorMuted
	if m.isError {
		statusColor = theme.ColorError
	}
	status := lipgloss.NewStyle().Foreground(statusColor).Render(" " + m.statusMsg)

	helpText := "[Tab] Switch View | [m] Mode | [r] Refresh | [q] Quit"
	switch m.activeTab {
	case TabProxies:
		helpText = "[←/→] Pane | [↑/↓] Move | [Enter] Select | [t] Ping | [T] Ping Group | " + helpText
	case TabConfigs:
		helpText = "[↑/↓] Move | [Enter] Apply Profile | " + helpText
	case TabSubscriptions:
		helpText = "[↑/↓] Move | [Enter/u] Update | [U] Update All | [a] Add Sub | [d] Delete | " + helpText
	}
	helpBar := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(helpText)

	footer := lipgloss.JoinVertical(lipgloss.Left, status, helpBar)
	fullView := lipgloss.JoinVertical(lipgloss.Left, header, "\n", body, "\n", footer)

	// Ensure view height never exceeds terminal height to guarantee top header stays visible
	if m.height > 0 {
		lines := strings.Split(fullView, "\n")
		if len(lines) > m.height {
			lines = lines[:m.height]
			return strings.Join(lines, "\n")
		}
	}

	return fullView
}
