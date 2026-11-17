package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jackie8tao/mihotui/internal/app/theme"
	"github.com/jackie8tao/mihotui/internal/profile"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

// TabPage defines the contract for an independent TUI tab sub-model.
type TabPage interface {
	Title() string
	Help() string
	CapturingInput() bool
	Init() tea.Cmd
	Update(msg tea.Msg) tea.Cmd
	View(width, height int) string
}

// Shared messages exchanged between AppModel and tab sub-models.
type (
	trafficMsg     mihomo.Traffic
	modeUpdatedMsg struct {
		newMode string
		err     error
	}
	modeSyncedMsg struct {
		mode string
	}
	statusMsg struct {
		text    string
		isError bool
	}
)

func statusCmd(text string, isError bool) tea.Cmd {
	return func() tea.Msg {
		return statusMsg{text: text, isError: isError}
	}
}

func modeSyncedCmd(mode string) tea.Cmd {
	return func() tea.Msg {
		return modeSyncedMsg{mode: mode}
	}
}

// AppModel is the main Bubble Tea container model routing events to independent TabPage sub-models.
type AppModel struct {
	client    mihomo.Client
	keys      KeyMap
	tabs      []TabPage
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
	keys := DefaultKeyMap()
	return &AppModel{
		client: cli,
		keys:   keys,
		tabs: []TabPage{
			NewOverviewModel(cli),
			NewProxiesModel(cli, keys),
			NewConfigsModel(cli, mgr, keys),
			NewSubscriptionsModel(mgr, keys),
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
		return trafficMsg(t)
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

	case trafficMsg:
		return m, tea.Batch(m.waitForTrafficCmd(), m.broadcast(msg))

	case statusMsg:
		m.statusMsg = msg.text
		m.isError = msg.isError
		return m, nil

	case modeSyncedMsg:
		m.mode = msg.mode
		return m, nil

	case modeUpdatedMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Update mode failed: %v", msg.err)
			m.isError = true
			return m, nil
		}
		m.mode = msg.newMode
		m.statusMsg = fmt.Sprintf("Mode switched to: %s", strings.ToUpper(msg.newMode))
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
	tabs := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	modeBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ColorDark).
		Background(theme.ColorSecondary).
		Padding(0, 1).
		Render(strings.ToUpper(m.mode))

	header := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", tabs, "  ", modeBadge)

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
