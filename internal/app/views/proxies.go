package views

import (
	"fmt"
	"strings"

	"mihomo-tui/internal/app/theme"
	"mihomo-tui/pkg/mihomo"

	"github.com/charmbracelet/lipgloss"
)

// RenderProxies renders the proxies and node groups view.
func RenderProxies(p ProxiesProps) string {
	width := p.Width - 4
	if width < 50 {
		width = 50
	}
	height := p.Height - 8
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
	if len(p.Groups) == 0 {
		groupLines = append(groupLines, lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("No proxy groups found"))
	} else {
		for i, g := range p.Groups {
			name := g.Name
			if len(name) > leftWidth-8 {
				name = name[:leftWidth-11] + "..."
			}

			line := fmt.Sprintf("[%d] %s", len(g.All), name)
			if i == p.GroupCursor {
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
	leftPanel := leftBoxStyle.Width(leftWidth).Height(height).Render(
		"📂 Proxy Groups\n\n" + strings.Join(groupLines, "\n"),
	)

	// 2. Render Right Column: Nodes in selected Group
	var currGroup *mihomo.ProxyItem
	if len(p.Groups) > 0 && p.GroupCursor >= 0 && p.GroupCursor < len(p.Groups) {
		currGroup = &p.Groups[p.GroupCursor]
	}

	var rightLines []string
	if currGroup == nil || len(currGroup.All) == 0 {
		rightLines = append(rightLines, lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("No nodes in this group"))
	} else {
		// Calculate visible window for scrolling
		visibleRows := height - 4
		if visibleRows < 5 {
			visibleRows = 5
		}

		startIdx := 0
		if p.NodeCursor >= visibleRows {
			startIdx = p.NodeCursor - visibleRows + 1
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

			// Delay text and styling
			delayStr := "      -"
			if exists {
				if nodeItem.TestingDelay {
					delayStr = lipgloss.NewStyle().Foreground(theme.ColorWarning).Render("Testing...")
				} else if len(nodeItem.History) > 0 {
					lastDelay := nodeItem.History[len(nodeItem.History)-1].Delay
					delayFormatted := mihomo.FormatDelay(lastDelay)
					switch {
					case lastDelay <= 0:
						delayStr = lipgloss.NewStyle().Foreground(theme.ColorError).Render(fmt.Sprintf("%7s", delayFormatted))
					case lastDelay < 120:
						delayStr = lipgloss.NewStyle().Foreground(theme.ColorSecondary).Render(fmt.Sprintf("%7s", delayFormatted))
					case lastDelay < 300:
						delayStr = lipgloss.NewStyle().Foreground(theme.ColorWarning).Render(fmt.Sprintf("%7s", delayFormatted))
					default:
						delayStr = lipgloss.NewStyle().Foreground(theme.ColorError).Render(fmt.Sprintf("%7s", delayFormatted))
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
				lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(nodeType),
				delayStr,
			)

			if i == p.NodeCursor && p.FocusPane == PaneNodes {
				row = theme.SelectedRowStyle.Render("▶ " + row[2:])
			}

			rightLines = append(rightLines, row)
		}
	}

	rightBoxStyle := theme.CardBoxStyle
	if p.FocusPane == PaneNodes {
		rightBoxStyle = theme.CardBoxActiveStyle
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

