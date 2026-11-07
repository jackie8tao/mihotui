package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines keybindings for the application.
type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	Tab        key.Binding
	Enter      key.Binding
	TestDelay  key.Binding
	TestAll    key.Binding
	SwitchMode key.Binding
	Refresh    key.Binding
	UpdateSub  key.Binding
	UpdateAll  key.Binding
	AddSub     key.Binding
	DeleteSub  key.Binding
	Esc        key.Binding
	Quit       key.Binding
	Help       key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "group left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "node right"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch tab"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select / apply"),
		),
		TestDelay: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "test delay"),
		),
		TestAll: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "test all in group"),
		),
		SwitchMode: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "switch mode"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh data"),
		),
		UpdateSub: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "update sub"),
		),
		UpdateAll: key.NewBinding(
			key.WithKeys("U"),
			key.WithHelp("U", "update all subs"),
		),
		AddSub: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add subscription"),
		),
		DeleteSub: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete subscription"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel / close"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

