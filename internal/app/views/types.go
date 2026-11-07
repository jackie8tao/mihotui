package views

import (
	"mihomo-tui/internal/profile"
	"mihomo-tui/pkg/mihomo"
)

// FocusPane represents which pane has focus in a two-column view.
type FocusPane int

const (
	PaneGroups FocusPane = iota
	PaneNodes
)

// ModalType represents currently active modal dialogue.
type ModalType int

const (
	ModalNone ModalType = iota
	ModalAddSub
)

// OverviewProps contains the data needed to render the overview tab.
type OverviewProps struct {
	Width     int
	Traffic   mihomo.Traffic
	TotalUp   int64
	TotalDown int64
	Config    mihomo.Config
	Version   mihomo.Version
	Memory    mihomo.Memory
}

// ProxiesProps contains the data needed to render the proxies tab.
type ProxiesProps struct {
	Width       int
	Height      int
	FocusPane   FocusPane
	Groups      []mihomo.ProxyItem
	Proxies     map[string]mihomo.ProxyItem
	GroupCursor int
	NodeCursor  int
}

// ConfigsProps contains the data needed to render the configs & subscriptions tab.
type ConfigsProps struct {
	Width            int
	Height           int
	ConfigDir        string
	Profiles         []profile.Profile
	ProfileCursor    int
	ModalActive      ModalType
	SubNameInputView string
	SubURLInputView  string
}

