package pages

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Inter-page events routed through the host container between page sub-models.
type profileAppliedEventMsg struct{}

// StatusCmdFunc produces a command to display status text in the host container.
type StatusCmdFunc func(text string, isError bool) tea.Cmd

// ModeSyncCmdFunc produces a command to synchronize the outbound mode badge in the host container.
type ModeSyncCmdFunc func(mode string) tea.Cmd

func emitStatus(fn StatusCmdFunc, text string, isError bool) tea.Cmd {
	if fn == nil {
		return nil
	}
	return fn(text, isError)
}

func emitModeSync(fn ModeSyncCmdFunc, mode string) tea.Cmd {
	if fn == nil {
		return nil
	}
	return fn(mode)
}
