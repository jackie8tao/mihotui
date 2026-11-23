package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jackie8tao/mihotui/internal/app/events"
	"github.com/jackie8tao/mihotui/internal/app/keymaps"
	"github.com/jackie8tao/mihotui/internal/app/pages"
	"github.com/jackie8tao/mihotui/internal/app/theme"
	"github.com/jackie8tao/mihotui/internal/app/views"
	"github.com/jackie8tao/mihotui/internal/profile"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
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

// AppModel is the main Bubble Tea container model routing events to independent Page sub-models.
type AppModel struct {
	client     mihomo.Client
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

	return &AppModel{
		client: cli,
		pages: []Page{
			pages.NewOverviewModel(ctx, cli),
			pages.NewProxiesModel(cli),
			pages.NewConfigsModel(cli, mgr),
			pages.NewSubscriptionsModel(cli, mgr),
		},
		width:     80,
		height:    24,
		cancelCtx: cancel,
	}
}

// Init initializes all registered page sub-models.
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
		return events.ModeUpdatedMsg{NewMode: mode, Err: err}
	}
}

func (m *AppModel) handleGlobalKeys(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, keymaps.Quit):
		if m.cancelCtx != nil {
			m.cancelCtx()
		}
		return tea.Quit, true

	case key.Matches(msg, keymaps.Tab):
		if len(m.pages) > 0 {
			m.activePage = (m.activePage + 1) % len(m.pages)
		}
		return nil, true

	case key.Matches(msg, keymaps.Refresh):
		m.statusMsg = "Refreshing data..."
		m.isError = false
		var cmds []tea.Cmd
		for _, p := range m.pages {
			if cmd := p.Init(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return tea.Batch(cmds...), true

	case key.Matches(msg, keymaps.SwitchMode):
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

// Update routes Bubble Tea messages and keyboard events to global handlers and page sub-models.
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case events.StatusMsg:
		m.statusMsg = msg.Text
		m.isError = msg.IsError
		return m, nil

	case events.ModeSyncedMsg:
		m.mode = msg.Mode
		return m, m.broadcast(msg)

	case events.ModeUpdatedMsg:
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

// View renders the global header, active page content, and status/help footer.
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

	// Render single-line status message so long errors never wrap over or push out the help bar.
	statusText := strings.Join(strings.Fields(m.statusMsg), " ")
	if m.width > 2 {
		statusText = views.TruncateVisual(statusText, m.width-2)
	}
	statusColor := theme.ColorMuted
	if m.isError {
		statusColor = theme.ColorError
	}
	status := lipgloss.NewStyle().Foreground(statusColor).MaxHeight(1).Render(" " + statusText)

	var helpText string
	if active.CapturingInput() {
		helpText = active.Help()
	} else {
		helpText = "[Tab] Switch View | [m] Mode | [r] Refresh | [q] Quit"
		if pageHelp := active.Help(); pageHelp != "" {
			helpText = pageHelp + " | " + helpText
		}
	}
	helpStyle := lipgloss.NewStyle().Foreground(theme.ColorMuted)
	if m.width > 2 {
		helpStyle = helpStyle.Width(m.width - 1)
	}
	helpBar := helpStyle.Render(" " + helpText)

	footer := lipgloss.JoinVertical(lipgloss.Left, status, helpBar)

	// Allocate exact remaining height to the page body so header and footer (status + shortcuts)
	// are always pinned and never clipped or overwritten.
	bodyHeight := 16
	if m.height > 0 {
		reserved := lipgloss.Height(header) + lipgloss.Height(footer) + 2
		bodyHeight = m.height - reserved
		if bodyHeight < 6 {
			bodyHeight = 6
		}
	}

	body := active.View(m.width, bodyHeight)
	body = lipgloss.NewStyle().Height(bodyHeight).MaxHeight(bodyHeight).Render(body)

	return lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", footer)
}
