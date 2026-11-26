package theme

import "github.com/charmbracelet/lipgloss"

// Colors used across the TUI
var (
	ColorPrimary    = lipgloss.Color("#7D56F4") // Purple
	ColorSecondary  = lipgloss.Color("#04B575") // Green
	ColorHighlight  = lipgloss.Color("#FF79C6") // Pink
	ColorWarning    = lipgloss.Color("#FFB86C") // Orange/Amber
	ColorError      = lipgloss.Color("#FF5555") // Red
	ColorMuted      = lipgloss.Color("#6272A4") // Dim grey-blue
	ColorDark       = lipgloss.Color("#21222C") // Background dark
	ColorSelectedBg = lipgloss.Color("#44475A") // Active list row highlight background
	ColorLight      = lipgloss.Color("#F8F8F2") // Text white
)

// Common UI Styles
var (
	TitleStyle           = lipgloss.NewStyle().Bold(true).Foreground(ColorLight).Background(ColorPrimary).Padding(0, 1)
	TabActiveStyle       = lipgloss.NewStyle().Bold(true).Foreground(ColorLight).Background(ColorPrimary).Padding(0, 1)
	TabInactiveStyle     = lipgloss.NewStyle().Foreground(ColorMuted).Padding(0, 1)
	ModeBadgeStyle       = lipgloss.NewStyle().Bold(true).Foreground(ColorDark).Background(ColorSecondary).Padding(0, 1)
	CardBoxStyle         = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorMuted).Padding(0, 1)
	CardBoxActiveStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorPrimary).Padding(0, 1)
	ModalBoxStyle        = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(ColorPrimary).BorderBackground(ColorDark).Background(ColorDark).Padding(1, 2)
	SelectedRowStyle     = lipgloss.NewStyle().Bold(true).Foreground(ColorLight).Background(ColorSelectedBg)
	ActiveIndicatorStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary)
	SpeedUpStyle         = lipgloss.NewStyle().Bold(true).Foreground(ColorWarning)
	SpeedDownStyle       = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary)
)

// Common text styles shared across views.
var (
	TextStyle       = lipgloss.NewStyle().Foreground(ColorLight)
	BoldStyle       = lipgloss.NewStyle().Bold(true)
	MutedStyle      = lipgloss.NewStyle().Foreground(ColorMuted)
	HighlightStyle  = lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight)
	SuccessStyle    = lipgloss.NewStyle().Foreground(ColorSecondary)
	WarningStyle    = lipgloss.NewStyle().Foreground(ColorWarning)
	ErrorStyle      = lipgloss.NewStyle().Foreground(ColorError)
	ModalTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorLight)
)
