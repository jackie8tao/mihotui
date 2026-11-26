package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// SanitizeDisplayText normalizes display text for deterministic single-line terminal rendering:
//   - Regional-indicator flag pairs (e.g. U+1F1E8 U+1F1F3) -> ASCII country code ([CN])
//   - Emoji symbols, pictographs, skin-tone modifiers, ZWJ, variation selectors, and tag chars -> stripped
//   - Line breaks and tabs (\r, \n, \t) -> single space
//   - Spaces left orphaned adjacent to stripped emojis are collapsed cleanly
func SanitizeDisplayText(s string) string {
	runes := []rune(s)
	var b strings.Builder
	b.Grow(len(s))

	skipNextSpace := false
	lastWasSpace := false
	endedWithStrippedEmoji := false

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Convert paired Regional Indicator Symbols into deterministic [XX] country codes.
		if isRegionalIndicator(r) {
			if i+1 < len(runes) && isRegionalIndicator(runes[i+1]) {
				b.WriteByte('[')
				b.WriteByte(byte('A' + (r - 0x1F1E6)))
				b.WriteByte(byte('A' + (runes[i+1] - 0x1F1E6)))
				b.WriteByte(']')
				i++
				skipNextSpace = false
				lastWasSpace = false
				endedWithStrippedEmoji = false
			} else {
				skipNextSpace = b.Len() == 0 || lastWasSpace
				endedWithStrippedEmoji = true
			}
			continue
		}

		// Strip emoji codepoints and Unicode emoji joiner/modifier sequences.
		if isEmojiRune(r) {
			skipNextSpace = b.Len() == 0 || lastWasSpace
			endedWithStrippedEmoji = true
			continue
		}

		// Normalize control whitespace so single-line cells never break panel height.
		if r == '\r' || r == '\n' || r == '\t' {
			r = ' '
		}

		if r == ' ' && skipNextSpace {
			continue
		}

		b.WriteRune(r)
		skipNextSpace = false
		lastWasSpace = r == ' '
		endedWithStrippedEmoji = false
	}

	out := b.String()
	if endedWithStrippedEmoji {
		out = strings.TrimRight(out, " ")
	}
	return out
}

// TruncateVisual sanitizes s and truncates it to at most maxWidth terminal display columns,
// preserving multi-byte UTF-8 and CJK grapheme clusters.
func TruncateVisual(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	s = SanitizeDisplayText(s)
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	if maxWidth <= 3 {
		return strings.Repeat(".", maxWidth)
	}
	return ansi.Truncate(s, maxWidth, "...")
}

// PadRightVisual sanitizes and truncates s to width display columns, then right-pads
// with spaces so columns align accurately across ASCII and CJK characters.
func PadRightVisual(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = TruncateVisual(s, width)
	if pad := width - lipgloss.Width(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

// PadLeftVisual sanitizes and truncates s to width display columns, then left-pads
// with spaces for right-aligned numeric/status columns.
func PadLeftVisual(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = TruncateVisual(s, width)
	if pad := width - lipgloss.Width(s); pad > 0 {
		return strings.Repeat(" ", pad) + s
	}
	return s
}

// isRegionalIndicator reports whether r is a Unicode Regional Indicator Symbol Letter (A-Z).
func isRegionalIndicator(r rune) bool {
	return r >= 0x1F1E6 && r <= 0x1F1FF
}

// isEmojiRune reports whether r belongs to Unicode emoji/pictograph blocks or emoji
// formatting sequences whose terminal cell widths vary across fonts and SSH clients.
// Standard TUI symbols intentionally used by components (box drawing, arrows, geometric
// markers, and U+2714 heavy check mark) are preserved.
func isEmojiRune(r rune) bool {
	switch {
	case r == 0x2714: // Keep heavy check mark used as active status indicator.
		return false
	case r == 0xFE0E || r == 0xFE0F || r == 0x200D || r == 0x20E3:
		// Variation selectors, zero-width joiner, and combining enclosing keycap.
		return true
	case r >= 0xE0020 && r <= 0xE007F:
		// Tag characters used in subdivision flag sequences.
		return true
	case r >= 0x1F000 && r <= 0x1FAFF:
		// Supplementary plane pictographs, emoticons, transport, flags, skin-tone modifiers.
		return true
	case r >= 0x2600 && r <= 0x26FF:
		// Miscellaneous symbols (weather, warning, lightning, gear, etc.).
		return true
	case r >= 0x2700 && r <= 0x27BF:
		// Dingbats (sparkles, airplane, cross, heart, plus, etc.).
		return true
	case (r >= 0x231A && r <= 0x231B) || (r >= 0x23E9 && r <= 0x23FA):
		// Watch, hourglass, and media control emoji symbols.
		return true
	case r == 0x2B50 || r == 0x2B55 || r == 0x2934 || r == 0x2935 ||
		r == 0x3030 || r == 0x303D || r == 0x3297 || r == 0x3299:
		return true
	default:
		return false
	}
}
