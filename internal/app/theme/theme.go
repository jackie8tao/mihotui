package theme

import "github.com/charmbracelet/lipgloss"

// Colors used across the TUI
var (
	ColorPrimary   = lipgloss.Color("#7D56F4") // Purple
	ColorSecondary = lipgloss.Color("#04B575") // Green
	ColorHighlight = lipgloss.Color("#FF79C6") // Pink
	ColorWarning   = lipgloss.Color("#FFB86C") // Orange/Amber
	ColorError     = lipgloss.Color("#FF5555") // Red
	ColorMuted     = lipgloss.Color("#6272A4") // Dim grey-blue
	ColorDark      = lipgloss.Color("#21222C") // Background dark
	ColorLight     = lipgloss.Color("#F8F8F2") // Text white
)

// Common UI Styles
var (
	TitleStyle           = lipgloss.NewStyle().Bold(true).Foreground(ColorLight).Background(ColorPrimary).Padding(0, 1)
	TabActiveStyle       = lipgloss.NewStyle().Bold(true).Foreground(ColorLight).Background(ColorPrimary).Padding(0, 1)
	TabInactiveStyle     = lipgloss.NewStyle().Foreground(ColorMuted).Padding(0, 1)
	StatusBarStyle       = lipgloss.NewStyle().Foreground(ColorMuted).Background(lipgloss.Color("#191A21")).Padding(0, 1)
	CardBoxStyle         = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorMuted).Padding(0, 1)
	CardBoxActiveStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorPrimary).Padding(0, 1)
	SelectedRowStyle     = lipgloss.NewStyle().Bold(true).Foreground(ColorLight).Background(lipgloss.Color("#44475A"))
	NormalRowStyle       = lipgloss.NewStyle().Foreground(ColorLight)
	ActiveIndicatorStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary)
	SpeedUpStyle         = lipgloss.NewStyle().Bold(true).Foreground(ColorWarning)
	SpeedDownStyle       = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary)
)
