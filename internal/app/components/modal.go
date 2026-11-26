package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"

	"github.com/jackie8tao/mihotui/internal/app/theme"
)

// ModalProps defines the structured sections of a centered modal dialog.
type ModalProps struct {
	Title          string
	Body           string
	Footer         string
	PreferredWidth int
}

// ClampModalWidth returns preferred clamped to fit comfortably within containerWidth.
func ClampModalWidth(preferred, containerWidth int) int {
	w := preferred
	if maxW := containerWidth - 4; maxW > 0 && w > maxW {
		w = maxW
	}
	if w < 24 {
		w = 24
	}
	return w
}

// ModalOverlay wraps structured modal content in theme.ModalBoxStyle and floats it
// centered over baseView while dimming the underlying screen.
func ModalOverlay(baseView string, width, height int, p ModalProps) string {
	modalWidth := ClampModalWidth(p.PreferredWidth, width)
	// theme.ModalBoxStyle has Padding(1, 2), leaving modalWidth - 4 columns for text.
	innerW := modalWidth - 4
	if innerW < 16 {
		innerW = 16
	}

	var sections []string
	if p.Title != "" {
		sections = append(sections, theme.ModalTitleStyle.Render(TruncateVisual(p.Title, innerW)))
	}
	if p.Body != "" {
		if len(sections) > 0 {
			sections = append(sections, "")
		}
		sections = append(sections, p.Body)
	}
	if p.Footer != "" {
		if len(sections) > 0 {
			sections = append(sections, "")
		}
		sections = append(sections, theme.MutedStyle.Render(TruncateVisual(p.Footer, innerW)))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	modal := theme.ModalBoxStyle.Width(modalWidth).Render(content)
	return OverlayCenter(baseView, modal, width, height)
}

// OverlayCenter composites overlay centered on top of base using a 2D terminal cell buffer.
// Non-empty background cells outside the overlay are dimmed so the floating dialog stands out,
// while ANSI styles and wide characters (CJK) around the edges stay aligned.
func OverlayCenter(base, overlay string, width, height int) string {
	baseW, baseH := lipgloss.Width(base), lipgloss.Height(base)
	overlayW, overlayH := lipgloss.Width(overlay), lipgloss.Height(overlay)

	canvasW := width
	if canvasW <= 0 {
		canvasW = max(baseW, overlayW)
	}
	canvasH := height
	if canvasH <= 0 {
		canvasH = max(baseH, overlayH)
	}
	if canvasW <= 0 || canvasH <= 0 {
		return overlay
	}

	buf := cellbuf.NewBuffer(canvasW, canvasH)
	cellbuf.SetContent(buf, base)

	// Dim visible background cells so the centered modal pops forward without bloating
	// empty padding cells with unnecessary ANSI attribute toggles.
	for y := 0; y < canvasH; y++ {
		for x := 0; x < canvasW; x++ {
			c := buf.Cell(x, y)
			if c == nil || c.Rune == 0 || c.Rune == ' ' {
				continue
			}
			c.Style.Bold(false)
			c.Style.Faint(true)
		}
	}

	x := (canvasW - overlayW) / 2
	if x < 0 {
		x = 0
	}
	y := (canvasH - overlayH) / 2
	if y < 0 {
		y = 0
	}

	rect := cellbuf.Rect(x, y, min(overlayW, canvasW-x), min(overlayH, canvasH-y))
	cellbuf.SetContentRect(buf, overlay, rect)

	// Normalize CRLF from cellbuf.Render to LF for Bubble Tea / Lipgloss compatibility.
	return strings.ReplaceAll(cellbuf.Render(buf), "\r\n", "\n")
}
