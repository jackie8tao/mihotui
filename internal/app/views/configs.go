package views

import (
	"fmt"
	"strings"

	"mihomo-tui/internal/app/theme"
	"mihomo-tui/internal/profile"
	"mihomo-tui/pkg/mihomo"

	"github.com/charmbracelet/lipgloss"
)

// RenderConfigs renders the local configuration profiles view.
func RenderConfigs(p ConfigsProps) string {
	width := p.Width - 4
	if width < 50 {
		width = 50
	}
	height := p.Height - 8
	if height < 10 {
		height = 10
	}

	leftWidth := 34
	rightWidth := width - leftWidth - 5
	if rightWidth < 25 {
		rightWidth = 25
	}

	// 1. Render Left Column: Profiles List with scrolling viewport
	var listLines []string
	if len(p.Profiles) == 0 {
		listLines = append(listLines, lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("No config profiles found"))
	} else {
		visibleRows := height - 4
		if visibleRows < 5 {
			visibleRows = 5
		}

		startIdx := 0
		if p.ProfileCursor >= visibleRows {
			startIdx = p.ProfileCursor - visibleRows + 1
		}
		endIdx := startIdx + visibleRows
		if endIdx > len(p.Profiles) {
			endIdx = len(p.Profiles)
		}

		for i := startIdx; i < endIdx; i++ {
			prof := p.Profiles[i]
			name := prof.Name
			maxNameLen := leftWidth - 10
			if len(name) > maxNameLen {
				name = name[:maxNameLen-3] + "..."
			}

			tag := ""
			if prof.Subscription != nil {
				tag = lipgloss.NewStyle().Foreground(theme.ColorWarning).Render("[SUB] ")
			}

			activeMark := "  "
			if prof.IsActive {
				activeMark = theme.ActiveIndicatorStyle.Render("✔ ")
			}

			line := fmt.Sprintf("%s%s%s", activeMark, tag, name)
			if i == p.ProfileCursor {
				line = theme.SelectedRowStyle.Render("▶ " + line[2:])
			}

			listLines = append(listLines, line)
		}
	}

	leftPanel := theme.CardBoxActiveStyle.Width(leftWidth).Height(height).Render(
		fmt.Sprintf("📁 Local Profiles (%d)\n\n%s", len(p.Profiles), strings.Join(listLines, "\n")),
	)

	// 2. Render Right Column: Profile Details
	var curr *profile.Profile
	if len(p.Profiles) > 0 && p.ProfileCursor >= 0 && p.ProfileCursor < len(p.Profiles) {
		curr = &p.Profiles[p.ProfileCursor]
	}

	var rightContent string
	if curr == nil {
		rightContent = lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("Select a profile to view details")
	} else {
		var lines []string

		// Status Badge
		statusBadge := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("[INACTIVE]")
		if curr.IsActive {
			statusBadge = theme.ActiveIndicatorStyle.Render("[CURRENTLY ACTIVE]")
		}

		maxValWidth := rightWidth - 18
		if maxValWidth < 15 {
			maxValWidth = 15
		}

		nameDisplay := truncateStr(curr.Name, maxValWidth-10)
		fileDisplay := truncateStr(curr.FileName, maxValWidth)
		pathDisplay := truncateStr(curr.FullPath, maxValWidth)

		lines = append(lines, fmt.Sprintf("Profile Name : %s  %s", lipgloss.NewStyle().Bold(true).Render(nameDisplay), statusBadge))
		lines = append(lines, fmt.Sprintf("File Name    : %s", fileDisplay))
		lines = append(lines, fmt.Sprintf("Full Path    : %s", pathDisplay))
		lines = append(lines, fmt.Sprintf("File Size    : %s", mihomo.FormatBytes(curr.Size)))
		lines = append(lines, fmt.Sprintf("Last Modified: %s", curr.ModTime.Format("2006-01-02 15:04:05")))

		if curr.Subscription != nil {
			subName := truncateStr(curr.Subscription.Name, maxValWidth-12)
			lines = append(lines, "")
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorWarning).Render(
				fmt.Sprintf("ℹ️ Linked to [%s] (see '4. Subs' tab)", subName),
			))
		}

		// Actions helper
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("─── Available Actions ───"))
		lines = append(lines, "• [Enter] Apply & Hot-reload this profile to Mihomo")
		lines = append(lines, "• [r]     Refresh profile list from disk")
		lines = append(lines, "• [Tab]   Switch to other tabs")

		rightContent = strings.Join(lines, "\n")
	}

	headerTitle := fmt.Sprintf("⚙️ Profile Details (Dir: %s)", truncateStr(p.ConfigDir, rightWidth-24))
	rightPanel := theme.CardBoxStyle.Width(rightWidth).Height(height).Render(
		headerTitle + "\n\n" + rightContent,
	)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)
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
