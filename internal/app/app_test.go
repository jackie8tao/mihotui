package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAppModel_InitialState(t *testing.T) {
	m := NewAppModel(nil, nil, nil)

	if m.activeTab != TabOverview {
		t.Errorf("expected activeTab to be TabOverview, got %v", m.activeTab)
	}
	if m.proxiesState.FocusPane != PaneGroups {
		t.Errorf("expected proxiesState.FocusPane to be PaneGroups, got %v", m.proxiesState.FocusPane)
	}
	if m.subsState.ModalActive != ModalNone {
		t.Errorf("expected subsState.ModalActive to be ModalNone, got %v", m.subsState.ModalActive)
	}
	if m.configsState.Cursor != 0 {
		t.Errorf("expected configsState.Cursor to be 0, got %d", m.configsState.Cursor)
	}
}

func TestAppModel_TabSwitching(t *testing.T) {
	m := NewAppModel(nil, nil, nil)

	// Simulate pressing Tab key
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	model := updated.(*AppModel)
	if model.activeTab != TabProxies {
		t.Errorf("expected activeTab to switch to TabProxies, got %v", model.activeTab)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(*AppModel)
	if model.activeTab != TabConfigs {
		t.Errorf("expected activeTab to switch to TabConfigs, got %v", model.activeTab)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(*AppModel)
	if model.activeTab != TabSubscriptions {
		t.Errorf("expected activeTab to switch to TabSubscriptions, got %v", model.activeTab)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(*AppModel)
	if model.activeTab != TabOverview {
		t.Errorf("expected activeTab to cycle back to TabOverview, got %v", model.activeTab)
	}
}
