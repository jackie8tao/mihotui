package app

import (
	"github.com/jackie8tao/mihotui/internal/app/views"
)

// ProxiesTabState encapsulates the UI navigation and focus state for Tab 2 (Proxies).
type ProxiesTabState struct {
	FocusPane   views.FocusPane
	GroupCursor int
	NodeCursor  int
}

// ConfigsTabState encapsulates the list cursor state for Tab 3 (Configs).
type ConfigsTabState struct {
	Cursor int
}
