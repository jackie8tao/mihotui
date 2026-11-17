package pages

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jackie8tao/mihotui/internal/app/views"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

type trafficMsg mihomo.Traffic

type overviewLoadedMsg struct {
	version      *mihomo.Version
	config       *mihomo.Config
	memory       *mihomo.Memory
	notifyStatus bool
	err          error
}

// OverviewModel encapsulates state, data loading, and rendering for the Overview tab.
type OverviewModel struct {
	ctx         context.Context
	client      mihomo.Client
	onStatus    StatusCmdFunc
	onModeSync  ModeSyncCmdFunc
	trafficChan chan mihomo.Traffic
	version     mihomo.Version
	config      mihomo.Config
	memory      mihomo.Memory
	traffic     mihomo.Traffic
	totalUp     int64
	totalDown   int64
}

// NewOverviewModel initializes a self-contained OverviewModel bound to the lifecycle ctx.
func NewOverviewModel(ctx context.Context, cli mihomo.Client, onStatus StatusCmdFunc, onModeSync ModeSyncCmdFunc) *OverviewModel {
	if ctx == nil {
		ctx = context.Background()
	}
	return &OverviewModel{
		ctx:         ctx,
		client:      cli,
		onStatus:    onStatus,
		onModeSync:  onModeSync,
		trafficChan: make(chan mihomo.Traffic, 50),
	}
}

// SetMode updates the locally displayed outbound mode when switched globally.
func (m *OverviewModel) SetMode(mode string) {
	m.config.Mode = mode
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
	go func() {
		_ = m.client.SubscribeTraffic(m.ctx, m.trafficChan)
	}()

	return tea.Batch(m.fetchOverviewCmd(true), m.waitForTrafficCmd())
}

func (m *OverviewModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case trafficMsg:
		m.traffic = mihomo.Traffic(msg)
		m.totalUp += m.traffic.Up
		m.totalDown += m.traffic.Down
		return m.waitForTrafficCmd()

	case overviewLoadedMsg:
		if msg.err != nil {
			return emitStatus(m.onStatus, fmt.Sprintf("API Connection Error: %v", msg.err), true)
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
		cmds := []tea.Cmd{emitModeSync(m.onModeSync, m.config.Mode)}
		if msg.notifyStatus {
			cmds = append(cmds, emitStatus(m.onStatus, "Connected to Mihomo", false))
		}
		return tea.Batch(cmds...)

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

func (m *OverviewModel) waitForTrafficCmd() tea.Cmd {
	return func() tea.Msg {
		select {
		case <-m.ctx.Done():
			return nil
		case t, ok := <-m.trafficChan:
			if !ok {
				return nil
			}
			return trafficMsg(t)
		}
	}
}
