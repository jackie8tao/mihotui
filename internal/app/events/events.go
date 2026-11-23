package events

import tea "github.com/charmbracelet/bubbletea"

// Cross-component and inter-page event messages routed through AppModel.
type (
	// StatusMsg reports status bar updates to the root AppModel container.
	StatusMsg struct {
		Text    string
		IsError bool
	}
	// ModeSyncedMsg synchronizes the global outbound mode badge from the core.
	ModeSyncedMsg struct {
		Mode string
	}
	// ModeUpdatedMsg carries the result of an interactive outbound mode switch.
	ModeUpdatedMsg struct {
		NewMode string
		Err     error
	}
	// ProfileAppliedMsg notifies pages that a configuration profile was hot-reloaded.
	ProfileAppliedMsg struct{}
)

// StatusCmd returns a command that emits a StatusMsg.
func StatusCmd(text string, isError bool) tea.Cmd {
	return func() tea.Msg {
		return StatusMsg{Text: text, IsError: isError}
	}
}

// ModeSyncedCmd returns a command that emits a ModeSyncedMsg.
func ModeSyncedCmd(mode string) tea.Cmd {
	return func() tea.Msg {
		return ModeSyncedMsg{Mode: mode}
	}
}

// ProfileAppliedCmd returns a command that emits a ProfileAppliedMsg.
func ProfileAppliedCmd() tea.Cmd {
	return func() tea.Msg {
		return ProfileAppliedMsg{}
	}
}
