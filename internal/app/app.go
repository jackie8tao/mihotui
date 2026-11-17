package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jackie8tao/mihotui/internal/app/tabs"
	"github.com/jackie8tao/mihotui/internal/app/theme"
	"github.com/jackie8tao/mihotui/internal/profile"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

// AppModel is the main Bubble Tea container model routing events to independent tabs.Page sub-models.
type AppModel struct {
	client    mihomo.Client
	keys      KeyMap
	tabs      []tabs.Page
	activeTab int

	width  int
	height int
	mode   string

	trafficChan chan mihomo.Traffic
	cancelCtx   context.CancelFunc

	statusMsg string
	isError   bool
}

// New initializes and returns a new AppModel.
func New(cli mihomo.Client, mgr *profile.Manager) *AppModel {
	return &AppModel{
		client: cli,
		keys:   DefaultKeyMap(),
		tabs: []tabs.Page{
			tabs.NewOverviewModel(cli),
			tabs.NewProxiesModel(cli),
			tabs.NewConfigsModel(cli, mgr),
			tabs.NewSubscriptionsModel(mgr),
		},
		width:       80,
		height:      24,
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

	cmds := []tea.Cmd{m.waitForTrafficCmd()}
	for _, tab := range m.tabs {
		if cmd := tab.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (m *AppModel) waitForTrafficCmd() tea.Cmd {
	return func() tea.Msg {
		t, ok := <-m.trafficChan
		if !ok {
			return nil
		}
		return tabs.TrafficMsg(t)
	}
}

func (m *AppModel) updateModeCmd(mode string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := m.client.UpdateMode(ctx, mode)
		return tabs.ModeUpdatedMsg{NewMode: mode, Err: err}
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
		if len(m.tabs) > 0 {
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
		}
		return nil, true

	case key.Matches(msg, m.keys.Refresh):
		m.statusMsg = "Refreshing data..."
		m.isError = false
		var cmds []tea.Cmd
		for _, tab := range m.tabs {
			if cmd := tab.Init(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return tea.Batch(cmds...), true

	case key.Matches(msg, m.keys.SwitchMode):
		next := "rule"
		switch strings.ToLower(m.mode) {
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

func (m *AppModel) broadcast(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for _, tab := range m.tabs {
		if cmd := tab.Update(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tabs.TrafficMsg:
		return m, tea.Batch(m.waitForTrafficCmd(), m.broadcast(msg))

	case tabs.StatusMsg:
		m.statusMsg = msg.Text
		m.isError = msg.IsError
		return m, nil

	case tabs.ModeSyncedMsg:
		m.mode = msg.Mode
		return m, nil

	case tabs.ModeUpdatedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Update mode failed: %v", msg.Err)
			m.isError = true
			return m, nil
		}
		m.mode = msg.NewMode
		m.statusMsg = fmt.Sprintf("Mode switched to: %s", strings.ToUpper(msg.NewMode))
		m.isError = false
		return m, m.broadcast(msg)

	case tea.KeyMsg:
		if len(m.tabs) == 0 {
			return m, nil
		}
		active := m.tabs[m.activeTab]
		if active.CapturingInput() {
			return m, active.Update(msg)
		}
		if cmd, handled := m.handleGlobalKeys(msg); handled {
			return m, cmd
		}
		return m, active.Update(msg)

	default:
		return m, m.broadcast(msg)
	}
}

func (m *AppModel) View() string {
	if len(m.tabs) == 0 {
		return ""
	}

	title := theme.TitleStyle.Render(" MihoTui ")

	var renderedTabs []string
	for i, tab := range m.tabs {
		if i == m.activeTab {
			renderedTabs = append(renderedTabs, theme.TabActiveStyle.Render(tab.Title()))
		} else {
			renderedTabs = append(renderedTabs, theme.TabInactiveStyle.Render(tab.Title()))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	modeBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorDark).
		Background(theme.ColorSecondary).
		Padding(0, 1).
		Render(strings.ToUpper(m.mode))

	header := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", tabBar, "  ", modeBadge)

	active := m.tabs[m.activeTab]
	body := active.View(m.width, m.height)

	statusColor := theme.ColorMuted
	if m.isError {
		statusColor = theme.ColorError
	}
	status := lipgloss.NewStyle().Foreground(statusColor).Render(" " + m.statusMsg)

	helpText := "[Tab] Switch View | [m] Mode | [r] Refresh | [q] Quit"
	if tabHelp := active.Help(); tabHelp != "" {
		helpText = tabHelp + " | " + helpText
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
