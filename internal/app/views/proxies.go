package views

import (
	"fmt"

	"github.com/jackie8tao/mihotui/internal/app/components"
	"github.com/jackie8tao/mihotui/internal/app/theme"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

const (
	proxiesLeftPaneWidth = 30
	proxyDelayColWidth   = 10
	proxyTypeColWidth    = 14
)

// RenderProxies renders the proxies and node groups view.
func RenderProxies(p ProxiesProps) string {
	var currGroup *mihomo.ProxyItem
	if len(p.Groups) > 0 && p.SelectedGroupIdx >= 0 && p.SelectedGroupIdx < len(p.Groups) {
		currGroup = &p.Groups[p.SelectedGroupIdx]
	}

	return components.SplitPane(p.Width, p.Height, proxiesLeftPaneWidth,
		func(w, h int) string {
			return renderProxyGroupsPane(p, w, h)
		},
		func(w, h int) string {
			return renderProxyNodesPane(p, currGroup, w, h)
		},
	)
}

// renderProxyGroupsPane renders the left sidebar listing selectable proxy groups.
func renderProxyGroupsPane(p ProxiesProps, width, height int) string {
	return components.Panel(components.PanelProps{
		Title:  "Proxy Groups",
		Active: p.FocusPane == PaneGroups,
		Width:  width,
		Height: height,
	}, func(cw, ch int) string {
		return components.ScrollList(components.ListProps{
			Total:        len(p.Groups),
			SelectedIdx:  p.SelectedGroupIdx,
			Focused:      p.FocusPane == PaneGroups,
			EmptyMessage: "No proxy groups found",
			RenderRow: func(idx, rowW int, selected, focused bool) []components.Segment {
				g := p.Groups[idx]
				prefix := fmt.Sprintf("[%d] ", len(g.All))
				nameMax := rowW - 2 - len(prefix)
				if nameMax < 4 {
					nameMax = 4
				}
				label := prefix + components.TruncateVisual(g.Name, nameMax)
				if selected && !focused {
					return []components.Segment{components.Styled("● "+label, theme.HighlightStyle)}
				}
				cursor := "  "
				if selected {
					cursor = "▶ "
				}
				return []components.Segment{components.Plain(cursor + label)}
			},
		}, cw, ch)
	})
}

// renderProxyNodesPane renders the right pane listing nodes in the currently selected group.
func renderProxyNodesPane(p ProxiesProps, currGroup *mihomo.ProxyItem, width, height int) string {
	headerTitle := "Nodes"
	if currGroup != nil {
		headerTitle = fmt.Sprintf("Nodes in [%s] (Selected: %s)", currGroup.Name, currGroup.Now)
	}

	return components.Panel(components.PanelProps{
		Title:  headerTitle,
		Active: p.FocusPane == PaneNodes,
		Width:  width,
		Height: height,
	}, func(cw, ch int) string {
		if currGroup == nil || len(currGroup.All) == 0 {
			return theme.MutedStyle.Render("No nodes in this group")
		}

		return components.ScrollList(components.ListProps{
			Total:        len(currGroup.All),
			SelectedIdx:  p.SelectedNodeIdx,
			Focused:      p.FocusPane == PaneNodes,
			EmptyMessage: "No nodes in this group",
			RenderRow: func(idx, rowW int, selected, focused bool) []components.Segment {
				return renderProxyNodeRow(p.Proxies, currGroup, currGroup.All[idx], rowW, selected && focused)
			},
		}, cw, ch)
	})
}

// renderProxyNodeRow builds the aligned multi-segment row for a single proxy node.
func renderProxyNodeRow(proxies map[string]mihomo.ProxyItem, group *mihomo.ProxyItem, nodeName string, rowW int, activeCursor bool) []components.Segment {
	nodeItem, exists := proxies[nodeName]

	cursor := "  "
	if activeCursor {
		cursor = "▶ "
	}

	activeSeg := components.Plain("  ")
	if nodeName == group.Now {
		activeSeg = components.Styled("✔ ", theme.ActiveIndicatorStyle)
	}

	nodeType := ""
	if exists && nodeItem.Type != "" {
		nodeType = fmt.Sprintf("[%s]", nodeItem.Type)
	}

	// Adapt column widths on narrow panes so rows never wrap onto a second line.
	const fixedPrefix = 4 // cursor (2) + activeMark (2)
	typeW := proxyTypeColWidth
	nameW := rowW - fixedPrefix - typeW - proxyDelayColWidth - 2
	if nameW < 8 {
		typeW = max(0, rowW-fixedPrefix-proxyDelayColWidth-10)
		gapW := 1
		if typeW > 0 {
			gapW = 2
		}
		nameW = max(4, rowW-fixedPrefix-typeW-proxyDelayColWidth-gapW)
	}

	segs := []components.Segment{
		components.Plain(cursor),
		activeSeg,
		components.Plain(components.PadRightVisual(nodeName, nameW) + " "),
	}
	if typeW > 0 {
		segs = append(segs,
			components.Styled(components.PadRightVisual(nodeType, typeW), theme.MutedStyle),
			components.Plain(" "),
		)
	}
	segs = append(segs, formatDelaySegment(nodeItem, exists))
	return segs
}

// formatDelaySegment formats and colors the right-aligned node latency cell.
func formatDelaySegment(item mihomo.ProxyItem, exists bool) components.Segment {
	if !exists {
		return components.Plain(components.PadLeftVisual("-", proxyDelayColWidth))
	}
	if item.TestingDelay {
		return components.Styled(components.PadLeftVisual("Testing...", proxyDelayColWidth), theme.WarningStyle)
	}
	if len(item.History) == 0 {
		return components.Plain(components.PadLeftVisual("-", proxyDelayColWidth))
	}

	lastDelay := item.History[len(item.History)-1].Delay
	formatted := components.PadLeftVisual(mihomo.FormatDelay(lastDelay), proxyDelayColWidth)
	switch {
	case lastDelay <= 0:
		return components.Styled(formatted, theme.ErrorStyle)
	case lastDelay < 120:
		return components.Styled(formatted, theme.SuccessStyle)
	case lastDelay < 300:
		return components.Styled(formatted, theme.WarningStyle)
	default:
		return components.Styled(formatted, theme.ErrorStyle)
	}
}
