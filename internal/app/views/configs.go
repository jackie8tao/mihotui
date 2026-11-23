package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jackie8tao/mihotui/internal/app/theme"
)

// RenderConfigs renders the schema-driven override.yaml system configuration view.
func RenderConfigs(p ConfigsProps) string {
	width := p.Width - 4
	if width < 56 {
		width = 56
	}
	height := p.Height - 8
	if height < 12 {
		height = 12
	}

	leftWidth := 26
	rightWidth := width - leftWidth - 5
	if rightWidth < 30 {
		rightWidth = 30
	}

	// 1. Render Left Column: Configuration Sections
	var secLines []string
	for i, sec := range p.Sections {
		badge := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(fmt.Sprintf("(%d)", sec.OverrideCount))
		if sec.OverrideCount > 0 {
			badge = theme.ActiveIndicatorStyle.Render(fmt.Sprintf("(%d)", sec.OverrideCount))
		}
		label := fmt.Sprintf("%-14s %s", truncateStr(sec.Title, 14), badge)

		var line string
		if i == p.SelectedSecIdx {
			if p.FocusPane == ConfigPaneSections {
				line = theme.SelectedRowStyle.Render("▶ " + label)
			} else {
				line = lipgloss.NewStyle().Bold(true).Foreground(theme.ColorHighlight).Render("● " + label)
			}
		} else {
			line = "  " + label
		}
		secLines = append(secLines, line)
	}

	leftBoxStyle := theme.CardBoxStyle
	if p.FocusPane == ConfigPaneSections {
		leftBoxStyle = theme.CardBoxActiveStyle
	}
	leftHeader := lipgloss.NewStyle().Bold(true).Render("Sections")
	leftPanel := leftBoxStyle.Width(leftWidth).Height(height).Render(
		leftHeader + "\n\n" + strings.Join(secLines, "\n"),
	)

	// 2. Render Right Column: Fields List, Selected Field Info, and Live YAML Preview
	var rightLines []string
	maxValWidth := rightWidth - 26
	if maxValWidth < 12 {
		maxValWidth = 12
	}

	for i, f := range p.Fields {
		valText := truncateStr(f.DisplayValue, maxValWidth)
		var styledVal string
		if !f.IsOverridden {
			styledVal = lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(valText)
		} else if valText == "[ON]" {
			styledVal = theme.ActiveIndicatorStyle.Render(valText)
		} else {
			styledVal = lipgloss.NewStyle().Bold(true).Foreground(theme.ColorHighlight).Render(valText)
		}

		row := fmt.Sprintf("%-20s %s", truncateStr(f.Label, 20), styledVal)
		if i == p.SelectedFieldIdx && p.FocusPane == ConfigPaneFields {
			row = theme.SelectedRowStyle.Render("▶ " + row)
		} else {
			row = "  " + row
		}
		rightLines = append(rightLines, row)
	}

	if p.SelectedFieldIdx >= 0 && p.SelectedFieldIdx < len(p.Fields) {
		curr := p.Fields[p.SelectedFieldIdx]
		rightLines = append(rightLines, "")
		rightLines = append(rightLines, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorHighlight).Render("─── Field Info ───"))
		rightLines = append(rightLines, fmt.Sprintf("Key  : %s  %s",
			lipgloss.NewStyle().Bold(true).Render(curr.YAMLPath),
			lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("["+curr.KindHint+"]"),
		))
		rightLines = append(rightLines, lipgloss.NewStyle().Width(rightWidth-4).Foreground(theme.ColorLight).Render(curr.Description))
	}

	activeSubDisplay := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("(none)")
	if p.ActiveSubName != "" {
		activeSubDisplay = theme.ActiveIndicatorStyle.Render(truncateStr(p.ActiveSubName, rightWidth-24))
	}
	rightLines = append(rightLines, "")
	rightLines = append(rightLines, lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(
		fmt.Sprintf("Active Sub: %s | Override: %s", activeSubDisplay, truncateStr(p.OverridePath, rightWidth-36)),
	))

	rightBoxStyle := theme.CardBoxStyle
	if p.FocusPane == ConfigPaneFields {
		rightBoxStyle = theme.CardBoxActiveStyle
	}
	rightHeader := lipgloss.NewStyle().Bold(true).Render(p.ActiveSecTitle)
	if p.Dirty {
		rightHeader += " " + lipgloss.NewStyle().Foreground(theme.ColorWarning).Bold(true).Render("[Modified]")
	}
	rightPanel := rightBoxStyle.Width(rightWidth).Height(height).Render(
		rightHeader + "\n\n" + strings.Join(rightLines, "\n"),
	)

	baseView := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
	if p.Editing {
		return renderConfigEditModal(p)
	}
	return baseView
}

func renderConfigEditModal(p ConfigsProps) string {
	modalWidth := 58
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.ColorPrimary).
		Background(theme.ColorDark).
		Padding(1, 2).
		Width(modalWidth)

	title := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorLight).Render(
		fmt.Sprintf("Edit %s (%s)", p.EditLabel, p.EditYAMLPath),
	)
	desc := lipgloss.NewStyle().Width(modalWidth - 6).Foreground(theme.ColorMuted).Render(p.EditDescription)
	inputField := fmt.Sprintf("Value:\n%s", p.EditInputView)
	help := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("[Enter] Confirm | [Esc] Cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, "\n", desc, "\n", inputField, "\n", help)
	modal := modalStyle.Render(content)

	return lipgloss.Place(p.Width, p.Height-4, lipgloss.Center, lipgloss.Center, modal)
}
