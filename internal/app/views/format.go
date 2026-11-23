package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TruncateVisual sanitizes ambiguous Unicode sequences and truncates s to at most
// maxWidth terminal display columns, preserving multi-byte UTF-8 runes.
func TruncateVisual(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	s = sanitizeDisplayText(s)
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
	return string(runes[:end]) + "..."
}

func truncateStr(s string, maxWidth int) string {
	return TruncateVisual(s, maxWidth)
}

// padRightVisual sanitizes and truncates s to width display columns, then right-pads
// with spaces so columns align accurately regardless of CJK or Emoji character widths.
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

// sanitizeDisplayText normalizes Unicode sequences whose rendered cell width differs
// across terminals (such as Windows Terminal / SSH clients vs lipgloss/uniseg):
//   - Regional-indicator flag pairs (e.g. 🇨🇳) -> deterministic ASCII country code ([CN])
//   - Variation selectors (U+FE0E/U+FE0F) and zero-width joiners (U+200D) -> stripped
func sanitizeDisplayText(s string) string {
	runes := []rune(s)
	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if isRegionalIndicator(r) {
			if i+1 < len(runes) && isRegionalIndicator(runes[i+1]) {
				b.WriteByte('[')
				b.WriteByte(byte('A' + (r - 0x1F1E6)))
				b.WriteByte(byte('A' + (runes[i+1] - 0x1F1E6)))
				b.WriteByte(']')
				i++
			}
			continue
		}
		if r == 0xFE0E || r == 0xFE0F || r == 0x200D {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isRegionalIndicator(r rune) bool {
	return r >= 0x1F1E6 && r <= 0x1F1FF
}
