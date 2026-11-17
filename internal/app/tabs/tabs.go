package tabs

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

// Page defines the contract for an independent TUI tab sub-model.
type Page interface {
	Title() string
	Help() string
	CapturingInput() bool
	Init() tea.Cmd
	Update(msg tea.Msg) tea.Cmd
	View(width, height int) string
}

// Shared messages exchanged between the root AppModel and tab sub-models.
type (
	TrafficMsg     mihomo.Traffic
	ModeUpdatedMsg struct {
		NewMode string
		Err     error
	}
	ModeSyncedMsg struct {
		Mode string
	}
	StatusMsg struct {
		Text    string
		IsError bool
	}
	SubscriptionsChangedMsg struct{}
	ProfileAppliedEventMsg  struct{}
)

func statusCmd(text string, isError bool) tea.Cmd {
	return func() tea.Msg {
		return StatusMsg{Text: text, IsError: isError}
	}
}

func modeSyncedCmd(mode string) tea.Cmd {
	return func() tea.Msg {
		return ModeSyncedMsg{Mode: mode}
	}
}
