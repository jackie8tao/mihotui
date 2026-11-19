package keymaps

import "github.com/charmbracelet/bubbles/key"

// Global and page keybindings treated as read-only application-wide bindings.
var (
	Tab        = key.NewBinding(key.WithKeys("tab"))
	SwitchMode = key.NewBinding(key.WithKeys("m"))
	Refresh    = key.NewBinding(key.WithKeys("r"))
	Quit       = key.NewBinding(key.WithKeys("q"))
	Up         = key.NewBinding(key.WithKeys("up"))
	Down       = key.NewBinding(key.WithKeys("down"))
	Left       = key.NewBinding(key.WithKeys("left"))
	Right      = key.NewBinding(key.WithKeys("right"))
	Enter      = key.NewBinding(key.WithKeys("enter"))
	Esc        = key.NewBinding(key.WithKeys("esc"))
	TestDelay  = key.NewBinding(key.WithKeys("t"))
	TestAll    = key.NewBinding(key.WithKeys("T"))
	UpdateSub  = key.NewBinding(key.WithKeys("u"))
	UpdateAll  = key.NewBinding(key.WithKeys("U"))
	AddSub     = key.NewBinding(key.WithKeys("a"))
	DeleteSub  = key.NewBinding(key.WithKeys("d"))
)
