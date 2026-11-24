package views

import (
	"fmt"
	"strings"

	"github.com/jackie8tao/mihotui/internal/app/theme"
	"github.com/jackie8tao/mihotui/pkg/mihomo"

	"github.com/charmbracelet/lipgloss"
)

// RenderOverview renders the overview dashboard view.
func RenderOverview(p OverviewProps) string {
	width := p.Width - 4
	if width < 44 {
		width = 44
	}

	leftWidth := (width - 4) / 2
	rightWidth := width - 4 - leftWidth
	topBoxHeight := 7

	// 1. Traffic & Speed Box
	upSpeedStr := mihomo.FormatSpeed(p.Traffic.Up)
	downSpeedStr := mihomo.FormatSpeed(p.Traffic.Down)

	trafficHeader := theme.BoldStyle.Render("Real-time Bandwidth")
	trafficContent := fmt.Sprintf(
		"▲ Upload   : %s\n▼ Download : %s\n\nTotal Up   : %s\nTotal Down : %s",
		theme.SpeedUpStyle.Render(upSpeedStr),
		theme.SpeedDownStyle.Render(downSpeedStr),
		mihomo.FormatBytes(p.TotalUp),
		mihomo.FormatBytes(p.TotalDown),
	)
	trafficBox := theme.CardBoxStyle.Width(leftWidth).Height(topBoxHeight).Render(trafficHeader + "\n\n" + trafficContent)

	// 2. Mode Box
	modeList := []string{"rule", "global", "direct"}
	var modeStrs []string
	currMode := strings.ToLower(p.Config.Mode)
	for _, mode := range modeList {
		if mode == currMode {
			modeStrs = append(modeStrs, theme.ActiveIndicatorStyle.Render(fmt.Sprintf("● [%s]", strings.ToUpper(mode))))
		} else {
			modeStrs = append(modeStrs, theme.MutedStyle.Render(fmt.Sprintf("○  %s", strings.ToUpper(mode))))
		}
	}
	modeHeader := theme.BoldStyle.Render("Outbound Mode")
	modeHelp := theme.MutedStyle.Render("(Press 'm' to cycle mode)")
	modeBox := theme.CardBoxStyle.Width(rightWidth).Height(topBoxHeight).Render(
		modeHeader + "\n\n" + strings.Join(modeStrs, "\n") + "\n\n" + modeHelp,
	)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, trafficBox, "  ", modeBox)

	// 3. System & Config Info Box
	tunStatus := "Disabled"
	if p.Config.Tun.Enable {
		tunStatus = fmt.Sprintf("Enabled (dev: %s, stack: %s)", p.Config.Tun.Device, p.Config.Tun.Stack)
	}

	metaTag := ""
	if p.Version.Meta {
		metaTag = " (Meta)"
	}

	memStr := "N/A"
	if p.Memory.Inuse > 0 {
		memStr = fmt.Sprintf("%s (OS: %s)", mihomo.FormatBytes(p.Memory.Inuse), mihomo.FormatBytes(p.Memory.Osalloc))
	}

	sysContent := fmt.Sprintf(
		"Core Version  : %s%s\n"+
			"Mixed Port    : %d\n"+
			"Socks5 Port   : %d\n"+
			"Redir Port    : %d\n"+
			"Tun Mode      : %s\n"+
			"Allow LAN     : %v\n"+
			"Log Level     : %s\n"+
			"Memory Usage  : %s",
		p.Version.Version, metaTag,
		p.Config.MixedPort,
		p.Config.SocksPort,
		p.Config.RedirPort,
		tunStatus,
		p.Config.AllowLan,
		p.Config.LogLevel,
		memStr,
	)
	sysHeader := theme.BoldStyle.Render("Runtime Information")
	sysBox := theme.CardBoxStyle.Width(width).Render(sysHeader + "\n\n" + sysContent)

	return lipgloss.JoinVertical(lipgloss.Left, topRow, "", sysBox)
}

