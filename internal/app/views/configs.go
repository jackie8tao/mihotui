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
	height := p.Height - 2
	if height < 6 {
		height = 6
	}

	leftWidth := 26
	rightWidth := width - leftWidth - 5
	if rightWidth < 30 {
		rightWidth = 30
	}

	// 1. Render Left Column: Configuration Sections
	var secLines []string
	for i, sec := range p.Sections {
		badge := theme.MutedStyle.Render(fmt.Sprintf("(%d)", sec.OverrideCount))
		if sec.OverrideCount > 0 {
			badge = theme.ActiveIndicatorStyle.Render(fmt.Sprintf("(%d)", sec.OverrideCount))
		}
		label := fmt.Sprintf("%-14s %s", TruncateVisual(sec.Title, 14), badge)

		var line string
		if i == p.SelectedSecIdx {
			if p.FocusPane == ConfigPaneSections {
				line = theme.SelectedRowStyle.Render("▶ " + label)
			} else {
				line = theme.HighlightStyle.Render("● " + label)
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
	leftHeader := theme.BoldStyle.Render("Sections")
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
		valText := TruncateVisual(f.DisplayValue, maxValWidth)
		var styledVal string
		if !f.IsOverridden {
			styledVal = theme.MutedStyle.Render(valText)
		} else if valText == "[ON]" {
			styledVal = theme.ActiveIndicatorStyle.Render(valText)
		} else {
			styledVal = theme.HighlightStyle.Render(valText)
		}

		row := fmt.Sprintf("%-20s %s", TruncateVisual(f.Label, 20), styledVal)
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
		rightLines = append(rightLines, theme.HighlightStyle.Render("─── Field Info ───"))
		rightLines = append(rightLines, fmt.Sprintf("Key  : %s  %s",
			theme.BoldStyle.Render(curr.YAMLPath),
			theme.MutedStyle.Render("["+curr.KindHint+"]"),
		))
		rightLines = append(rightLines, theme.TextStyle.Width(rightWidth-4).Render(curr.Description))
	}

	activeSubDisplay := theme.MutedStyle.Render("(none)")
	if p.ActiveSubName != "" {
		activeSubDisplay = theme.ActiveIndicatorStyle.Render(TruncateVisual(p.ActiveSubName, rightWidth-24))
	}
	rightLines = append(rightLines, "")
	rightLines = append(rightLines, theme.MutedStyle.Render(
		fmt.Sprintf("Active Sub: %s | Override: %s", activeSubDisplay, TruncateVisual(p.OverridePath, rightWidth-36)),
	))

	rightBoxStyle := theme.CardBoxStyle
	if p.FocusPane == ConfigPaneFields {
		rightBoxStyle = theme.CardBoxActiveStyle
	}
	rightHeader := theme.BoldStyle.Render(p.ActiveSecTitle)
	if p.Dirty {
		rightHeader += " " + theme.WarningStyle.Bold(true).Render("[Modified]")
	}
	rightPanel := rightBoxStyle.Width(rightWidth).Height(height).Render(
		rightHeader + "\n\n" + strings.Join(rightLines, "\n"),
	)

	baseView := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
	if p.Editing {
		return renderConfigEditModal(p, baseView)
	}
	return baseView
}

// renderConfigEditModal renders the centered edit dialog for the currently selected field over baseView.
func renderConfigEditModal(p ConfigsProps, baseView string) string {
	if p.SelectedFieldIdx < 0 || p.SelectedFieldIdx >= len(p.Fields) {
		return baseView
	}
	curr := p.Fields[p.SelectedFieldIdx]

	modalWidth := clampModalWidth(58, p.Width)
	descWidth := modalWidth - 6
	if descWidth < 16 {
		descWidth = 16
	}

	title := theme.ModalTitleStyle.Render(
		fmt.Sprintf("Edit %s (%s)", curr.Label, curr.YAMLPath),
	)
	desc := theme.MutedStyle.Width(descWidth).Render(curr.Description)
	inputField := fmt.Sprintf("Value:\n%s", p.EditInputView)
	help := theme.MutedStyle.Render("[Enter] Confirm | [Esc] Cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, "\n", desc, "\n", inputField, "\n", help)
	return renderModalOverlay(baseView, p.Width, p.Height, modalWidth, content)
}
