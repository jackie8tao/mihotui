package views

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/jackie8tao/mihotui/internal/config"
)

func TestOverlayCenterPreservesBackgroundAndCentersModal(t *testing.T) {
	bgLine := strings.Repeat("BG", 40) // 80 cols
	bgLines := make([]string, 20)
	for i := range bgLines {
		bgLines[i] = bgLine
	}
	base := strings.Join(bgLines, "\n")

	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		Width(20).
		Render("HELLO MODAL")

	out := overlayCenter(base, modal, 80, 20)

	if strings.Contains(out, "\r") {
		t.Fatalf("expected LF line endings only, found CR in output")
	}

	lines := strings.Split(out, "\n")
	if len(lines) != 20 {
		t.Fatalf("expected 20 lines, got %d", len(lines))
	}

	plain := ansi.Strip(out)
	if !strings.Contains(plain, "HELLO MODAL") {
		t.Fatalf("expected modal text in composited output")
	}
	// Top row and bottom row should still show background content.
	if !strings.Contains(ansi.Strip(lines[0]), "BGBG") {
		t.Fatalf("expected top background row to remain visible, got %q", ansi.Strip(lines[0]))
	}
	if !strings.Contains(ansi.Strip(lines[19]), "BGBG") {
		t.Fatalf("expected bottom background row to remain visible, got %q", ansi.Strip(lines[19]))
	}

	// Find the modal line and confirm background text remains visible on both left and right sides.
	foundRow := false
	for _, l := range lines {
		stripped := ansi.Strip(l)
		if strings.Contains(stripped, "HELLO MODAL") {
			foundRow = true
			if !strings.HasPrefix(stripped, "BGBG") || !strings.HasSuffix(stripped, "BGBG") {
				t.Fatalf("expected background visible on left and right of modal row, got %q", stripped)
			}
			if w := lipgloss.Width(l); w > 80 {
				t.Fatalf("expected line width <= 80, got %d", w)
			}
		}
	}
	if !foundRow {
		t.Fatalf("modal row not found")
	}
}

func TestOverlayCenterHandlesWideCharsAtEdges(t *testing.T) {
	// 30 CJK chars = 60 display columns
	bgLine := strings.Repeat("中", 30)
	bgLines := make([]string, 12)
	for i := range bgLines {
		bgLines[i] = bgLine
	}
	base := strings.Join(bgLines, "\n")

	// Odd width (21 cols including border) intentionally forces boundary intersection with 2-col CJK chars.
	modal := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		Width(19).
		Render("MODAL")

	out := overlayCenter(base, modal, 60, 12)
	for i, line := range strings.Split(out, "\n") {
		if w := lipgloss.Width(line); w > 60 {
			t.Fatalf("line %d exceeded canvas width 60: got %d (%q)", i, w, ansi.Strip(line))
		}
	}
	if !strings.Contains(ansi.Strip(out), "MODAL") {
		t.Fatalf("expected modal text to be present")
	}
}

func TestRenderConfigsEditModalOverlaysBaseView(t *testing.T) {
	out := RenderConfigs(ConfigsProps{
		Width:          100,
		Height:         24,
		FocusPane:      ConfigPaneFields,
		Sections:       []ConfigSectionView{{Title: "TUN Mode", OverrideCount: 1}},
		SelectedSecIdx: 0,
		ActiveSecTitle: "TUN Mode",
		Fields: []ConfigFieldView{
			{
				Label:        "Device Name",
				YAMLPath:     "tun.device",
				Description:  "Virtual interface name.",
				KindHint:     "string",
				DisplayValue: "mihomo",
				IsOverridden: true,
			},
		},
		SelectedFieldIdx: 0,
		OverridePath:     "/etc/mihotui/override.yaml",
		ActiveSubName:    "my-sub",
		Editing:          true,
		EditLabel:        "Device Name",
		EditYAMLPath:     "tun.device",
		EditDescription:  "Virtual interface name.",
		EditInputView:    "> mihomo",
	})

	plain := ansi.Strip(out)
	if !strings.Contains(plain, "Edit Device Name (tun.device)") {
		t.Fatalf("expected edit modal title in output")
	}
	if !strings.Contains(plain, "Sections") {
		t.Fatalf("expected background 'Sections' header to remain visible behind overlay")
	}
}

func TestRenderSubscriptionsAddModalOverlaysBaseView(t *testing.T) {
	out := RenderSubscriptions(SubscriptionsProps{
		Width:  100,
		Height: 24,
		Subscriptions: []config.Subscription{
			{
				Name:      "airport-1",
				URL:       "https://example.com/sub",
				UpdatedAt: time.Now(),
			},
		},
		ActiveSubName:    "airport-1",
		SelectedIdx:      0,
		ModalActive:      ModalAddSub,
		SubNameInputView: "> demo",
		SubURLInputView:  "> https://example.com",
	})

	plain := ansi.Strip(out)
	if !strings.Contains(plain, "Add New Subscription") {
		t.Fatalf("expected add-subscription modal title in output")
	}
	if !strings.Contains(plain, "Subscriptions (1)") {
		t.Fatalf("expected background 'Subscriptions (1)' header to remain visible behind overlay")
	}
}
