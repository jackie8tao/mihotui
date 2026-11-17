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
	Quit       key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:         key.NewBinding(key.WithKeys("up", "k")),
		Down:       key.NewBinding(key.WithKeys("down", "j")),
		Left:       key.NewBinding(key.WithKeys("left", "h")),
		Right:      key.NewBinding(key.WithKeys("right", "l")),
		Tab:        key.NewBinding(key.WithKeys("tab")),
		Enter:      key.NewBinding(key.WithKeys("enter")),
		TestDelay:  key.NewBinding(key.WithKeys("t")),
		TestAll:    key.NewBinding(key.WithKeys("T")),
		SwitchMode: key.NewBinding(key.WithKeys("m")),
		Refresh:    key.NewBinding(key.WithKeys("r")),
		UpdateSub:  key.NewBinding(key.WithKeys("u")),
		UpdateAll:  key.NewBinding(key.WithKeys("U")),
		AddSub:     key.NewBinding(key.WithKeys("a")),
		DeleteSub:  key.NewBinding(key.WithKeys("d")),
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c")),
	}
}
