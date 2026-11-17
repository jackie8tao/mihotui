package pages

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Shared messages exchanged between the root AppModel and page sub-models.
type (
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
