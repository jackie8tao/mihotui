package views

import (
	"fmt"
	"strings"
	"time"

	"mihomo-tui/internal/app/theme"
	"mihomo-tui/internal/profile"
	"mihomo-tui/pkg/mihomo"

	"github.com/charmbracelet/lipgloss"
)

// RenderSubscriptions renders the subscription management view.
func RenderSubscriptions(p SubscriptionsProps) string {
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

	// 1. Render Left Column: Subscriptions List with scrolling viewport
	var listLines []string
	if len(p.Subscriptions) == 0 {
		listLines = append(listLines, lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("No subscriptions added yet.\nPress 'a' to add one."))
	} else {
		visibleRows := height - 4
		if visibleRows < 5 {
			visibleRows = 5
		}

		startIdx := 0
		if p.SubCursor >= visibleRows {
			startIdx = p.SubCursor - visibleRows + 1
		}
		endIdx := startIdx + visibleRows
		if endIdx > len(p.Subscriptions) {
			endIdx = len(p.Subscriptions)
		}

		for i := startIdx; i < endIdx; i++ {
			sub := p.Subscriptions[i]
			name := sub.Name
			maxNameLen := leftWidth - 10
			if len(name) > maxNameLen {
				name = name[:maxNameLen-3] + "..."
			}

			tag := lipgloss.NewStyle().Foreground(theme.ColorWarning).Render("[SUB] ")
			line := fmt.Sprintf("%s%s", tag, name)
			if i == p.SubCursor {
				line = theme.SelectedRowStyle.Render("▶ " + line)
			} else {
				line = "  " + line
			}

			listLines = append(listLines, line)
		}
	}

	leftPanel := theme.CardBoxActiveStyle.Width(leftWidth).Height(height).Render(
		fmt.Sprintf("📡 Subscriptions (%d)\n\n%s", len(p.Subscriptions), strings.Join(listLines, "\n")),
	)

	// 2. Render Right Column: Subscription Details
	var curr *profile.Subscription
	if len(p.Subscriptions) > 0 && p.SubCursor >= 0 && p.SubCursor < len(p.Subscriptions) {
		curr = &p.Subscriptions[p.SubCursor]
	}

	var rightContent string
	if curr == nil {
		rightContent = lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("Select a subscription to view details, or press 'a' to add a new subscription.")
	} else {
		var lines []string

		maxValWidth := rightWidth - 16
		if maxValWidth < 15 {
			maxValWidth = 15
		}

		nameDisplay := truncateStr(curr.Name, maxValWidth)
		fileDisplay := truncateStr(curr.FileName, maxValWidth)
		urlDisplay := truncateStr(curr.URL, maxValWidth)

		lines = append(lines, fmt.Sprintf("Subscription : %s", lipgloss.NewStyle().Bold(true).Render(nameDisplay)))
		lines = append(lines, fmt.Sprintf("Target File  : %s", fileDisplay))
		lines = append(lines, fmt.Sprintf("Remote URL   : %s", urlDisplay))
		lines = append(lines, fmt.Sprintf("Last Updated : %s", curr.UpdatedAt.Format("2006-01-02 15:04:05")))

		// Quota and expiration
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(theme.ColorHighlight).Render("─── Traffic & Quota ───"))

		if curr.UserInfo != nil {
			u := curr.UserInfo
			used := u.Upload + u.Download
			total := u.Total

			var pct float64
			if total > 0 {
				pct = float64(used) / float64(total) * 100
			}

			// Draw simple ASCII progress bar
			barWidth := 20
			filled := int(float64(barWidth) * (pct / 100))
			if filled > barWidth {
				filled = barWidth
			}
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

			lines = append(lines, fmt.Sprintf("Usage Ratio  : [%s] %.1f%%", theme.SpeedDownStyle.Render(bar), pct))
			lines = append(lines, fmt.Sprintf("Used / Total : %s / %s (↑ %s, ↓ %s)",
				mihomo.FormatBytes(used),
				mihomo.FormatBytes(total),
				mihomo.FormatBytes(u.Upload),
				mihomo.FormatBytes(u.Download),
			))

			if !u.Expire.IsZero() {
				remaining := time.Until(u.Expire)
				days := int(remaining.Hours() / 24)
				expireStr := fmt.Sprintf("%s (%d days left)", u.Expire.Format("2006-01-02"), days)
				if days <= 7 {
					expireStr = lipgloss.NewStyle().Foreground(theme.ColorWarning).Render(expireStr)
				}
				lines = append(lines, fmt.Sprintf("Expires On   : %s", expireStr))
			}
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("(No quota header returned from subscription provider)"))
		}

		// Actions helper
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("─── Available Actions ───"))
		lines = append(lines, "• [u] / [Enter] Update & re-download this subscription")
		lines = append(lines, "• [U]           Batch update all subscriptions")
		lines = append(lines, "• [a]           Add new subscription (modal)")
		lines = append(lines, "• [d]           Delete this subscription and its local file")

		rightContent = strings.Join(lines, "\n")
	}

	headerTitle := "📦 Subscription Details"
	rightPanel := theme.CardBoxStyle.Width(rightWidth).Height(height).Render(
		headerTitle + "\n\n" + rightContent,
	)

	baseView := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)

	// If modal is active, overlay modal window in center
	if p.ModalActive == ModalAddSub {
		return renderAddSubModal(p, baseView)
	}

	return baseView
}

func renderAddSubModal(p SubscriptionsProps, baseView string) string {
	modalWidth := 56
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.ColorPrimary).
		Background(theme.ColorDark).
		Padding(1, 2).
		Width(modalWidth)

	title := lipgloss.NewStyle().Bold(true).Foreground(theme.ColorLight).Render("➕ Add New Subscription")

	field1 := fmt.Sprintf("Subscription Name:\n%s", p.SubNameInputView)
	field2 := fmt.Sprintf("Subscription URL:\n%s", p.SubURLInputView)

	help := lipgloss.NewStyle().Foreground(theme.ColorMuted).Render("[Tab] Next Field | [Enter] Confirm & Download | [Esc] Cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, "\n", field1, "\n", field2, "\n", help)
	modal := modalStyle.Render(content)

	// Center overlay
	return lipgloss.Place(p.Width, p.Height-4, lipgloss.Center, lipgloss.Center, modal)
}

