package pages

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jackie8tao/mihotui/internal/app/events"
	"github.com/jackie8tao/mihotui/internal/app/keymaps"
	"github.com/jackie8tao/mihotui/internal/app/views"
	"github.com/jackie8tao/mihotui/internal/profile"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

type (
	overrideLoadedMsg struct {
		doc *profile.OverrideDoc
		err error
	}
	overrideAppliedMsg struct {
		reloaded  bool
		activeSub string
		err       error
	}
)

// ConfigsModel is a schema-driven editor for user system overrides stored in override.yaml.
type ConfigsModel struct {
	client     mihomo.Client
	profileMgr *profile.Manager
	sections   []SectionSpec
	doc        *profile.OverrideDoc

	focusPane  views.ConfigFocusPane
	sectionIdx int
	fieldIdx   int
	dirty      bool

	editing   bool
	editInput textinput.Model
}

// NewConfigsModel initializes a self-contained ConfigsModel.
func NewConfigsModel(cli mihomo.Client, mgr *profile.Manager) *ConfigsModel {
	if cli == nil {
		panic("pages: nil mihomo client")
	}
	if mgr == nil {
		panic("pages: nil profile manager")
	}

	input := textinput.New()
	input.Width = 48

	return &ConfigsModel{
		client:     cli,
		profileMgr: mgr,
		sections:   defaultConfigSections(),
		doc:        profile.NewOverrideDoc(),
		focusPane:  views.ConfigPaneFields,
		editInput:  input,
	}
}

// Title returns the tab header title for the Configs page.
func (m *ConfigsModel) Title() string {
	return "3. Configs"
}

// Help returns the keybinding hints displayed in the footer for the Configs page.
func (m *ConfigsModel) Help() string {
	if m.editing {
		return "[Enter] Confirm | [Esc] Cancel"
	}
	return "[←/→] Pane | [↑/↓] Move | [Enter] Edit | [x] Reset | [s] Save & Apply"
}

// CapturingInput reports whether the field edit modal is currently capturing keyboard input.
func (m *ConfigsModel) CapturingInput() bool {
	return m.editing
}

// Init loads override.yaml from disk.
func (m *ConfigsModel) Init() tea.Cmd {
	return m.loadOverrideCmd()
}

// Update handles override loading, save & merge results, and keyboard navigation/editing.
func (m *ConfigsModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case overrideLoadedMsg:
		if msg.err != nil {
			return events.StatusCmd(fmt.Sprintf("Load override.yaml failed: %v", msg.err), true)
		}
		m.doc = msg.doc
		m.dirty = false
		m.editing = false
		return nil

	case overrideAppliedMsg:
		if msg.err != nil {
			return events.StatusCmd(fmt.Sprintf("Apply override failed: %v", msg.err), true)
		}
		m.dirty = false
		if !msg.reloaded {
			return events.StatusCmd("Saved override.yaml (no active subscription to reload)", false)
		}
		return tea.Batch(
			events.StatusCmd(fmt.Sprintf("Saved override.yaml & hot-reloaded with [%s]!", msg.activeSub), false),
			events.ProfileAppliedCmd(),
		)

	case tea.KeyMsg:
		if m.editing {
			return m.updateEditModal(msg)
		}
		return m.handleKeyMsg(msg)
	}

	return nil
}

// View renders the Configs page within the given terminal dimensions.
func (m *ConfigsModel) View(width, height int) string {
	secViews := make([]views.ConfigSectionView, len(m.sections))
	for i, sec := range m.sections {
		count := 0
		for _, f := range sec.Fields {
			if _, ok := m.doc.Get(f.Path...); ok {
				count++
			}
		}
		secViews[i] = views.ConfigSectionView{
			Title:         sec.Title,
			OverrideCount: count,
		}
	}

	var activeSecTitle string
	var fieldViews []views.ConfigFieldView
	var yamlPreview string
	if sec := m.currentSection(); sec != nil {
		activeSecTitle = sec.Title
		yamlPreview = m.doc.MarshalSection(sec.Key)
		fieldViews = make([]views.ConfigFieldView, len(sec.Fields))
		for i, f := range sec.Fields {
			val, ok := m.doc.Get(f.Path...)
			display := "(inherit)"
			if ok {
				display = f.FormatDisplay(val)
			}
			fieldViews[i] = views.ConfigFieldView{
				Label:        f.Label,
				YAMLPath:     f.YAMLPath(),
				Description:  f.Description,
				KindHint:     f.KindHint(),
				DisplayValue: display,
				IsOverridden: ok,
			}
		}
	}

	var editLabel, editPath, editDesc string
	if m.editing {
		if f := m.currentField(); f != nil {
			editLabel = f.Label
			editPath = f.YAMLPath()
			editDesc = f.Description
		}
	}

	return views.RenderConfigs(views.ConfigsProps{
		Width:            width,
		Height:           height,
		FocusPane:        m.focusPane,
		Sections:         secViews,
		SelectedSecIdx:   m.sectionIdx,
		ActiveSecTitle:   activeSecTitle,
		Fields:           fieldViews,
		SelectedFieldIdx: m.fieldIdx,
		YAMLPreview:      yamlPreview,
		OverridePath:     m.profileMgr.OverridePath(),
		MihomoConfigPath: m.profileMgr.MihomoConfigPath(),
		ActiveSubName:    m.profileMgr.ActiveProfile(),
		Dirty:            m.dirty,
		Editing:          m.editing,
		EditLabel:        editLabel,
		EditYAMLPath:     editPath,
		EditDescription:  editDesc,
		EditInputView:    m.editInput.View(),
	})
}

func (m *ConfigsModel) currentSection() *SectionSpec {
	if len(m.sections) == 0 || m.sectionIdx < 0 || m.sectionIdx >= len(m.sections) {
		return nil
	}
	return &m.sections[m.sectionIdx]
}

func (m *ConfigsModel) currentField() *FieldSpec {
	sec := m.currentSection()
	if sec == nil || len(sec.Fields) == 0 || m.fieldIdx < 0 || m.fieldIdx >= len(sec.Fields) {
		return nil
	}
	return &sec.Fields[m.fieldIdx]
}

func (m *ConfigsModel) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, keymaps.Left):
		m.focusPane = views.ConfigPaneSections

	case key.Matches(msg, keymaps.Right):
		if len(m.sections) > 0 {
			m.focusPane = views.ConfigPaneFields
		}

	case key.Matches(msg, keymaps.Up):
		if m.focusPane == views.ConfigPaneSections {
			if m.sectionIdx > 0 {
				m.sectionIdx--
				m.fieldIdx = 0
			}
		} else if m.fieldIdx > 0 {
			m.fieldIdx--
		}

	case key.Matches(msg, keymaps.Down):
		if m.focusPane == views.ConfigPaneSections {
			if m.sectionIdx < len(m.sections)-1 {
				m.sectionIdx++
				m.fieldIdx = 0
			}
		} else if sec := m.currentSection(); sec != nil && m.fieldIdx < len(sec.Fields)-1 {
			m.fieldIdx++
		}

	case key.Matches(msg, keymaps.Enter):
		if m.focusPane != views.ConfigPaneFields {
			return nil
		}
		return m.triggerFieldEdit()

	case key.Matches(msg, keymaps.ResetField):
		if m.focusPane != views.ConfigPaneFields {
			return nil
		}
		field := m.currentField()
		if field == nil {
			return nil
		}
		if m.doc.Delete(field.Path...) {
			m.dirty = true
			return events.StatusCmd(fmt.Sprintf("Reset [%s] to inherit (press 's' to save)", field.YAMLPath()), false)
		}

	case key.Matches(msg, keymaps.SaveConfig):
		snapshot := m.doc.Clone()
		return tea.Batch(
			events.StatusCmd("Saving override.yaml and merging Mihomo config...", false),
			m.saveAndApplyCmd(snapshot),
		)
	}

	return nil
}

func (m *ConfigsModel) triggerFieldEdit() tea.Cmd {
	field := m.currentField()
	if field == nil {
		return nil
	}

	switch field.Kind {
	case KindBool:
		cur, exists := m.doc.Get(field.Path...)
		next := true
		if exists {
			if b, ok := cur.(bool); ok {
				next = !b
			}
		}
		m.doc.Set(next, field.Path...)
		m.dirty = true
		return nil

	case KindSelect:
		if len(field.Options) == 0 {
			return nil
		}
		cur, exists := m.doc.Get(field.Path...)
		next := field.Options[0]
		if exists {
			curStr := fmt.Sprintf("%v", cur)
			for i, opt := range field.Options {
				if opt == curStr {
					next = field.Options[(i+1)%len(field.Options)]
					break
				}
			}
		}
		m.doc.Set(next, field.Path...)
		m.dirty = true
		return nil

	case KindInt, KindString, KindStringList:
		cur, exists := m.doc.Get(field.Path...)
		m.editInput.Placeholder = field.Placeholder
		if exists {
			m.editInput.SetValue(field.FormatEditValue(cur))
		} else {
			m.editInput.SetValue("")
		}
		m.editInput.CursorEnd()
		m.editInput.Focus()
		m.editing = true
		return nil
	}

	return nil
}

func (m *ConfigsModel) updateEditModal(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, keymaps.Esc):
		m.editing = false
		m.editInput.Blur()
		return nil

	case key.Matches(msg, keymaps.Enter):
		field := m.currentField()
		if field == nil {
			m.editing = false
			return nil
		}
		val, err := field.ParseInput(m.editInput.Value())
		if err != nil {
			return events.StatusCmd(fmt.Sprintf("Invalid %s: %v", field.YAMLPath(), err), true)
		}
		m.doc.Set(val, field.Path...)
		m.dirty = true
		m.editing = false
		m.editInput.Blur()
		return events.StatusCmd(fmt.Sprintf("Updated [%s] (press 's' to save & apply)", field.YAMLPath()), false)

	default:
		var cmd tea.Cmd
		m.editInput, cmd = m.editInput.Update(msg)
		return cmd
	}
}

func (m *ConfigsModel) loadOverrideCmd() tea.Cmd {
	return func() tea.Msg {
		doc, err := m.profileMgr.LoadOverride()
		return overrideLoadedMsg{doc: doc, err: err}
	}
}

func (m *ConfigsModel) saveAndApplyCmd(doc *profile.OverrideDoc) tea.Cmd {
	return func() tea.Msg {
		if err := m.profileMgr.SaveOverride(doc); err != nil {
			return overrideAppliedMsg{err: err}
		}

		activeSub := m.profileMgr.ActiveProfile()
		if activeSub == "" {
			return overrideAppliedMsg{reloaded: false}
		}

		mihomoCfgPath, err := m.profileMgr.BuildMihomoConfig(activeSub)
		if err != nil {
			return overrideAppliedMsg{activeSub: activeSub, err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := m.client.ReloadConfig(ctx, mihomoCfgPath, true); err != nil {
			return overrideAppliedMsg{activeSub: activeSub, err: err}
		}

		return overrideAppliedMsg{reloaded: true, activeSub: activeSub}
	}
}
