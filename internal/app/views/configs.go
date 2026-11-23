package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jackie8tao/mihotui/internal/app/theme"
)

type configItemMeta struct {
	label string
	key   string
	desc  string
}

var tunItemsMeta = []configItemMeta{
	{
		label: "TUN Mode",
		key:   "tun.enable",
		desc:  "Enable or disable Mihomo virtual TUN interface to capture system-wide traffic.",
	},
	{
		label: "Network Stack",
		key:   "tun.stack",
		desc:  "TCP/IP stack implementation: mixed (recommended), system (native OS), or gvisor (userspace).",
	},
	{
		label: "Auto Route",
		key:   "tun.auto-route",
		desc:  "Automatically configure system routing table to direct traffic into the TUN interface.",
	},
	{
		label: "Auto Detect Iface",
		key:   "tun.auto-detect-interface",
		desc:  "Automatically bind outbound traffic to the default physical network interface.",
	},
	{
		label: "DNS Hijack",
		key:   "tun.dns-hijack",
		desc:  "Intercept outbound DNS queries (any:53) through Mihomo DNS to prevent DNS leaks.",
	},
}

// RenderConfigs renders the override.yaml system configuration view.
func RenderConfigs(p ConfigsProps) string {
	width := p.Width - 4
	if width < 50 {
		width = 50
	}
	height := p.Height - 8
	if height < 10 {
		height = 10
	}

	leftWidth := 36
	rightWidth := width - leftWidth - 5
	if rightWidth < 28 {
		rightWidth = 28
	}

	// 1. Render Left Column: TUN Override Options
	values := []string{
		renderBoolBadge(p.Tun.Enable),
		lipgloss.NewStyle().Bold(true).Foreground(theme.ColorHighlight).Render(fmt.Sprintf("< %s >", p.Tun.Stack)),
		renderBoolBadge(p.Tun.AutoRoute),
		renderBoolBadge(p.Tun.AutoDetectInterface),
		renderBoolBadge(p.Tun.DNSHijack),
	}

	var listLines []string
	listLines = append(listLines, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorHighlight).Render("─── TUN Settings ───"))
	listLines = append(listLines, "")

	for i, item := range tunItemsMeta {
		row := fmt.Sprintf("%-18s %s", item.label, values[i])
		if i == p.SelectedIdx {
			row = theme.SelectedRowStyle.Render("▶ " + row)
		} else {
			row = "  " + row
		}
		listLines = append(listLines, row)
	}

	leftHeader := "⚙️ System Override"
	if p.Dirty {
		leftHeader += " " + lipgloss.NewStyle().Foreground(theme.ColorWarning).Bold(true).Render("[Modified]")
	}
	leftPanel := theme.CardBoxActiveStyle.Width(leftWidth).Height(height).Render(
		leftHeader + "\n\n" + strings.Join(listLines, "\n"),
	)

	// 2. Render Right Column: Paths, Selected Description, and Live YAML Preview
	maxValWidth := rightWidth - 16
	if maxValWidth < 15 {
		maxValWidth = 15
	}

	activeSubDisplay := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("(none)")
	if p.ActiveSubName != "" {
		activeSubDisplay = theme.ActiveIndicatorStyle.Render(truncateStr(p.ActiveSubName, maxValWidth))
	}

	var rightLines []string
	rightLines = append(rightLines, fmt.Sprintf("Override File : %s", truncateStr(p.OverridePath, maxValWidth)))
	rightLines = append(rightLines, fmt.Sprintf("Active Sub    : %s", activeSubDisplay))
	rightLines = append(rightLines, fmt.Sprintf("Merged Output : %s", truncateStr(p.MihomoConfigPath, maxValWidth)))

	if p.SelectedIdx >= 0 && p.SelectedIdx < len(tunItemsMeta) {
		meta := tunItemsMeta[p.SelectedIdx]
		rightLines = append(rightLines, "")
		rightLines = append(rightLines, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorHighlight).Render("─── Selected Option ───"))
		rightLines = append(rightLines, fmt.Sprintf("Key  : %s", lipgloss.NewStyle().Bold(true).Render(meta.key)))
		rightLines = append(rightLines, lipgloss.NewStyle().Width(rightWidth-4).Foreground(theme.ColorLight).Render(meta.desc))
	}

	rightLines = append(rightLines, "")
	rightLines = append(rightLines, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorHighlight).Render("─── override.yaml Preview ───"))
	rightLines = append(rightLines, renderTunYAMLPreview(p))

	rightLines = append(rightLines, "")
	rightLines = append(rightLines, lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("─── Available Actions ───"))
	rightLines = append(rightLines, "• [←/→]   Toggle or cycle selected option")
	rightLines = append(rightLines, "• [Enter] Save override.yaml, merge with active sub & reload Mihomo")

	rightPanel := theme.CardBoxStyle.Width(rightWidth).Height(height).Render(
		"📄 Override Details\n\n" + strings.Join(rightLines, "\n"),
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
}

func renderBoolBadge(v bool) string {
	if v {
		return theme.ActiveIndicatorStyle.Render("[ON]")
	}
	return lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("[OFF]")
}

func renderTunYAMLPreview(p ConfigsProps) string {
	var lines []string
	lines = append(lines, "tun:")
	lines = append(lines, fmt.Sprintf("  enable: %v", p.Tun.Enable))
	lines = append(lines, fmt.Sprintf("  stack: %s", p.Tun.Stack))
	lines = append(lines, fmt.Sprintf("  auto-route: %v", p.Tun.AutoRoute))
	lines = append(lines, fmt.Sprintf("  auto-detect-interface: %v", p.Tun.AutoDetectInterface))
	if p.Tun.DNSHijack {
		lines = append(lines, "  dns-hijack:")
		lines = append(lines, "    - any:53")
	}
	return lipgloss.NewStyle().Foreground(theme.ColorMuted).Render(strings.Join(lines, "\n"))
}

func truncateStr(s string, maxLen int) string {
	if maxLen <= 3 {
		return s
	}
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen-3]) + "..."
	}
	return s
}
