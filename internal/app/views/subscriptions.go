package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/jackie8tao/mihotui/internal/app/components"
	"github.com/jackie8tao/mihotui/internal/app/theme"
	"github.com/jackie8tao/mihotui/internal/config"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

const subscriptionsLeftPaneWidth = 36

// RenderSubscriptions renders the subscription management view.
func RenderSubscriptions(p SubscriptionsProps) string {
	var curr *config.Subscription
	if len(p.Subscriptions) > 0 && p.SelectedIdx >= 0 && p.SelectedIdx < len(p.Subscriptions) {
		curr = &p.Subscriptions[p.SelectedIdx]
	}

	baseView := components.SplitPane(p.Width, p.Height, subscriptionsLeftPaneWidth,
		func(w, h int) string {
			return renderSubscriptionsListPane(p, w, h)
		},
		func(w, h int) string {
			return renderSubscriptionDetailsPane(p, curr, w, h)
		},
	)

	if p.ModalActive == ModalAddSub {
		return renderAddSubModal(p, baseView)
	}
	return baseView
}

// renderSubscriptionsListPane renders the left sidebar listing managed subscriptions.
func renderSubscriptionsListPane(p SubscriptionsProps, width, height int) string {
	return components.Panel(components.PanelProps{
		Title:  fmt.Sprintf("Subscriptions (%d)", len(p.Subscriptions)),
		Active: true,
		Width:  width,
		Height: height,
	}, func(cw, ch int) string {
		return components.ScrollList(components.ListProps{
			Total:        len(p.Subscriptions),
			SelectedIdx:  p.SelectedIdx,
			Focused:      true,
			EmptyMessage: "No subscriptions added yet.\nPress 'a' to add one.",
			RenderRow: func(idx, rowW int, selected, _ bool) []components.Segment {
				sub := p.Subscriptions[idx]
				cursor := "  "
				if selected {
					cursor = "▶ "
				}
				activeSeg := components.Plain("  ")
				if sub.Name == p.ActiveSubName {
					activeSeg = components.Styled("✔ ", theme.ActiveIndicatorStyle)
				}
				const tag = "[SUB] "
				nameMax := max(4, rowW-4-len(tag))
				return []components.Segment{
					components.Plain(cursor),
					activeSeg,
					components.Styled(tag, theme.WarningStyle),
					components.Plain(components.TruncateVisual(sub.Name, nameMax)),
				}
			},
		}, cw, ch)
	})
}

// renderSubscriptionDetailsPane renders the right details panel for the selected subscription.
func renderSubscriptionDetailsPane(p SubscriptionsProps, curr *config.Subscription, width, height int) string {
	return components.Panel(components.PanelProps{
		Title:  "Subscription Details",
		Active: false,
		Width:  width,
		Height: height,
	}, func(cw, _ int) string {
		if curr == nil {
			return theme.MutedStyle.Width(max(1, cw)).Render("Select a subscription to view details, or press 'a' to add a new subscription.")
		}

		badgeText := "[INACTIVE]"
		badgeStyle := theme.MutedStyle
		if curr.Name == p.ActiveSubName {
			badgeText = "[ACTIVE]"
			badgeStyle = theme.ActiveIndicatorStyle
		}

		nameMax := max(6, cw-16-len(badgeText)-2)
		summary := components.KVList([]components.KVItem{
			{
				Key: "Subscription",
				Segments: []components.Segment{
					components.Styled(components.TruncateVisual(curr.Name, nameMax), theme.BoldStyle),
					components.Plain("  "),
					components.Styled(badgeText, badgeStyle),
				},
			},
			{Key: "Remote URL", Value: curr.URL},
			{Key: "Last Updated", Value: curr.UpdatedAt.Format("2006-01-02 15:04:05")},
		}, cw)

		quotaSection := renderSubscriptionQuota(curr.UserInfo, cw)

		actions := []string{
			components.SectionDivider("Available Actions", theme.MutedStyle, cw),
			components.TruncateVisual("• [u]     Update & re-download this subscription", cw),
			components.TruncateVisual("• [U]     Batch update all subscriptions", cw),
			components.TruncateVisual("• [a]     Add new subscription (modal)", cw),
			components.TruncateVisual("• [d]     Delete this subscription and its local file", cw),
			components.TruncateVisual("• [enter] Apply this subscription", cw),
		}

		return strings.Join([]string{summary, "", quotaSection, "", strings.Join(actions, "\n")}, "\n")
	})
}

// renderSubscriptionQuota renders traffic usage and expiration info from subscription headers.
func renderSubscriptionQuota(u *config.SubUserInfo, width int) string {
	header := components.SectionDivider("Traffic & Quota", theme.HighlightStyle, width)
	if u == nil {
		return header + "\n" + theme.MutedStyle.Render(components.TruncateVisual("(No quota header returned from subscription provider)", width))
	}

	used := u.Upload + u.Download
	total := u.Total
	var pct float64
	if total > 0 {
		pct = float64(used) / float64(total) * 100
	}

	items := []components.KVItem{
		{
			Key: "Usage Ratio",
			Segments: []components.Segment{
				components.Plain("["),
				components.Plain(components.ProgressBar(pct, 20)),
				components.Plain(fmt.Sprintf("] %.1f%%", pct)),
			},
		},
		{
			Key: "Used / Total",
			Value: fmt.Sprintf("%s / %s (↑ %s, ↓ %s)",
				mihomo.FormatBytes(used),
				mihomo.FormatBytes(total),
				mihomo.FormatBytes(u.Upload),
				mihomo.FormatBytes(u.Download),
			),
		},
	}

	if !u.Expire.IsZero() {
		days := int(time.Until(u.Expire).Hours() / 24)
		expireItem := components.KVItem{Key: "Expires On"}
		switch {
		case days < 0:
			expireItem.Value = fmt.Sprintf("%s (Expired %d days ago)", u.Expire.Format("2006-01-02"), -days)
			expireItem.ValueStyle = &theme.ErrorStyle
		case days <= 7:
			expireItem.Value = fmt.Sprintf("%s (%d days left)", u.Expire.Format("2006-01-02"), days)
			expireItem.ValueStyle = &theme.WarningStyle
		default:
			expireItem.Value = fmt.Sprintf("%s (%d days left)", u.Expire.Format("2006-01-02"), days)
		}
		items = append(items, expireItem)
	}

	return header + "\n" + components.KVList(items, width)
}

// renderAddSubModal renders the centered modal dialog for adding a new subscription over baseView.
func renderAddSubModal(p SubscriptionsProps, baseView string) string {
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		"Subscription Name:",
		p.SubNameInputView,
		"",
		"Subscription URL:",
		p.SubURLInputView,
	)

	return components.ModalOverlay(baseView, p.Width, p.Height, components.ModalProps{
		Title:          "Add New Subscription",
		Body:           body,
		Footer:         "[↑/↓] Switch Field | [Enter] Confirm & Download | [Esc] Cancel",
		PreferredWidth: 56,
	})
}
