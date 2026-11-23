package views

import (
	"fmt"
	"strings"

	"github.com/jackie8tao/mihotui/internal/app/theme"
	"github.com/jackie8tao/mihotui/pkg/mihomo"

	"github.com/charmbracelet/lipgloss"
)

// RenderProxies renders the proxies and node groups view.
func RenderProxies(p ProxiesProps) string {
	width := p.Width - 4
	if width < 50 {
		width = 50
	}
	height := p.Height - 2
	if height < 6 {
		height = 6
	}

	leftWidth := 28
	rightWidth := width - leftWidth - 5
	if rightWidth < 25 {
		rightWidth = 25
	}

	visibleRows := height - 2
	if visibleRows < 3 {
		visibleRows = 3
	}

	// 1. Render Left Column: Groups List (with visible window scrolling)
	var groupLines []string
	if len(p.Groups) == 0 {
		groupLines = append(groupLines, lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("No proxy groups found"))
	} else {
		startGroup := 0
		if p.SelectedGroupIdx >= visibleRows {
			startGroup = p.SelectedGroupIdx - visibleRows + 1
		}
		endGroup := startGroup + visibleRows
		if endGroup > len(p.Groups) {
			endGroup = len(p.Groups)
		}

		for i := startGroup; i < endGroup; i++ {
			g := p.Groups[i]
			prefix := fmt.Sprintf("[%d] ", len(g.All))
			nameMax := leftWidth - 4 - len(prefix)
			if nameMax < 4 {
				nameMax = 4
			}
			line := prefix + truncateStr(g.Name, nameMax)
			if i == p.SelectedGroupIdx {
				if p.FocusPane == PaneGroups {
					line = theme.SelectedRowStyle.Render("▶ " + line)
				} else {
					line = lipgloss.NewStyle().Bold(true).Foreground(theme.ColorHighlight).Render("● " + line)
				}
			} else {
				line = "  " + line
			}
			groupLines = append(groupLines, line)
		}
	}

	leftBoxStyle := theme.CardBoxStyle
	if p.FocusPane == PaneGroups {
		leftBoxStyle = theme.CardBoxActiveStyle
	}
	leftHeader := lipgloss.NewStyle().Bold(true).Render(TruncateVisual("Proxy Groups", leftWidth-2))
	leftPanel := leftBoxStyle.Width(leftWidth).Height(height).Render(
		leftHeader + "\n\n" + strings.Join(groupLines, "\n"),
	)

	// 2. Render Right Column: Nodes in selected Group
	var currGroup *mihomo.ProxyItem
	if len(p.Groups) > 0 && p.SelectedGroupIdx >= 0 && p.SelectedGroupIdx < len(p.Groups) {
		currGroup = &p.Groups[p.SelectedGroupIdx]
	}

	var rightLines []string
	if currGroup == nil || len(currGroup.All) == 0 {
		rightLines = append(rightLines, lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("No nodes in this group"))
	} else {
		startIdx := 0
		if p.SelectedNodeIdx >= visibleRows {
			startIdx = p.SelectedNodeIdx - visibleRows + 1
		}
		endIdx := startIdx + visibleRows
		if endIdx > len(currGroup.All) {
			endIdx = len(currGroup.All)
		}

		for i := startIdx; i < endIdx; i++ {
			nodeName := currGroup.All[i]
			nodeItem, exists := p.Proxies[nodeName]

			// Active mark
			activeMark := "  "
			if nodeName == currGroup.Now {
				activeMark = theme.ActiveIndicatorStyle.Render("✔ ")
			}

			// Delay text and styling (fixed 10-column visual width)
			delayStr := fmt.Sprintf("%10s", "-")
			if exists {
				if nodeItem.TestingDelay {
					delayStr = lipgloss.NewStyle().Foreground(theme.ColorWarning).Render(fmt.Sprintf("%10s", "Testing..."))
				} else if len(nodeItem.History) > 0 {
					lastDelay := nodeItem.History[len(nodeItem.History)-1].Delay
					delayFormatted := fmt.Sprintf("%10s", mihomo.FormatDelay(lastDelay))
					switch {
					case lastDelay <= 0:
						delayStr = lipgloss.NewStyle().Foreground(theme.ColorError).Render(delayFormatted)
					case lastDelay < 120:
						delayStr = lipgloss.NewStyle().Foreground(theme.ColorSecondary).Render(delayFormatted)
					case lastDelay < 300:
						delayStr = lipgloss.NewStyle().Foreground(theme.ColorWarning).Render(delayFormatted)
					default:
						delayStr = lipgloss.NewStyle().Foreground(theme.ColorError).Render(delayFormatted)
					}
				}
			}

			// Type text (pad before styling so ANSI escapes do not break column width)
			nodeType := ""
			if exists && nodeItem.Type != "" {
				nodeType = fmt.Sprintf("[%s]", nodeItem.Type)
			}
			styledType := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(padRightVisual(nodeType, 14))

			// Node name visual-width truncation & padding (handles CJK & Emoji accurately)
			availNameWidth := rightWidth - 32
			if availNameWidth < 12 {
				availNameWidth = 12
			}
			paddedName := padRightVisual(nodeName, availNameWidth)

			cursor := "  "
			isSelected := i == p.SelectedNodeIdx && p.FocusPane == PaneNodes
			if isSelected {
				cursor = "▶ "
			}

			row := fmt.Sprintf("%s%s%s %s %s", cursor, activeMark, paddedName, styledType, delayStr)
			if isSelected {
				row = theme.SelectedRowStyle.Render(row)
			}

			rightLines = append(rightLines, row)
		}
	}

	rightBoxStyle := theme.CardBoxStyle
	if p.FocusPane == PaneNodes {
		rightBoxStyle = theme.CardBoxActiveStyle
	}

	headerText := "Nodes"
	if currGroup != nil {
		headerText = fmt.Sprintf("Nodes in [%s] (Selected: %s)", currGroup.Name, currGroup.Now)
	}
	headerTitle := lipgloss.NewStyle().Bold(true).Render(TruncateVisual(headerText, rightWidth-2))

	rightPanel := rightBoxStyle.Width(rightWidth).Height(height).Render(
		headerTitle + "\n\n" + strings.Join(rightLines, "\n"),
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
}

