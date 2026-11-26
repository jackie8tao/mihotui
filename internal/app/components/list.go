package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jackie8tao/mihotui/internal/app/theme"
)

// Segment represents a styled span within a single row of a list or key-value view.
type Segment struct {
	Text     string
	Style    lipgloss.Style
	hasStyle bool
}

// Plain creates an unstyled text segment that inherits the row's default or selected style.
func Plain(text string) Segment {
	return Segment{Text: text}
}

// Styled creates a text segment with an explicit foreground/weight style.
// When rendered inside a focused selected row, the highlight background is automatically
// preserved across the segment so ANSI reset sequences never break the highlight bar.
func Styled(text string, style lipgloss.Style) Segment {
	return Segment{Text: text, Style: style, hasStyle: true}
}

// ListProps configures a stateless virtual-scrolling list component.
type ListProps struct {
	Total        int
	SelectedIdx  int
	Focused      bool
	EmptyMessage string
	// RenderRow builds the segments for row idx within available visual width.
	RenderRow func(idx, width int, selected, focused bool) []Segment
}

// ScrollWindow computes the inclusive start and exclusive end indices for a vertical
// scrolling viewport of visible rows containing selected.
func ScrollWindow(total, selected, visible int) (start, end int) {
	if total <= 0 || visible <= 0 {
		return 0, 0
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= total {
		selected = total - 1
	}
	if selected >= visible {
		start = selected - visible + 1
	}
	end = start + visible
	if end > total {
		end = total
		start = end - visible
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

// ScrollList renders the visible slice of a list with consistent cursor highlighting
// and full-width selection bars.
func ScrollList(p ListProps, width, height int) string {
	if p.Total <= 0 {
		if p.EmptyMessage == "" {
			return ""
		}
		return theme.MutedStyle.Width(max(1, width)).Render(p.EmptyMessage)
	}
	if height <= 0 {
		height = p.Total
	}

	start, end := ScrollWindow(p.Total, p.SelectedIdx, height)
	lines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		selected := i == p.SelectedIdx
		segs := p.RenderRow(i, width, selected, p.Focused)
		lines = append(lines, RenderRowSegments(segs, width, selected && p.Focused))
	}
	return strings.Join(lines, "\n")
}

// RenderRowSegments renders a horizontal sequence of segments into a single row.
// When highlight is true, every segment and trailing padding receive theme.ColorSelectedBg
// so multi-colored rows maintain a continuous selection bar without ANSI resets breaking it.
func RenderRowSegments(segments []Segment, width int, highlight bool) string {
	var b strings.Builder
	totalW := 0

	for _, seg := range segments {
		if seg.Text == "" {
			continue
		}
		totalW += lipgloss.Width(seg.Text)
		if highlight {
			if seg.hasStyle {
				b.WriteString(seg.Style.Background(theme.ColorSelectedBg).Render(seg.Text))
			} else {
				b.WriteString(theme.SelectedRowStyle.Render(seg.Text))
			}
			continue
		}

		if seg.hasStyle {
			b.WriteString(seg.Style.Render(seg.Text))
		} else {
			b.WriteString(seg.Text)
		}
	}

	if highlight && width > totalW {
		b.WriteString(theme.SelectedRowStyle.Render(strings.Repeat(" ", width-totalW)))
	}
	return b.String()
}
