package pages

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jackie8tao/mihotui/internal/app/events"
	"github.com/jackie8tao/mihotui/internal/app/keymaps"
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
	client           mihomo.Client
	mode             string
	focusPane        views.FocusPane
	selectedGroupIdx int
	selectedNodeIdx  int
	proxies          map[string]mihomo.ProxyItem
	groups           []mihomo.ProxyItem
}

// NewProxiesModel initializes a self-contained ProxiesModel.
func NewProxiesModel(cli mihomo.Client) *ProxiesModel {
	return &ProxiesModel{
		client:    cli,
		focusPane: views.PaneGroups,
		proxies:   make(map[string]mihomo.ProxyItem),
	}
}

// Title returns the tab header title for the Proxies page.
func (m *ProxiesModel) Title() string {
	return "2. Proxies"
}

// Help returns the keybinding hints displayed in the footer for the Proxies page.
func (m *ProxiesModel) Help() string {
	return "[←/→] Pane | [↑/↓] Move | [Enter] Select | [t] Ping | [T] Ping Group"
}

// CapturingInput reports whether the Proxies page is currently capturing raw keyboard input.
func (m *ProxiesModel) CapturingInput() bool {
	return false
}

// Init triggers the initial fetch of proxy groups and nodes.
func (m *ProxiesModel) Init() tea.Cmd {
	return m.fetchProxiesCmd()
}

// Update handles proxy loading, latency testing, node selection, and keyboard navigation.
func (m *ProxiesModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case proxiesLoadedMsg:
		if msg.err != nil {
			m.clearTestingFlags()
			return events.StatusCmd(fmt.Sprintf("Load proxies failed: %v", msg.err), true)
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
			return events.StatusCmd(fmt.Sprintf("Latency test failed for %s: %v", msg.proxyName, msg.err), true)
		}

		item.History = append(item.History, mihomo.DelayHistory{
			Time:  time.Now(),
			Delay: msg.delay,
		})
		m.proxies[msg.proxyName] = item
		return events.StatusCmd(fmt.Sprintf("Latency for %s: %s", msg.proxyName, mihomo.FormatDelay(msg.delay)), false)

	case testAllDoneMsg:
		return tea.Batch(
			events.StatusCmd(fmt.Sprintf("Finished testing %d nodes in [%s]", msg.count, msg.groupName), false),
			m.fetchProxiesCmd(),
		)

	case proxySelectedMsg:
		if msg.err != nil {
			return events.StatusCmd(fmt.Sprintf("Select proxy failed: %v", msg.err), true)
		}
		return tea.Batch(
			events.StatusCmd(fmt.Sprintf("[%s] => Switched to: %s", msg.groupName, msg.proxyName), false),
			m.fetchProxiesCmd(),
		)

	case events.ModeSyncedMsg:
		if !strings.EqualFold(m.mode, msg.Mode) {
			m.mode = msg.Mode
			m.rebuildGroups(strings.EqualFold(msg.Mode, "global"))
		}
		return nil

	case events.ModeUpdatedMsg:
		if msg.Err == nil && !strings.EqualFold(m.mode, msg.NewMode) {
			m.mode = msg.NewMode
			m.rebuildGroups(strings.EqualFold(msg.NewMode, "global"))
		}
		return nil

	case events.ProfileAppliedMsg:
		return m.fetchProxiesCmd()

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return nil
}

// View renders the Proxies page within the given terminal dimensions.
func (m *ProxiesModel) View(width, height int) string {
	return views.RenderProxies(views.ProxiesProps{
		Width:            width,
		Height:           height,
		FocusPane:        m.focusPane,
		Groups:           m.groups,
		Proxies:          m.proxies,
		SelectedGroupIdx: m.selectedGroupIdx,
		SelectedNodeIdx:  m.selectedNodeIdx,
	})
}

// applyLoadedProxies replaces the cached proxies map and rebuilds the sorted group list.
func (m *ProxiesModel) applyLoadedProxies(proxies map[string]mihomo.ProxyItem) {
	m.proxies = proxies
	m.rebuildGroups(false)
}

// rebuildGroups derives visible proxy groups for the active outbound mode and clamps cursor indices.
func (m *ProxiesModel) rebuildGroups(focusGlobal bool) {
	prevGroupName := ""
	if grp := m.currentGroup(); grp != nil {
		prevGroupName = grp.Name
	}

	// Hide GLOBAL in non-global modes (e.g. rule/direct) and strip nested proxy groups
	// (Selector/URLTest/etc.) from GLOBAL so only concrete outbound nodes are shown.
	isGlobalMode := strings.EqualFold(m.mode, "global")
	var groups []mihomo.ProxyItem
	for _, item := range m.proxies {
		if len(item.All) == 0 {
			continue
		}
		if item.Name == "GLOBAL" {
			if !isGlobalMode {
				continue
			}
			item.All = m.filterConcreteNodes(item.All)
		}
		groups = append(groups, item)
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
		m.selectedGroupIdx = 0
		m.selectedNodeIdx = 0
		return
	}

	if focusGlobal {
		m.selectedGroupIdx = 0
		m.selectedNodeIdx = 0
		return
	}

	if prevGroupName != "" {
		for i, g := range m.groups {
			if g.Name == prevGroupName {
				m.selectedGroupIdx = i
				break
			}
		}
	}
	if m.selectedGroupIdx >= len(m.groups) {
		m.selectedGroupIdx = len(m.groups) - 1
	}

	nodes := m.groups[m.selectedGroupIdx].All
	if len(nodes) == 0 {
		m.selectedNodeIdx = 0
		return
	}
	if m.selectedNodeIdx >= len(nodes) {
		m.selectedNodeIdx = len(nodes) - 1
	}
}

// filterConcreteNodes removes nested proxy groups from names, keeping only leaf outbound nodes.
func (m *ProxiesModel) filterConcreteNodes(names []string) []string {
	filtered := make([]string, 0, len(names))
	for _, name := range names {
		if child, ok := m.proxies[name]; ok && len(child.All) > 0 {
			continue
		}
		filtered = append(filtered, name)
	}
	return filtered
}

// clearTestingFlags resets the in-progress latency testing indicator on all cached proxies.
func (m *ProxiesModel) clearTestingFlags() {
	for name, item := range m.proxies {
		if item.TestingDelay {
			item.TestingDelay = false
			m.proxies[name] = item
		}
	}
}

// currentGroup returns the currently selected proxy group, or nil if none is selected.
func (m *ProxiesModel) currentGroup() *mihomo.ProxyItem {
	if len(m.groups) == 0 || m.selectedGroupIdx < 0 || m.selectedGroupIdx >= len(m.groups) {
		return nil
	}
	return &m.groups[m.selectedGroupIdx]
}

// currentSelectedNode returns the name of the currently selected node in the active group.
func (m *ProxiesModel) currentSelectedNode() string {
	grp := m.currentGroup()
	if grp == nil || len(grp.All) == 0 || m.selectedNodeIdx < 0 || m.selectedNodeIdx >= len(grp.All) {
		return ""
	}
	return grp.All[m.selectedNodeIdx]
}

// handleKeyMsg processes pane focus switching, list navigation, proxy selection, and latency test shortcuts.
func (m *ProxiesModel) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, keymaps.Left):
		m.focusPane = views.PaneGroups

	case key.Matches(msg, keymaps.Right):
		if len(m.groups) > 0 {
			m.focusPane = views.PaneNodes
		}

	case key.Matches(msg, keymaps.Up):
		if m.focusPane == views.PaneGroups {
			if m.selectedGroupIdx > 0 {
				m.selectedGroupIdx--
				m.selectedNodeIdx = 0
			}
		} else if m.selectedNodeIdx > 0 {
			m.selectedNodeIdx--
		}

	case key.Matches(msg, keymaps.Down):
		if m.focusPane == views.PaneGroups {
			if m.selectedGroupIdx < len(m.groups)-1 {
				m.selectedGroupIdx++
				m.selectedNodeIdx = 0
			}
		} else {
			grp := m.currentGroup()
			if grp != nil && m.selectedNodeIdx < len(grp.All)-1 {
				m.selectedNodeIdx++
			}
		}

	case key.Matches(msg, keymaps.Enter):
		if m.focusPane == views.PaneNodes {
			grp := m.currentGroup()
			node := m.currentSelectedNode()
			if grp != nil && node != "" {
				return m.selectProxyCmd(grp.Name, node)
			}
		}

	case key.Matches(msg, keymaps.TestDelay):
		node := m.currentSelectedNode()
		if node == "" {
			return nil
		}
		if item, ok := m.proxies[node]; ok {
			item.TestingDelay = true
			m.proxies[node] = item
		}
		return tea.Batch(
			events.StatusCmd(fmt.Sprintf("Testing latency for %s...", node), false),
			m.testDelayCmd(node),
		)

	case key.Matches(msg, keymaps.TestAll):
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
			events.StatusCmd(fmt.Sprintf("Testing all %d nodes in [%s]...", len(nodesToTest), grp.Name), false),
			m.testAllCmd(grp.Name, nodesToTest),
		)
	}

	return nil
}

// fetchProxiesCmd asynchronously fetches all proxy groups and nodes from Mihomo.
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

// selectProxyCmd sends an asynchronous request to switch the active node in groupName.
func (m *ProxiesModel) selectProxyCmd(groupName, nodeName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := m.client.SelectProxy(ctx, groupName, nodeName)
		return proxySelectedMsg{groupName: groupName, proxyName: nodeName, err: err}
	}
}

// testDelayCmd asynchronously measures the latency of a single proxy node.
func (m *ProxiesModel) testDelayCmd(nodeName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(defaultTestTimeoutMs+1000)*time.Millisecond)
		defer cancel()
		delay, err := m.client.TestDelay(ctx, nodeName, defaultTestURL, defaultTestTimeoutMs)
		return delayResultMsg{proxyName: nodeName, delay: delay, err: err}
	}
}

// testAllCmd concurrently measures latency for all nodes in groupName with bounded parallelism.
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
