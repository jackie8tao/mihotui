package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jackie8tao/mihotui/internal/app/theme"
)

const (
	// Lipgloss box model geometry for theme.CardBoxStyle / theme.CardBoxActiveStyle:
	//   - Border adds 2 columns horizontally and 2 rows vertically outside Style.Width/Height.
	//   - Padding(0, 1) reserves 2 columns horizontally inside Style.Width.
	//
	// Outer Width = p.Width
	// ┌────────────────────────┐  <- top border (1 row)
	// │ Title [Badge]          │  <- header (1 row), content width = p.Width - 4
	// │                        │  <- spacer (1 row)
	// │ <body content>         │  <- content height = p.Height - 4 (when Title != "")
	// └────────────────────────┘  <- bottom border (1 row)
	// Outer Height = p.Height
	panelBorderWidth  = 2
	panelBorderHeight = 2
	panelHorizPadding = 2
	panelHeaderRows   = 2 // Title line + blank line below title
	splitPaneGap      = 2 // Two-space column gutter between side-by-side panes
)

// PanelProps configures a bordered card container with an optional bold header and badge.
// Width and Height specify the final outer dimensions including borders; Panel deducts
// border, padding, and header space before passing safe (innerW, innerH) to body.
type PanelProps struct {
	Title  string
	Badge  string
	Active bool
	Width  int
	Height int // <= 0 lets the panel grow vertically to fit its content
}

// Panel renders a bordered card container and invokes body with the safe inner content
// dimensions (innerW, innerH). When Height > 0, body output is clamped to innerH lines so
// the bottom border is never pushed out or clipped by the viewport.
func Panel(p PanelProps, body func(innerW, innerH int) string) string {
	boxW := p.Width - panelBorderWidth
	if boxW < panelHorizPadding+1 {
		boxW = panelHorizPadding + 1
	}
	innerW := boxW - panelHorizPadding
	if innerW < 1 {
		innerW = 1
	}

	hasHeader := p.Title != "" || p.Badge != ""
	headerRows := 0
	if hasHeader {
		headerRows = panelHeaderRows
	}

	boxH := 0
	innerH := 0
	if p.Height > 0 {
		boxH = p.Height - panelBorderHeight
		if boxH < headerRows+1 {
			boxH = headerRows + 1
		}
		innerH = boxH - headerRows
		if innerH < 1 {
			innerH = 1
		}
	}

	content := ""
	if body != nil {
		content = body(innerW, innerH)
	}
	if innerH > 0 {
		content = clampLines(content, innerH)
	}

	var rendered string
	if hasHeader {
		header := renderPanelHeader(p.Title, p.Badge, innerW)
		rendered = header + "\n\n" + content
	} else {
		rendered = content
	}

	style := theme.CardBoxStyle
	if p.Active {
		style = theme.CardBoxActiveStyle
	}
	style = style.Width(boxW)
	if boxH > 0 {
		style = style.Height(boxH)
	}
	return style.Render(rendered)
}

// SplitPane lays out two side-by-side panes strictly within totalW columns and totalH rows.
// preferredLeftW is gracefully scaled down on narrow terminals so the combined outer width
// never exceeds totalW.
func SplitPane(totalW, totalH, preferredLeftW int, left, right func(w, h int) string) string {
	availW := totalW
	if availW < 16 {
		availW = 16
	}

	leftW := preferredLeftW
	maxLeft := (availW - splitPaneGap) / 2
	if leftW > maxLeft {
		leftW = maxLeft
	}
	if leftW < 8 {
		leftW = 8
	}

	rightW := availW - leftW - splitPaneGap
	if rightW < 8 {
		rightW = 8
	}

	leftOut := ""
	if left != nil {
		leftOut = left(leftW, totalH)
	}
	rightOut := ""
	if right != nil {
		rightOut = right(rightW, totalH)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, leftOut, strings.Repeat(" ", splitPaneGap), rightOut)
}

// renderPanelHeader formats a single-line bold title with an optional styled badge suffix.
func renderPanelHeader(title, badge string, maxWidth int) string {
	if badge == "" {
		return theme.BoldStyle.Render(TruncateVisual(title, maxWidth))
	}

	badgeW := lipgloss.Width(badge)
	availTitleW := maxWidth - badgeW - 1
	if availTitleW < 1 {
		return TruncateVisual(title, maxWidth)
	}
	return theme.BoldStyle.Render(TruncateVisual(title, availTitleW)) + " " + badge
}

// clampLines truncates s to at most maxLines newline-separated lines.
func clampLines(s string, maxLines int) string {
	if maxLines <= 0 || s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n")
}
