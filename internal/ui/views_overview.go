package ui

import (
	"fmt"
	"strings"

	"mihomo-tui/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func renderOverview(m *AppModel) string {
	width := m.width - 4
	if width < 40 {
		width = 40
	}

	halfWidth := (width - 4) / 2
	if halfWidth < 20 {
		halfWidth = 20
	}

	// 1. Traffic & Speed Box
	upSpeedStr := model.FormatSpeed(m.traffic.Up)
	downSpeedStr := model.FormatSpeed(m.traffic.Down)

	trafficContent := fmt.Sprintf(
		"▲ Upload   : %s\n▼ Download : %s\n\nTotal Up   : %s\nTotal Down : %s",
		SpeedUpStyle.Render(upSpeedStr),
		SpeedDownStyle.Render(downSpeedStr),
		model.FormatBytes(m.totalUp),
		model.FormatBytes(m.totalDown),
	)
	trafficBox := CardBoxStyle.Width(halfWidth).Render("🌐 Real-time Bandwidth\n\n" + trafficContent)

	// 2. Mode Box
	modeList := []string{"rule", "global", "direct"}
	var modeStrs []string
	currMode := strings.ToLower(m.config.Mode)
	for _, mode := range modeList {
		if mode == currMode {
			modeStrs = append(modeStrs, ActiveIndicatorStyle.Render(fmt.Sprintf("● [%s]", strings.ToUpper(mode))))
		} else {
			modeStrs = append(modeStrs, lipgloss.NewStyle().Foreground(ColorMuted).Render(fmt.Sprintf("○  %s", strings.ToUpper(mode))))
		}
	}
	modeHelp := lipgloss.NewStyle().Foreground(ColorMuted).Render("\n(Press 'm' to cycle mode)")
	modeBox := CardBoxStyle.Width(halfWidth).Render("⚙️ Outbound Mode\n\n" + strings.Join(modeStrs, "\n") + modeHelp)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, trafficBox, "  ", modeBox)

	// 3. System & Config Info Box
	tunStatus := "Disabled"
	if m.config.Tun.Enable {
		tunStatus = fmt.Sprintf("Enabled (dev: %s, stack: %s)", m.config.Tun.Device, m.config.Tun.Stack)
	}

	metaTag := ""
	if m.version.Meta {
		metaTag = " (Meta)"
	}

	memStr := "N/A"
	if m.memory.Inuse > 0 {
		memStr = fmt.Sprintf("%s (OS: %s)", model.FormatBytes(m.memory.Inuse), model.FormatBytes(m.memory.Osalloc))
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
		m.version.Version, metaTag,
		m.config.MixedPort,
		m.config.SocksPort,
		m.config.RedirPort,
		tunStatus,
		m.config.AllowLan,
		m.config.LogLevel,
		memStr,
	)
	sysBox := CardBoxStyle.Width(width).Render("📋 Runtime Information\n\n" + sysContent)

	return lipgloss.JoinVertical(lipgloss.Left, topRow, "\n", sysBox)
}
