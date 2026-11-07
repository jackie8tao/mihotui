package ui

import (
	"fmt"
	"strings"
	"time"

	"mihomo-tui/internal/model"

	"github.com/charmbracelet/lipgloss"
)

func renderConfigs(m *AppModel) string {
	width := m.width - 4
	if width < 50 {
		width = 50
	}
	height := m.height - 8
	if height < 10 {
		height = 10
	}

	leftWidth := 32
	rightWidth := width - leftWidth - 5
	if rightWidth < 25 {
		rightWidth = 25
	}

	// 1. Render Left Column: Profiles & Subscriptions List
	var listLines []string
	if len(m.profiles) == 0 {
		listLines = append(listLines, lipgloss.NewStyle().Foreground(ColorMuted).Render("No config profiles found"))
	} else {
		for i, p := range m.profiles {
			name := p.Name
			if len(name) > leftWidth-10 {
				name = name[:leftWidth-13] + "..."
			}

			tag := ""
			if p.Subscription != nil {
				tag = lipgloss.NewStyle().Foreground(ColorWarning).Render("[SUB] ")
			}

			activeMark := "  "
			if p.IsActive {
				activeMark = ActiveIndicatorStyle.Render("✔ ")
			}

			line := fmt.Sprintf("%s%s%s", activeMark, tag, name)
			if i == m.profileCursor {
				line = SelectedRowStyle.Render("▶ " + line[2:])
			}

			listLines = append(listLines, line)
		}
	}

	leftPanel := CardBoxActiveStyle.Width(leftWidth).Height(height).Render(
		fmt.Sprintf("📁 Profiles (%d)\n\n%s", len(m.profiles), strings.Join(listLines, "\n")),
	)

	// 2. Render Right Column: Profile Details
	curr := m.currentProfile()
	var rightContent string
	if curr == nil {
		rightContent = lipgloss.NewStyle().Foreground(ColorMuted).Render("Select a profile to view details")
	} else {
		var lines []string

		// Status Badge
		statusBadge := lipgloss.NewStyle().Foreground(ColorMuted).Render("[INACTIVE]")
		if curr.IsActive {
			statusBadge = ActiveIndicatorStyle.Render("[CURRENTLY ACTIVE]")
		}

		lines = append(lines, fmt.Sprintf("Profile Name : %s  %s", lipgloss.NewStyle().Bold(true).Render(curr.Name), statusBadge))
		lines = append(lines, fmt.Sprintf("File Name    : %s", curr.FileName))
		lines = append(lines, fmt.Sprintf("Path         : %s", curr.FullPath))
		lines = append(lines, fmt.Sprintf("Size         : %s", model.FormatBytes(curr.Size)))
		lines = append(lines, fmt.Sprintf("Modified     : %s", curr.ModTime.Format("2006-01-02 15:04:05")))

		// If it's a subscription
		if curr.Subscription != nil {
			sub := curr.Subscription
			lines = append(lines, "")
			lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(ColorHighlight).Render("─── Subscription Info ───"))
			lines = append(lines, fmt.Sprintf("URL          : %s", sub.URL))
			lines = append(lines, fmt.Sprintf("Last Updated : %s", sub.UpdatedAt.Format("2006-01-02 15:04:05")))

			if sub.UserInfo != nil {
				u := sub.UserInfo
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

				lines = append(lines, "")
				lines = append(lines, fmt.Sprintf("Traffic Quota: [%s] %.1f%%", SpeedDownStyle.Render(bar), pct))
				lines = append(lines, fmt.Sprintf("Used / Total : %s / %s (↑ %s, ↓ %s)",
					model.FormatBytes(used),
					model.FormatBytes(total),
					model.FormatBytes(u.Upload),
					model.FormatBytes(u.Download),
				))

				if !u.Expire.IsZero() {
					remaining := time.Until(u.Expire)
					days := int(remaining.Hours() / 24)
					expireStr := fmt.Sprintf("%s (%d days left)", u.Expire.Format("2006-01-02"), days)
					if days <= 7 {
						expireStr = lipgloss.NewStyle().Foreground(ColorWarning).Render(expireStr)
					}
					lines = append(lines, fmt.Sprintf("Expires On   : %s", expireStr))
				}
			} else {
				lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("(No quota header returned from provider)"))
			}
		}

		// Actions helper
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorMuted).Render("─── Available Actions ───"))
		lines = append(lines, "• [Enter] Apply & Hot-reload this profile to Mihomo")
		if curr.Subscription != nil {
			lines = append(lines, "• [u]     Update & re-download this subscription")
			lines = append(lines, "• [d]     Delete this subscription")
		}
		lines = append(lines, "• [U]     Update all subscriptions")
		lines = append(lines, "• [a]     Add new subscription")

		rightContent = strings.Join(lines, "\n")
	}

	headerTitle := fmt.Sprintf("⚙️ Profile Details (Dir: %s)", m.profileMgr.ConfigDir())
	rightPanel := CardBoxStyle.Width(rightWidth).Height(height).Render(
		headerTitle + "\n\n" + rightContent,
	)

	baseView := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, "  ", rightPanel)

	// If modal is active, overlay modal window in center
	if m.modalActive == ModalAddSub {
		return renderAddSubModal(m, baseView)
	}

	return baseView
}

func renderAddSubModal(m *AppModel, baseView string) string {
	modalWidth := 56
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ColorPrimary).
		Background(ColorDark).
		Padding(1, 2).
		Width(modalWidth)

	title := lipgloss.NewStyle().Bold(true).Foreground(ColorLight).Render("➕ Add New Subscription")

	field1 := fmt.Sprintf("Subscription Name:\n%s", m.subNameInput.View())
	field2 := fmt.Sprintf("Subscription URL:\n%s", m.subURLInput.View())

	help := lipgloss.NewStyle().Foreground(ColorMuted).Render("[Tab] Next Field | [Enter] Confirm & Download | [Esc] Cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, "\n", field1, "\n", field2, "\n", help)
	modal := modalStyle.Render(content)

	// Center overlay
	return lipgloss.Place(m.width, m.height-4, lipgloss.Center, lipgloss.Center, modal)
}
