package app

import (
	"github.com/charmbracelet/bubbles/textinput"
)

// ProxiesTabState encapsulates the UI navigation and focus state for Tab 2 (Proxies).
type ProxiesTabState struct {
	FocusPane   FocusPane
	GroupCursor int
	NodeCursor  int
}

// ConfigsTabState encapsulates the list cursor state for Tab 3 (Configs).
type ConfigsTabState struct {
	Cursor int
}

// SubscriptionsTabState encapsulates the list cursor and modal state for Tab 4 (Subscriptions).
type SubscriptionsTabState struct {
	Cursor      int
	ModalActive ModalType
	NameInput   textinput.Model
	URLInput    textinput.Model
	InputFocus  int // 0: Name, 1: URL
}
