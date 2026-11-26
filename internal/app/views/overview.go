package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jackie8tao/mihotui/internal/app/components"
	"github.com/jackie8tao/mihotui/internal/app/theme"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

const overviewTopPanelHeight = 9 // 2 border + 2 header + 5 content rows

// RenderOverview renders the overview dashboard view.
func RenderOverview(p OverviewProps) string {
	width := p.Width
	if width < 24 {
		width = 24
	}

	leftW := (width - 2) / 2
	topRow := components.SplitPane(width, overviewTopPanelHeight, leftW,
		func(w, h int) string {
			return renderTrafficPanel(p, w, h)
		},
		func(w, h int) string {
			return renderModePanel(p, w, h)
		},
	)

	bottomHeight := 0
	if p.Height > 0 {
		bottomHeight = p.Height - overviewTopPanelHeight - 1
		if bottomHeight < 5 {
			bottomHeight = 5
		}
	}

	sysBox := renderRuntimePanel(p, width, bottomHeight)
	return lipgloss.JoinVertical(lipgloss.Left, topRow, "", sysBox)
}

// renderTrafficPanel renders the real-time upload/download speed and cumulative totals card.
func renderTrafficPanel(p OverviewProps, width, height int) string {
	return components.Panel(components.PanelProps{
		Title:  "Real-time Bandwidth",
		Width:  width,
		Height: height,
	}, func(cw, _ int) string {
		speeds := components.KVList([]components.KVItem{
			{Key: "▲ Upload", Value: mihomo.FormatSpeed(p.Traffic.Up), ValueStyle: &theme.SpeedUpStyle},
			{Key: "▼ Download", Value: mihomo.FormatSpeed(p.Traffic.Down), ValueStyle: &theme.SpeedDownStyle},
		}, cw)
		totals := components.KVList([]components.KVItem{
			{Key: "Total Up", Value: mihomo.FormatBytes(p.TotalUp)},
			{Key: "Total Down", Value: mihomo.FormatBytes(p.TotalDown)},
		}, cw)
		return speeds + "\n\n" + totals
	})
}

// renderModePanel renders the outbound routing mode indicator card.
func renderModePanel(p OverviewProps, width, height int) string {
	return components.Panel(components.PanelProps{
		Title:  "Outbound Mode",
		Width:  width,
		Height: height,
	}, func(cw, _ int) string {
		modes := []string{"rule", "global", "direct"}
		currMode := strings.ToLower(p.Config.Mode)
		lines := make([]string, 0, len(modes)+2)
		for _, mode := range modes {
			if mode == currMode {
				lines = append(lines, theme.ActiveIndicatorStyle.Render(
					components.TruncateVisual(fmt.Sprintf("● [%s]", strings.ToUpper(mode)), cw),
				))
				continue
			}
			lines = append(lines, theme.MutedStyle.Render(
				components.TruncateVisual(fmt.Sprintf("○  %s", strings.ToUpper(mode)), cw),
			))
		}
		lines = append(lines, "", theme.MutedStyle.Render(components.TruncateVisual("(Press 'm' to cycle mode)", cw)))
		return strings.Join(lines, "\n")
	})
}

// renderRuntimePanel renders the Mihomo core version, ports, TUN, and memory info card.
func renderRuntimePanel(p OverviewProps, width, height int) string {
	return components.Panel(components.PanelProps{
		Title:  "Runtime Information",
		Width:  width,
		Height: height,
	}, func(cw, _ int) string {
		tunStatus := "Disabled"
		if p.Config.Tun.Enable {
			tunStatus = fmt.Sprintf("Enabled (dev: %s, stack: %s)", p.Config.Tun.Device, p.Config.Tun.Stack)
		}

		versionStr := p.Version.Version
		if p.Version.Meta {
			versionStr += " (Meta)"
		}

		memStr := "N/A"
		if p.Memory.Inuse > 0 {
			memStr = fmt.Sprintf("%s (OS: %s)", mihomo.FormatBytes(p.Memory.Inuse), mihomo.FormatBytes(p.Memory.Osalloc))
		}

		return components.KVList([]components.KVItem{
			{Key: "Core Version", Value: versionStr},
			{Key: "Mixed Port", Value: fmt.Sprintf("%d", p.Config.MixedPort)},
			{Key: "Socks5 Port", Value: fmt.Sprintf("%d", p.Config.SocksPort)},
			{Key: "Redir Port", Value: fmt.Sprintf("%d", p.Config.RedirPort)},
			{Key: "Tun Mode", Value: tunStatus},
			{Key: "Allow LAN", Value: fmt.Sprintf("%v", p.Config.AllowLan)},
			{Key: "Log Level", Value: p.Config.LogLevel},
			{Key: "Memory Usage", Value: memStr},
		}, cw)
	})
}
