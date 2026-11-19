package pages

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jackie8tao/mihotui/internal/app/keys"
	"github.com/jackie8tao/mihotui/internal/app/views"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

const (
	defaultTestURL       = "http://www.gstatic.com/generate_204"
	defaultTestTimeoutMs = 3000
)

type (
	proxiesLoadedMsg struct {
		proxies map[string]mihomo.ProxyItem
		err     error
	}
	delayResultMsg struct {
		proxyName string
		delay     int
		err       error
	}
	testAllDoneMsg struct {
		groupName string
		count     int
	}
	proxySelectedMsg struct {
		groupName string
		proxyName string
		err       error
	}
)

// ProxiesModel encapsulates state, interactions, and rendering for the Proxies tab.
type ProxiesModel struct {
	client      mihomo.Client
	onStatus    StatusCmdFunc
	focusPane   views.FocusPane
	groupCursor int
	nodeCursor  int
	proxies     map[string]mihomo.ProxyItem
	groups      []mihomo.ProxyItem
}

// NewProxiesModel initializes a self-contained ProxiesModel.
func NewProxiesModel(cli mihomo.Client, onStatus StatusCmdFunc) *ProxiesModel {
	return &ProxiesModel{
		client:    cli,
		onStatus:  onStatus,
		focusPane: views.PaneGroups,
		proxies:   make(map[string]mihomo.ProxyItem),
	}
}

func (m *ProxiesModel) Title() string {
	return "2. Proxies"
}

func (m *ProxiesModel) Help() string {
	return "[←/→] Pane | [↑/↓] Move | [Enter] Select | [t] Ping | [T] Ping Group"
}

func (m *ProxiesModel) CapturingInput() bool {
	return false
}

func (m *ProxiesModel) Init() tea.Cmd {
	return m.fetchProxiesCmd()
}

func (m *ProxiesModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case proxiesLoadedMsg:
		if msg.err != nil {
			m.clearTestingFlags()
			return emitStatus(m.onStatus, fmt.Sprintf("Load proxies failed: %v", msg.err), true)
		}
		m.applyLoadedProxies(msg.proxies)
		return nil

	case delayResultMsg:
		item, ok := m.proxies[msg.proxyName]
		if !ok {
			return nil
		}
		item.TestingDelay = false
		if msg.err != nil {
			m.proxies[msg.proxyName] = item
			return emitStatus(m.onStatus, fmt.Sprintf("Latency test failed for %s: %v", msg.proxyName, msg.err), true)
		}

		item.History = append(item.History, mihomo.DelayHistory{
			Time:  time.Now(),
			Delay: msg.delay,
		})
		m.proxies[msg.proxyName] = item
		return emitStatus(m.onStatus, fmt.Sprintf("Latency for %s: %s", msg.proxyName, mihomo.FormatDelay(msg.delay)), false)

	case testAllDoneMsg:
		return tea.Batch(
			emitStatus(m.onStatus, fmt.Sprintf("Finished testing %d nodes in [%s]", msg.count, msg.groupName), false),
			m.fetchProxiesCmd(),
		)

	case proxySelectedMsg:
		if msg.err != nil {
			return emitStatus(m.onStatus, fmt.Sprintf("Select proxy failed: %v", msg.err), true)
		}
		return tea.Batch(
			emitStatus(m.onStatus, fmt.Sprintf("[%s] => Switched to: %s", msg.groupName, msg.proxyName), false),
			m.fetchProxiesCmd(),
		)

	case ProfileAppliedEventMsg:
		return m.fetchProxiesCmd()

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return nil
}

func (m *ProxiesModel) View(width, height int) string {
	return views.RenderProxies(views.ProxiesProps{
		Width:       width,
		Height:      height,
		FocusPane:   m.focusPane,
		Groups:      m.groups,
		Proxies:     m.proxies,
		GroupCursor: m.groupCursor,
		NodeCursor:  m.nodeCursor,
	})
}

func (m *ProxiesModel) applyLoadedProxies(proxies map[string]mihomo.ProxyItem) {
	m.proxies = proxies

	var groups []mihomo.ProxyItem
	for _, item := range m.proxies {
		if len(item.All) > 0 {
			groups = append(groups, item)
		}
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Name == "GLOBAL" {
			return true
		}
		if groups[j].Name == "GLOBAL" {
			return false
		}
		return groups[i].Name < groups[j].Name
	})
	m.groups = groups

	if len(m.groups) == 0 {
		m.groupCursor = 0
		m.nodeCursor = 0
		return
	}
	if m.groupCursor >= len(m.groups) {
		m.groupCursor = len(m.groups) - 1
	}

	nodes := m.groups[m.groupCursor].All
	if len(nodes) == 0 {
		m.nodeCursor = 0
	} else if m.nodeCursor >= len(nodes) {
		m.nodeCursor = len(nodes) - 1
	}
}

func (m *ProxiesModel) clearTestingFlags() {
	for name, item := range m.proxies {
		if item.TestingDelay {
			item.TestingDelay = false
			m.proxies[name] = item
		}
	}
}

func (m *ProxiesModel) currentGroup() *mihomo.ProxyItem {
	if len(m.groups) == 0 || m.groupCursor < 0 || m.groupCursor >= len(m.groups) {
		return nil
	}
	return &m.groups[m.groupCursor]
}

func (m *ProxiesModel) currentSelectedNode() string {
	grp := m.currentGroup()
	if grp == nil || len(grp.All) == 0 || m.nodeCursor < 0 || m.nodeCursor >= len(grp.All) {
		return ""
	}
	return grp.All[m.nodeCursor]
}

func (m *ProxiesModel) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, keys.Left):
		m.focusPane = views.PaneGroups

	case key.Matches(msg, keys.Right):
		if len(m.groups) > 0 {
			m.focusPane = views.PaneNodes
		}

	case key.Matches(msg, keys.Up):
		if m.focusPane == views.PaneGroups {
			if m.groupCursor > 0 {
				m.groupCursor--
				m.nodeCursor = 0
			}
		} else if m.nodeCursor > 0 {
			m.nodeCursor--
		}

	case key.Matches(msg, keys.Down):
		if m.focusPane == views.PaneGroups {
			if m.groupCursor < len(m.groups)-1 {
				m.groupCursor++
				m.nodeCursor = 0
			}
		} else {
			grp := m.currentGroup()
			if grp != nil && m.nodeCursor < len(grp.All)-1 {
				m.nodeCursor++
			}
		}

	case key.Matches(msg, keys.Enter):
		if m.focusPane != views.PaneNodes {
			if len(m.groups) > 0 {
				m.focusPane = views.PaneNodes
			}
			return nil
		}
		grp := m.currentGroup()
		node := m.currentSelectedNode()
		if grp != nil && node != "" {
			return m.selectProxyCmd(grp.Name, node)
		}

	case key.Matches(msg, keys.TestDelay):
		node := m.currentSelectedNode()
		if node == "" {
			return nil
		}
		if item, ok := m.proxies[node]; ok {
			item.TestingDelay = true
			m.proxies[node] = item
		}
		return tea.Batch(
			emitStatus(m.onStatus, fmt.Sprintf("Testing latency for %s...", node), false),
			m.testDelayCmd(node),
		)

	case key.Matches(msg, keys.TestAll):
		grp := m.currentGroup()
		if grp == nil || len(grp.All) == 0 {
			return nil
		}
		for _, node := range grp.All {
			if item, ok := m.proxies[node]; ok {
				item.TestingDelay = true
				m.proxies[node] = item
			}
		}
		nodesToTest := append([]string(nil), grp.All...)
		return tea.Batch(
			emitStatus(m.onStatus, fmt.Sprintf("Testing all %d nodes in [%s]...", len(nodesToTest), grp.Name), false),
			m.testAllCmd(grp.Name, nodesToTest),
		)
	}

	return nil
}

func (m *ProxiesModel) fetchProxiesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := m.client.GetProxies(ctx)
		if err != nil {
			return proxiesLoadedMsg{err: err}
		}
		return proxiesLoadedMsg{proxies: resp.Proxies}
	}
}

func (m *ProxiesModel) selectProxyCmd(groupName, nodeName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := m.client.SelectProxy(ctx, groupName, nodeName)
		return proxySelectedMsg{groupName: groupName, proxyName: nodeName, err: err}
	}
}

func (m *ProxiesModel) testDelayCmd(nodeName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(defaultTestTimeoutMs+1000)*time.Millisecond)
		defer cancel()
		delay, err := m.client.TestDelay(ctx, nodeName, defaultTestURL, defaultTestTimeoutMs)
		return delayResultMsg{proxyName: nodeName, delay: delay, err: err}
	}
}

func (m *ProxiesModel) testAllCmd(groupName string, nodes []string) tea.Cmd {
	return func() tea.Msg {
		var wg sync.WaitGroup
		sem := make(chan struct{}, 8)
		for _, node := range nodes {
			wg.Add(1)
			go func(n string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(defaultTestTimeoutMs+1000)*time.Millisecond)
				defer cancel()
				_, _ = m.client.TestDelay(ctx, n, defaultTestURL, defaultTestTimeoutMs)
			}(node)
		}
		wg.Wait()
		return testAllDoneMsg{groupName: groupName, count: len(nodes)}
	}
}
