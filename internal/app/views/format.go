package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// truncateStr truncates s to at most maxWidth terminal display columns,
// preserving multi-byte UTF-8 runes and regional-indicator flag pairs.
func truncateStr(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return strings.Repeat(".", maxWidth)
	}

	limit := maxWidth - 3
	runes := []rune(s)
	end := len(runes)
	for end > 0 && lipgloss.Width(string(runes[:end])) > limit {
		end--
	}
	// Avoid leaving an orphaned regional-indicator rune (half of a flag emoji).
	if end > 0 && isRegionalIndicator(runes[end-1]) {
		count := 0
		for i := end - 1; i >= 0 && isRegionalIndicator(runes[i]); i-- {
			count++
		}
		if count%2 == 1 {
			end--
		}
	}
	return string(runes[:end]) + "..."
}

// padRightVisual truncates s to width display columns and right-pads with spaces
// so columns align accurately regardless of CJK or Emoji character widths.
func padRightVisual(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = truncateStr(s, width)
	if pad := width - lipgloss.Width(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

func isRegionalIndicator(r rune) bool {
	return r >= 0x1F1E6 && r <= 0x1F1FF
}
