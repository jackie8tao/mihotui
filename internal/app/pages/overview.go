package pages

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jackie8tao/mihotui/internal/app/views"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

type overviewLoadedMsg struct {
	version      *mihomo.Version
	config       *mihomo.Config
	memory       *mihomo.Memory
	notifyStatus bool
	err          error
}

// OverviewModel encapsulates state, data loading, and rendering for the Overview tab.
type OverviewModel struct {
	client    mihomo.Client
	version   mihomo.Version
	config    mihomo.Config
	memory    mihomo.Memory
	traffic   mihomo.Traffic
	totalUp   int64
	totalDown int64
}

// NewOverviewModel initializes a self-contained OverviewModel.
func NewOverviewModel(cli mihomo.Client) *OverviewModel {
	return &OverviewModel{
		client: cli,
	}
}

func (m *OverviewModel) Title() string {
	return "1. Overview"
}

func (m *OverviewModel) Help() string {
	return ""
}

func (m *OverviewModel) CapturingInput() bool {
	return false
}

func (m *OverviewModel) Init() tea.Cmd {
	return m.fetchOverviewCmd(true)
}

func (m *OverviewModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case TrafficMsg:
		m.traffic = mihomo.Traffic(msg)
		m.totalUp += m.traffic.Up
		m.totalDown += m.traffic.Down

	case overviewLoadedMsg:
		if msg.err != nil {
			return statusCmd(fmt.Sprintf("API Connection Error: %v", msg.err), true)
		}
		if msg.version != nil {
			m.version = *msg.version
		}
		if msg.config != nil {
			m.config = *msg.config
		}
		if msg.memory != nil {
			m.memory = *msg.memory
		}
		cmds := []tea.Cmd{modeSyncedCmd(m.config.Mode)}
		if msg.notifyStatus {
			cmds = append(cmds, statusCmd("Connected to Mihomo", false))
		}
		return tea.Batch(cmds...)

	case ModeUpdatedMsg:
		if msg.Err == nil {
			m.config.Mode = msg.NewMode
		}

	case ProfileAppliedEventMsg:
		return m.fetchOverviewCmd(false)
	}

	return nil
}

func (m *OverviewModel) View(width, height int) string {
	return views.RenderOverview(views.OverviewProps{
		Width:     width,
		Traffic:   m.traffic,
		TotalUp:   m.totalUp,
		TotalDown: m.totalDown,
		Config:    m.config,
		Version:   m.version,
		Memory:    m.memory,
	})
}

func (m *OverviewModel) fetchOverviewCmd(notifyStatus bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		v, err := m.client.GetVersion(ctx)
		if err != nil {
			return overviewLoadedMsg{err: err}
		}
		c, err := m.client.GetConfig(ctx)
		if err != nil {
			return overviewLoadedMsg{err: err}
		}
		mem, _ := m.client.GetMemory(ctx)

		return overviewLoadedMsg{
			version:      v,
			config:       c,
			memory:       mem,
			notifyStatus: notifyStatus,
		}
	}
}
