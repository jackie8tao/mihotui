package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jackie8tao/mihotui/internal/app/components"
	"github.com/jackie8tao/mihotui/internal/app/theme"
)

const configsLeftPaneWidth = 28

// RenderConfigs renders the schema-driven override.yaml system configuration view.
func RenderConfigs(p ConfigsProps) string {
	baseView := components.SplitPane(p.Width, p.Height, configsLeftPaneWidth,
		func(w, h int) string {
			return renderConfigSectionsPane(p, w, h)
		},
		func(w, h int) string {
			return renderConfigFieldsPane(p, w, h)
		},
	)

	if p.Editing {
		return renderConfigEditModal(p, baseView)
	}
	return baseView
}

// renderConfigSectionsPane renders the left sidebar listing configuration categories.
func renderConfigSectionsPane(p ConfigsProps, width, height int) string {
	return components.Panel(components.PanelProps{
		Title:  "Sections",
		Active: p.FocusPane == ConfigPaneSections,
		Width:  width,
		Height: height,
	}, func(cw, ch int) string {
		return components.ScrollList(components.ListProps{
			Total:        len(p.Sections),
			SelectedIdx:  p.SelectedSecIdx,
			Focused:      p.FocusPane == ConfigPaneSections,
			EmptyMessage: "No sections available",
			RenderRow: func(idx, rowW int, selected, focused bool) []components.Segment {
				sec := p.Sections[idx]
				badgeText := fmt.Sprintf("(%d)", sec.OverrideCount)
				badgeStyle := theme.MutedStyle
				if sec.OverrideCount > 0 {
					badgeStyle = theme.ActiveIndicatorStyle
				}

				titleW := rowW - 2 - len(badgeText) - 1
				if titleW < 4 {
					titleW = 4
				}
				paddedTitle := components.PadRightVisual(sec.Title, titleW) + " "

				if selected && !focused {
					return []components.Segment{
						components.Styled("● "+paddedTitle, theme.HighlightStyle),
						components.Styled(badgeText, badgeStyle),
					}
				}
				cursor := "  "
				if selected {
					cursor = "▶ "
				}
				return []components.Segment{
					components.Plain(cursor + paddedTitle),
					components.Styled(badgeText, badgeStyle),
				}
			},
		}, cw, ch)
	})
}

// renderConfigFieldsPane renders the right pane with scrollable fields, selected field info, and footer.
func renderConfigFieldsPane(p ConfigsProps, width, height int) string {
	badge := ""
	if p.Dirty {
		badge = theme.WarningStyle.Bold(true).Render("[Modified]")
	}

	return components.Panel(components.PanelProps{
		Title:  p.ActiveSecTitle,
		Badge:  badge,
		Active: p.FocusPane == ConfigPaneFields,
		Width:  width,
		Height: height,
	}, func(cw, ch int) string {
		footerLine := renderConfigFooterLine(p, cw)
		infoBlock := renderConfigFieldInfo(p, cw)

		// Reserve space for the bottom footer and optional Field Info block,
		// giving all remaining rows to the scrollable fields list.
		reserved := 2 // blank line + footerLine
		showInfo := false
		if infoBlock != "" {
			infoH := lipgloss.Height(infoBlock) + 1 // blank line + infoBlock
			if ch-reserved-infoH >= 2 {
				reserved += infoH
				showInfo = true
			}
		}

		listH := ch - reserved
		if listH < 1 {
			listH = 1
		}

		fieldsList := components.ScrollList(components.ListProps{
			Total:        len(p.Fields),
			SelectedIdx:  p.SelectedFieldIdx,
			Focused:      p.FocusPane == ConfigPaneFields,
			EmptyMessage: "No fields in this section",
			RenderRow: func(idx, rowW int, selected, focused bool) []components.Segment {
				return renderConfigFieldRow(p.Fields[idx], rowW, selected && focused)
			},
		}, cw, listH)

		parts := []string{fieldsList}
		if showInfo {
			parts = append(parts, "", infoBlock)
		}
		parts = append(parts, "", footerLine)
		return strings.Join(parts, "\n")
	})
}

// renderConfigFieldRow builds an aligned label + value row for a single configuration field.
func renderConfigFieldRow(f ConfigFieldView, rowW int, activeCursor bool) []components.Segment {
	cursor := "  "
	if activeCursor {
		cursor = "▶ "
	}

	labelW := 20
	if maxLabel := rowW/2 - 2; maxLabel < labelW {
		labelW = maxLabel
	}
	if labelW < 6 {
		labelW = 6
	}

	valW := rowW - 2 - labelW - 1
	if valW < 4 {
		valW = 4
	}

	valText := components.TruncateVisual(f.DisplayValue, valW)
	valStyle := theme.HighlightStyle
	switch {
	case !f.IsOverridden:
		valStyle = theme.MutedStyle
	case f.IsActive:
		valStyle = theme.ActiveIndicatorStyle
	}

	return []components.Segment{
		components.Plain(cursor + components.PadRightVisual(f.Label, labelW) + " "),
		components.Styled(valText, valStyle),
	}
}

// renderConfigFieldInfo renders metadata and description for the currently selected field.
func renderConfigFieldInfo(p ConfigsProps, width int) string {
	if p.SelectedFieldIdx < 0 || p.SelectedFieldIdx >= len(p.Fields) {
		return ""
	}
	curr := p.Fields[p.SelectedFieldIdx]

	divider := components.SectionDivider("Field Info", theme.HighlightStyle, width)
	keyLine := components.KVList([]components.KVItem{
		{
			Key: "Key",
			Segments: []components.Segment{
				components.Styled(components.SanitizeDisplayText(curr.YAMLPath), theme.BoldStyle),
				components.Plain("  "),
				components.Styled("["+components.SanitizeDisplayText(curr.KindHint)+"]", theme.MutedStyle),
			},
		},
	}, width)
	desc := theme.TextStyle.Width(max(8, width)).Render(components.SanitizeDisplayText(curr.Description))

	return strings.Join([]string{divider, keyLine, desc}, "\n")
}

// renderConfigFooterLine renders the single-line active subscription and override path summary.
func renderConfigFooterLine(p ConfigsProps, width int) string {
	const prefix = "Active Sub: "
	const mid = " | Override: "

	subName := "(none)"
	subStyle := theme.MutedStyle
	if p.ActiveSubName != "" {
		subMax := max(8, width/3)
		subName = components.TruncateVisual(p.ActiveSubName, subMax)
		subStyle = theme.ActiveIndicatorStyle
	}

	usedW := len(prefix) + lipgloss.Width(subName) + len(mid)
	pathMax := width - usedW
	segs := []components.Segment{
		components.Styled(prefix, theme.MutedStyle),
		components.Styled(subName, subStyle),
	}
	if pathMax >= 6 {
		segs = append(segs,
			components.Styled(mid+components.TruncateVisual(p.OverridePath, pathMax), theme.MutedStyle),
		)
	}
	return components.RenderRowSegments(segs, width, false)
}

// renderConfigEditModal renders the centered edit dialog for the currently selected field over baseView.
func renderConfigEditModal(p ConfigsProps, baseView string) string {
	if p.SelectedFieldIdx < 0 || p.SelectedFieldIdx >= len(p.Fields) {
		return baseView
	}
	curr := p.Fields[p.SelectedFieldIdx]

	modalW := components.ClampModalWidth(58, p.Width)
	descW := max(16, modalW-4)
	desc := theme.MutedStyle.Width(descW).Render(components.SanitizeDisplayText(curr.Description))
	body := lipgloss.JoinVertical(lipgloss.Left, desc, "", "Value:", p.EditInputView)

	return components.ModalOverlay(baseView, p.Width, p.Height, components.ModalProps{
		Title:          fmt.Sprintf("Edit %s (%s)", curr.Label, curr.YAMLPath),
		Body:           body,
		Footer:         "[Enter] Confirm | [Esc] Cancel",
		PreferredWidth: 58,
	})
}
