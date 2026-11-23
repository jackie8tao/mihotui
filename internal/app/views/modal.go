package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/cellbuf"

	"github.com/jackie8tao/mihotui/internal/app/theme"
)

// clampModalWidth returns preferred clamped to fit comfortably within containerWidth.
func clampModalWidth(preferred, containerWidth int) int {
	w := preferred
	if maxW := containerWidth - 4; maxW > 0 && w > maxW {
		w = maxW
	}
	if w < 24 {
		w = 24
	}
	return w
}

// renderModalOverlay wraps content in the standard modal frame and floats it centered over baseView.
func renderModalOverlay(baseView string, width, height, modalWidth int, content string) string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(theme.ColorPrimary).
		BorderBackground(theme.ColorDark).
		Background(theme.ColorDark).
		Padding(1, 2).
		Width(modalWidth)

	modal := modalStyle.Render(content)
	return overlayCenter(baseView, modal, width, height)
}

// overlayCenter composites overlay centered on top of base using a 2D terminal cell buffer.
// Background cells outside the overlay are dimmed so the floating dialog stands out clearly,
// while ANSI styles and wide characters (CJK/Emoji) around the edges stay aligned.
func overlayCenter(base, overlay string, width, height int) string {
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

	// Dim the background layer so the centered modal visually pops forward.
	for y := 0; y < canvasH; y++ {
		for x := 0; x < canvasW; x++ {
			if c := buf.Cell(x, y); c != nil {
				c.Style.Bold(false)
				c.Style.Faint(true)
			}
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
