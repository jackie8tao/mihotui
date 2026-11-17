package app

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines global keybindings handled by the root AppModel.
type KeyMap struct {
	Tab        key.Binding
	SwitchMode key.Binding
	Refresh    key.Binding
	Quit       key.Binding
}

// DefaultKeyMap returns the default global keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Tab:        key.NewBinding(key.WithKeys("tab")),
		SwitchMode: key.NewBinding(key.WithKeys("m")),
		Refresh:    key.NewBinding(key.WithKeys("r")),
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c")),
	}
}
