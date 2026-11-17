package pages

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Enter     key.Binding
	TestDelay key.Binding
	TestAll   key.Binding
	UpdateSub key.Binding
	UpdateAll key.Binding
	AddSub    key.Binding
	DeleteSub key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up:        key.NewBinding(key.WithKeys("up", "k")),
		Down:      key.NewBinding(key.WithKeys("down", "j")),
		Left:      key.NewBinding(key.WithKeys("left", "h")),
		Right:     key.NewBinding(key.WithKeys("right", "l")),
		Enter:     key.NewBinding(key.WithKeys("enter")),
		TestDelay: key.NewBinding(key.WithKeys("t")),
		TestAll:   key.NewBinding(key.WithKeys("T")),
		UpdateSub: key.NewBinding(key.WithKeys("u")),
		UpdateAll: key.NewBinding(key.WithKeys("U")),
		AddSub:    key.NewBinding(key.WithKeys("a")),
		DeleteSub: key.NewBinding(key.WithKeys("d")),
	}
}
