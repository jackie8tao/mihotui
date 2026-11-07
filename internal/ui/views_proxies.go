package ui

import (
	"fmt"
	"strings"

	"mihomo-tui/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func renderProxies(m *AppModel) string {
	width := m.width - 4
	if width < 50 {
		width = 50
	}
	height := m.height - 8
	if height < 10 {
		height = 10
	}

	leftWidth := 28
	rightWidth := width - leftWidth - 5
	if rightWidth < 25 {
		rightWidth = 25
	}

	// 1. Render Left Column: Groups List
	var groupLines []string
	if len(m.groups) == 0 {
		groupLines = append(groupLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("No proxy groups found"))
	} else {
		for i, g := range m.groups {
			name := g.Name
			if len(name) > leftWidth-8 {
				name = name[:leftWidth-11] + "..."
			}

			line := fmt.Sprintf("[%d] %s", len(g.All), name)
			if i == m.groupCursor {
				if m.focusPane == PaneGroups {
					line = SelectedRowStyle.Render("▶ " + line)
				} else {
					line = lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render("● " + line)
				}
			} else {
				line = "  " + line
			}
			groupLines = append(groupLines, line)
		}
	}

	leftBoxStyle := CardBoxStyle
	if m.focusPane == PaneGroups {
		leftBoxStyle = CardBoxActiveStyle
	}
	leftPanel := leftBoxStyle.Width(leftWidth).Height(height).Render(
		"📂 Proxy Groups\n\n" + strings.Join(groupLines, "\n"),
	)

	// 2. Render Right Column: Nodes in selected Group
	var rightLines []string
	currGroup := m.currentGroup()
	if currGroup == nil || len(currGroup.All) == 0 {
		rightLines = append(rightLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("No nodes in this group"))
	} else {
		// Calculate visible window for scrolling
		visibleRows := height - 4
		if visibleRows < 5 {
			visibleRows = 5
		}

		startIdx := 0
		if m.nodeCursor >= visibleRows {
			startIdx = m.nodeCursor - visibleRows + 1
		}
		endIdx := startIdx + visibleRows
		if endIdx > len(currGroup.All) {
			endIdx = len(currGroup.All)
		}

		for i := startIdx; i < endIdx; i++ {
			nodeName := currGroup.All[i]
			nodeItem, exists := m.proxies[nodeName]

			// Active mark
			activeMark := "  "
			if nodeName == currGroup.Now {
				activeMark = ActiveIndicatorStyle.Render("✔ ")
			}

			// Delay text and styling
			delayStr := "      -"
			if exists {
				if nodeItem.TestingDelay {
					delayStr = lipgloss.NewStyle().Foreground(ColorWarning).Render("Testing...")
				} else if len(nodeItem.History) > 0 {
					lastDelay := nodeItem.History[len(nodeItem.History)-1].Delay
					delayFormatted := model.FormatDelay(lastDelay)
					switch {
					case lastDelay <= 0:
						delayStr = lipgloss.NewStyle().Foreground(ColorError).Render(fmt.Sprintf("%7s", delayFormatted))
					case lastDelay < 120:
						delayStr = lipgloss.NewStyle().Foreground(ColorSecondary).Render(fmt.Sprintf("%7s", delayFormatted))
					case lastDelay < 300:
						delayStr = lipgloss.NewStyle().Foreground(ColorWarning).Render(fmt.Sprintf("%7s", delayFormatted))
					default:
						delayStr = lipgloss.NewStyle().Foreground(ColorError).Render(fmt.Sprintf("%7s", delayFormatted))
					}
				}
			}

			// Type text
			nodeType := ""
			if exists && nodeItem.Type != "" {
				nodeType = fmt.Sprintf("[%s]", nodeItem.Type)
			}

			// Node name truncation
			availNameWidth := rightWidth - 28
			if availNameWidth < 12 {
				availNameWidth = 12
			}
			displayName := nodeName
			if len(displayName) > availNameWidth {
				displayName = displayName[:availNameWidth-3] + "..."
			}

			row := fmt.Sprintf("%s%-*s %-10s %s",
				activeMark,
				availNameWidth, displayName,
				lipgloss.NewStyle().Foreground(ColorMuted).Render(nodeType),
				delayStr,
			)

			if i == m.nodeCursor && m.focusPane == PaneNodes {
				row = SelectedRowStyle.Render("▶ " + row[2:])
			}

			rightLines = append(rightLines, row)
		}
	}

	rightBoxStyle := CardBoxStyle
	if m.focusPane == PaneNodes {
		rightBoxStyle = CardBoxActiveStyle
	}

	headerTitle := "📡 Nodes"
	if currGroup != nil {
		headerTitle = fmt.Sprintf("📡 Nodes in [%s] (Selected: %s)", currGroup.Name, currGroup.Now)
	}

	rightPanel := rightBoxStyle.Width(rightWidth).Height(height).Render(
		headerTitle + "\n\n" + strings.Join(rightLines, "\n"),
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
}
