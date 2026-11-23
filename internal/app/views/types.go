package views

import (
	"github.com/jackie8tao/mihotui/internal/config"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

// FocusPane represents which pane has focus in a two-column view.
type FocusPane int

const (
	PaneGroups FocusPane = iota
	PaneNodes
)

// ConfigFocusPane represents which pane has focus on the Configs tab.
type ConfigFocusPane int

const (
	ConfigPaneSections ConfigFocusPane = iota
	ConfigPaneFields
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
	Width            int
	Height           int
	FocusPane        FocusPane
	Groups           []mihomo.ProxyItem
	Proxies          map[string]mihomo.ProxyItem
	SelectedGroupIdx int
	SelectedNodeIdx  int
}

// ConfigSectionView represents a single section entry in the Configs sidebar.
type ConfigSectionView struct {
	Title         string
	OverrideCount int
}

// ConfigFieldView represents a single field entry in the active Configs section.
type ConfigFieldView struct {
	Label        string
	YAMLPath     string
	Description  string
	KindHint     string
	DisplayValue string
	IsOverridden bool
}

// ConfigsProps contains the data needed to render the configs (override.yaml) tab.
type ConfigsProps struct {
	Width            int
	Height           int
	FocusPane        ConfigFocusPane
	Sections         []ConfigSectionView
	SelectedSecIdx   int
	ActiveSecTitle   string
	Fields           []ConfigFieldView
	SelectedFieldIdx int
	OverridePath     string
	MihomoConfigPath string
	ActiveSubName    string
	Dirty            bool
	Editing          bool
	EditLabel        string
	EditYAMLPath     string
	EditDescription  string
	EditInputView    string
}

// SubscriptionsProps contains the data needed to render the subscriptions tab.
type SubscriptionsProps struct {
	Width            int
	Height           int
	Subscriptions    []config.Subscription
	ActiveSubName    string
	SelectedIdx      int
	ModalActive      ModalType
	SubNameInputView string
	SubURLInputView  string
}

