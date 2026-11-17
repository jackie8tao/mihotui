package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jackie8tao/mihotui/internal/app/pages"
	"github.com/jackie8tao/mihotui/internal/app/theme"
	"github.com/jackie8tao/mihotui/internal/profile"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

// Global messages handled by the root AppModel container.
type (
	// StatusMsg reports status bar updates to AppModel.
	StatusMsg struct {
		Text    string
		IsError bool
	}
	// ModeSyncedMsg synchronizes the global outbound mode badge from the core.
	ModeSyncedMsg struct {
		Mode string
	}
	// ModeUpdatedMsg carries the result of an interactive mode switch.
	ModeUpdatedMsg struct {
		NewMode string
		Err     error
	}
)

// Page defines the contract for an independent TUI page sub-model consumed by AppModel.
type Page interface {
	Title() string
	Help() string
	CapturingInput() bool
	Init() tea.Cmd
	Update(msg tea.Msg) tea.Cmd
	View(width, height int) string
}

type modeSetter interface {
	SetMode(mode string)
}

// AppModel is the main Bubble Tea container model routing events to independent Page sub-models.
type AppModel struct {
	client     mihomo.Client
	keys       KeyMap
	pages      []Page
	activePage int

	width  int
	height int
	mode   string

	cancelCtx context.CancelFunc

	statusMsg string
	isError   bool
}

// New initializes and returns a new AppModel.
func New(cli mihomo.Client, mgr *profile.Manager) *AppModel {
	ctx, cancel := context.WithCancel(context.Background())

	reportStatus := func(text string, isError bool) tea.Cmd {
		return func() tea.Msg {
			return StatusMsg{Text: text, IsError: isError}
		}
	}
	syncMode := func(mode string) tea.Cmd {
		return func() tea.Msg {
			return ModeSyncedMsg{Mode: mode}
		}
	}

	return &AppModel{
		client: cli,
		keys:   DefaultKeyMap(),
		pages: []Page{
			pages.NewOverviewModel(ctx, cli, reportStatus, syncMode),
			pages.NewProxiesModel(cli, reportStatus),
			pages.NewConfigsModel(cli, mgr, reportStatus),
			pages.NewSubscriptionsModel(mgr, reportStatus),
		},
		width:     80,
		height:    24,
		cancelCtx: cancel,
	}
}

func (m *AppModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, p := range m.pages {
		if cmd := p.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (m *AppModel) updateModeCmd(mode string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := m.client.UpdateMode(ctx, mode)
		return ModeUpdatedMsg{NewMode: mode, Err: err}
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
		if len(m.pages) > 0 {
			m.activePage = (m.activePage + 1) % len(m.pages)
		}
		return nil, true

	case key.Matches(msg, m.keys.Refresh):
		m.statusMsg = "Refreshing data..."
		m.isError = false
		var cmds []tea.Cmd
		for _, p := range m.pages {
			if cmd := p.Init(); cmd != nil {
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
	for _, p := range m.pages {
		if cmd := p.Update(msg); cmd != nil {
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

	case StatusMsg:
		m.statusMsg = msg.Text
		m.isError = msg.IsError
		return m, nil

	case ModeSyncedMsg:
		m.mode = msg.Mode
		return m, nil

	case ModeUpdatedMsg:
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("Update mode failed: %v", msg.Err)
			m.isError = true
			return m, nil
		}
		m.mode = msg.NewMode
		m.statusMsg = fmt.Sprintf("Mode switched to: %s", strings.ToUpper(msg.NewMode))
		m.isError = false
		for _, p := range m.pages {
			if setter, ok := p.(modeSetter); ok {
				setter.SetMode(msg.NewMode)
			}
		}
		return m, nil

	case tea.KeyMsg:
		if len(m.pages) == 0 {
			return m, nil
		}
		active := m.pages[m.activePage]
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
	if len(m.pages) == 0 {
		return ""
	}

	title := theme.TitleStyle.Render(" MihoTui ")

	var renderedTabs []string
	for i, p := range m.pages {
		if i == m.activePage {
			renderedTabs = append(renderedTabs, theme.TabActiveStyle.Render(p.Title()))
		} else {
			renderedTabs = append(renderedTabs, theme.TabInactiveStyle.Render(p.Title()))
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

	active := m.pages[m.activePage]
	body := active.View(m.width, m.height)

	statusColor := theme.ColorMuted
	if m.isError {
		statusColor = theme.ColorError
	}
	status := lipgloss.NewStyle().Foreground(statusColor).Render(" " + m.statusMsg)

	helpText := "[Tab] Switch View | [m] Mode | [r] Refresh | [q] Quit"
	if pageHelp := active.Help(); pageHelp != "" {
		helpText = pageHelp + " | " + helpText
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
