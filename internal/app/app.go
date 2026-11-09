package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"mihomo-tui/internal/app/theme"
	"mihomo-tui/internal/app/views"
	"mihomo-tui/internal/config"
	"mihomo-tui/internal/profile"
	"mihomo-tui/pkg/mihomo"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	statusClearMsg struct{}
	tickMsg        time.Time
)

// AppModel is the main Bubble Tea model.
type AppModel struct {
	client     mihomo.Client
	cfg        *config.AppConfig
	profileMgr *profile.Manager
	keys       KeyMap

	// Layout & State
	width         int
	height        int
	activeTab     TabIndex
	focusPane     FocusPane
	groupCursor   int
	nodeCursor    int
	profileCursor int
	subCursor     int

	// Modal State (Add Subscription)
	modalActive  ModalType
	subNameInput textinput.Model
	subURLInput  textinput.Model
	inputFocus   int // 0: Name, 1: URL

	// Data
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

	// Channels & Async
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
		client:       cli,
		cfg:          cfg,
		profileMgr:   mgr,
		keys:         DefaultKeyMap(),
		activeTab:    TabOverview,
		focusPane:    PaneGroups,
		modalActive:  ModalNone,
		subNameInput: nameInput,
		subURLInput:  urlInput,
		inputFocus:   0,
		proxies:      make(map[string]mihomo.ProxyItem),
		trafficChan:  make(chan mihomo.Traffic, 50),
		width:        80,
		height:       24,
	}
}

func (m *AppModel) currentGroup() *mihomo.ProxyItem {
	if len(m.groups) == 0 || m.groupCursor < 0 || m.groupCursor >= len(m.groups) {
		return nil
	}
	return &m.groups[m.groupCursor]
}

func (m *AppModel) currentSelectedNode() string {
	grp := m.currentGroup()
	if grp == nil || len(grp.All) == 0 || m.nodeCursor < 0 || m.nodeCursor >= len(grp.All) {
		return ""
	}
	return grp.All[m.nodeCursor]
}

func (m *AppModel) currentProfile() *profile.Profile {
	if len(m.profiles) == 0 || m.profileCursor < 0 || m.profileCursor >= len(m.profiles) {
		return nil
	}
	return &m.profiles[m.profileCursor]
}

func (m *AppModel) currentSubscription() *profile.Subscription {
	if len(m.subscriptions) == 0 || m.subCursor < 0 || m.subCursor >= len(m.subscriptions) {
		return nil
	}
	return &m.subscriptions[m.subCursor]
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

func (m *AppModel) testDelayCmd(nodeName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(defaultTestTimeoutMs+1000)*time.Millisecond)
		defer cancel()
		delay, err := m.client.TestDelay(ctx, nodeName, defaultTestURL, defaultTestTimeoutMs)
		return delayResultMsg{proxyName: nodeName, delay: delay, err: err}
	}
}

func (m *AppModel) selectProxyCmd(groupName, nodeName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := m.client.SelectProxy(ctx, groupName, nodeName)
		return proxySelectedMsg{groupName: groupName, proxyName: nodeName, err: err}
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
		} else {
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
			if m.groupCursor >= len(m.groups) {
				m.groupCursor = 0
			}
			m.statusMsg = "Connected to Mihomo"
			m.isError = false
		}

	case profilesLoadedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Load profiles error: %v", msg.err)
			m.isError = true
		} else {
			m.profiles = msg.profiles
			if m.profileCursor >= len(m.profiles) {
				m.profileCursor = 0
			}
			if m.profileMgr != nil {
				m.subscriptions = m.profileMgr.ListSubscriptions()
				if m.subCursor >= len(m.subscriptions) {
					m.subCursor = 0
				}
			}
		}

	case profileAppliedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Hot-reload profile failed: %v", msg.err)
			m.isError = true
		} else {
			m.statusMsg = fmt.Sprintf("Applied profile [%s] & hot-reloaded successfully!", msg.profileName)
			m.isError = false
			cmds = append(cmds, m.fetchProfilesCmd(), m.fetchDataCmd())
		}

	case subUpdatedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Update sub failed: %v", msg.err)
			m.isError = true
		} else {
			m.statusMsg = fmt.Sprintf("Subscription [%s] updated successfully!", msg.name)
			m.isError = false
			cmds = append(cmds, m.fetchProfilesCmd())
		}

	case subAddedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Add subscription failed: %v", msg.err)
			m.isError = true
		} else {
			m.statusMsg = fmt.Sprintf("Subscription [%s] added and downloaded!", msg.name)
			m.isError = false
			m.modalActive = ModalNone
			m.subNameInput.Reset()
			m.subURLInput.Reset()
			cmds = append(cmds, m.fetchProfilesCmd())
		}

	case subDeletedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Delete subscription failed: %v", msg.err)
			m.isError = true
		} else {
			m.statusMsg = fmt.Sprintf("Deleted subscription [%s]", msg.name)
			m.isError = false
			cmds = append(cmds, m.fetchProfilesCmd())
		}

	case delayResultMsg:
		if item, ok := m.proxies[msg.proxyName]; ok {
			item.TestingDelay = false
			item.History = append(item.History, mihomo.DelayHistory{
				Time:  time.Now(),
				Delay: msg.delay,
			})
			m.proxies[msg.proxyName] = item
		}
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Test delay for %s failed: %v", msg.proxyName, msg.err)
		} else {
			m.statusMsg = fmt.Sprintf("Latency [%s]: %s", msg.proxyName, mihomo.FormatDelay(msg.delay))
		}

	case proxySelectedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Select proxy failed: %v", msg.err)
			m.isError = true
		} else {
			m.statusMsg = fmt.Sprintf("Switched [%s] -> %s", msg.groupName, msg.proxyName)
			m.isError = false
			for i, g := range m.groups {
				if g.Name == msg.groupName {
					m.groups[i].Now = msg.proxyName
					break
				}
			}
		}

	case modeUpdatedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Update mode failed: %v", msg.err)
			m.isError = true
		} else {
			m.config.Mode = msg.newMode
			m.statusMsg = fmt.Sprintf("Mode switched to: %s", strings.ToUpper(msg.newMode))
			m.isError = false
		}

	case tea.KeyMsg:
		// Modal Interception
		if m.modalActive == ModalAddSub {
			switch msg.String() {
			case "esc":
				m.modalActive = ModalNone
				return m, nil
			case "tab", "down":
				if m.inputFocus == 0 {
					m.inputFocus = 1
					m.subNameInput.Blur()
					m.subURLInput.Focus()
				} else {
					m.inputFocus = 0
					m.subURLInput.Blur()
					m.subNameInput.Focus()
				}
				return m, nil
			case "up":
				if m.inputFocus == 1 {
					m.inputFocus = 0
					m.subURLInput.Blur()
					m.subNameInput.Focus()
				}
				return m, nil
			case "enter":
				name := strings.TrimSpace(m.subNameInput.Value())
				subURL := strings.TrimSpace(m.subURLInput.Value())
				if name == "" || subURL == "" {
					m.statusMsg = "Name and URL cannot be empty"
					m.isError = true
					return m, nil
				}
				m.statusMsg = fmt.Sprintf("Downloading subscription [%s]...", name)
				return m, m.addSubCmd(name, subURL)
			default:
				var cmd tea.Cmd
				if m.inputFocus == 0 {
					m.subNameInput, cmd = m.subNameInput.Update(msg)
				} else {
					m.subURLInput, cmd = m.subURLInput.Update(msg)
				}
				return m, cmd
			}
		}

		// Standard Navigation & Actions
		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.cancelCtx != nil {
				m.cancelCtx()
			}
			return m, tea.Quit

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

		case key.Matches(msg, m.keys.Refresh):
			m.statusMsg = "Refreshing data..."
			cmds = append(cmds, m.fetchDataCmd(), m.fetchProfilesCmd())

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
			cmds = append(cmds, m.updateModeCmd(next))

		// TabProxies Navigation
		case m.activeTab == TabProxies:
			switch {
			case key.Matches(msg, m.keys.Left):
				m.focusPane = PaneGroups

			case key.Matches(msg, m.keys.Right):
				if len(m.groups) > 0 {
					m.focusPane = PaneNodes
				}

			case key.Matches(msg, m.keys.Up):
				if m.focusPane == PaneGroups {
					if m.groupCursor > 0 {
						m.groupCursor--
						m.nodeCursor = 0
					}
				} else {
					if m.nodeCursor > 0 {
						m.nodeCursor--
					}
				}

			case key.Matches(msg, m.keys.Down):
				if m.focusPane == PaneGroups {
					if m.groupCursor < len(m.groups)-1 {
						m.groupCursor++
						m.nodeCursor = 0
					}
				} else {
					grp := m.currentGroup()
					if grp != nil && m.nodeCursor < len(grp.All)-1 {
						m.nodeCursor++
					}
				}

			case key.Matches(msg, m.keys.Enter):
				if m.focusPane == PaneNodes {
					grp := m.currentGroup()
					node := m.currentSelectedNode()
					if grp != nil && node != "" {
						cmds = append(cmds, m.selectProxyCmd(grp.Name, node))
					}
				} else {
					m.focusPane = PaneNodes
				}

			case key.Matches(msg, m.keys.TestDelay):
				node := m.currentSelectedNode()
				if node != "" {
					if item, ok := m.proxies[node]; ok {
						item.TestingDelay = true
						m.proxies[node] = item
					}
					m.statusMsg = fmt.Sprintf("Testing latency for %s...", node)
					cmds = append(cmds, m.testDelayCmd(node))
				}

			case key.Matches(msg, m.keys.TestAll):
				grp := m.currentGroup()
				if grp != nil && len(grp.All) > 0 {
					m.statusMsg = fmt.Sprintf("Testing all %d nodes in [%s]...", len(grp.All), grp.Name)
					for _, node := range grp.All {
						if item, ok := m.proxies[node]; ok {
							item.TestingDelay = true
							m.proxies[node] = item
						}
					}
					nodesToTest := append([]string(nil), grp.All...)
					cmds = append(cmds, func() tea.Msg {
						var wg sync.WaitGroup
						sem := make(chan struct{}, 8)
						for _, node := range nodesToTest {
							wg.Add(1)
							go func(n string) {
								defer wg.Done()
								sem <- struct{}{}
								defer func() { <-sem }()
								ctx, cancel := context.WithTimeout(context.Background(), time.Duration(defaultTestTimeoutMs+1000)*time.Millisecond)
								defer cancel()
								_, _ = m.client.TestDelay(ctx, n, defaultTestURL, defaultTestTimeoutMs)
							}(node)
						}
						wg.Wait()
						return m.fetchDataCmd()()
					})
				}
			}

		// TabConfigs Navigation
		case m.activeTab == TabConfigs:
			switch {
			case key.Matches(msg, m.keys.Up):
				if m.profileCursor > 0 {
					m.profileCursor--
				}
			case key.Matches(msg, m.keys.Down):
				if m.profileCursor < len(m.profiles)-1 {
					m.profileCursor++
				}
			case key.Matches(msg, m.keys.Enter):
				p := m.currentProfile()
				if p != nil {
					m.statusMsg = fmt.Sprintf("Applying profile [%s] to Mihomo...", p.Name)
					cmds = append(cmds, m.applyProfileCmd(p))
				}
			}

		// TabSubscriptions Navigation
		case m.activeTab == TabSubscriptions:
			switch {
			case key.Matches(msg, m.keys.Up):
				if m.subCursor > 0 {
					m.subCursor--
				}
			case key.Matches(msg, m.keys.Down):
				if m.subCursor < len(m.subscriptions)-1 {
					m.subCursor++
				}
			case key.Matches(msg, m.keys.UpdateSub), key.Matches(msg, m.keys.Enter):
				sub := m.currentSubscription()
				if sub != nil {
					m.statusMsg = fmt.Sprintf("Updating subscription [%s]...", sub.Name)
					cmds = append(cmds, m.updateSubCmd(sub.ID, sub.Name))
				}
			case key.Matches(msg, m.keys.UpdateAll):
				m.statusMsg = "Updating all subscriptions..."
				cmds = append(cmds, m.updateAllSubsCmd())
			case key.Matches(msg, m.keys.AddSub):
				m.modalActive = ModalAddSub
				m.inputFocus = 0
				m.subNameInput.Focus()
				m.subURLInput.Blur()
				return m, nil
			case key.Matches(msg, m.keys.DeleteSub):
				sub := m.currentSubscription()
				if sub != nil {
					m.statusMsg = fmt.Sprintf("Deleting subscription [%s]...", sub.Name)
					cmds = append(cmds, m.deleteSubCmd(sub.ID, sub.Name))
				}
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
			FocusPane:   m.focusPane,
			Groups:      m.groups,
			Proxies:     m.proxies,
			GroupCursor: m.groupCursor,
			NodeCursor:  m.nodeCursor,
		})
	case TabConfigs:
		body = views.RenderConfigs(views.ConfigsProps{
			Width:         m.width,
			Height:        m.height,
			ConfigDir:     m.profileMgr.ConfigDir(),
			Profiles:      m.profiles,
			ProfileCursor: m.profileCursor,
		})
	case TabSubscriptions:
		body = views.RenderSubscriptions(views.SubscriptionsProps{
			Width:            m.width,
			Height:           m.height,
			Subscriptions:    m.subscriptions,
			SubCursor:        m.subCursor,
			ModalActive:      m.modalActive,
			SubNameInputView: m.subNameInput.View(),
			SubURLInputView:  m.subURLInput.View(),
		})
	}

	// Bottom Status & Help
	statusColor := theme.ColorMuted
	if m.isError {
		statusColor = theme.ColorError
	}
	status := lipgloss.NewStyle().Foreground(statusColor).Render(" " + m.statusMsg)

	helpText := "[Tab] Switch View | [m] Mode | [r] Refresh | [q] Quit"
	if m.activeTab == TabProxies {
		helpText = "[←/→] Pane | [↑/↓] Move | [Enter] Select | [t] Ping | [T] Ping Group | " + helpText
	} else if m.activeTab == TabConfigs {
		helpText = "[↑/↓] Move | [Enter] Apply Profile | " + helpText
	} else if m.activeTab == TabSubscriptions {
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
