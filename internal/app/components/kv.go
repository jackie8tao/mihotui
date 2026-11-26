package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jackie8tao/mihotui/internal/app/theme"
)

// KVItem represents a single aligned key-value row in a KVList.
// When Segments is non-empty, it takes precedence over Value/ValueStyle.
type KVItem struct {
	Key        string
	Value      string
	ValueStyle *lipgloss.Style
	Segments   []Segment
}

// KVList renders vertically aligned "Key : Value" rows clamped cleanly to width columns.
func KVList(items []KVItem, width int) string {
	if len(items) == 0 {
		return ""
	}

	maxKeyW := 0
	for _, it := range items {
		if w := lipgloss.Width(SanitizeDisplayText(it.Key)); w > maxKeyW {
			maxKeyW = w
		}
	}

	const sep = " : "
	valWidth := width - maxKeyW - len(sep)
	if valWidth < 4 {
		valWidth = 4
	}

	lines := make([]string, 0, len(items))
	for _, it := range items {
		prefix := PadRightVisual(it.Key, maxKeyW) + sep
		if len(it.Segments) > 0 {
			lines = append(lines, prefix+RenderRowSegments(it.Segments, valWidth, false))
			continue
		}

		val := TruncateVisual(it.Value, valWidth)
		if it.ValueStyle != nil {
			val = it.ValueStyle.Render(val)
		}
		lines = append(lines, prefix+val)
	}
	return strings.Join(lines, "\n")
}

// SectionDivider renders a single-line section heading divider clamped to width.
func SectionDivider(title string, style lipgloss.Style, width int) string {
	label := TruncateVisual("─── "+title+" ───", width)
	return style.Render(label)
}

// ProgressBar renders an ASCII block progress bar safely clamped to [0, 100] percent.
func ProgressBar(pct float64, width int) string {
	if width < 1 {
		width = 1
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}

	filled := int(float64(width) * (pct / 100))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	switch {
	case pct >= 95:
		return theme.ErrorStyle.Render(bar)
	case pct >= 80:
		return theme.WarningStyle.Render(bar)
	default:
		return theme.SpeedDownStyle.Render(bar)
	}
}
