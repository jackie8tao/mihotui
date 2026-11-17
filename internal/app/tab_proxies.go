package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jackie8tao/mihotui/internal/app/views"
	"github.com/jackie8tao/mihotui/pkg/mihomo"
)

func (m *AppModel) currentGroup() *mihomo.ProxyItem {
	if len(m.groups) == 0 || m.proxiesState.GroupCursor < 0 || m.proxiesState.GroupCursor >= len(m.groups) {
		return nil
	}
	return &m.groups[m.proxiesState.GroupCursor]
}

func (m *AppModel) currentSelectedNode() string {
	grp := m.currentGroup()
	if grp == nil || len(grp.All) == 0 || m.proxiesState.NodeCursor < 0 || m.proxiesState.NodeCursor >= len(grp.All) {
		return ""
	}
	return grp.All[m.proxiesState.NodeCursor]
}

func (m *AppModel) selectProxyCmd(groupName, nodeName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := m.client.SelectProxy(ctx, groupName, nodeName)
		return proxySelectedMsg{groupName: groupName, proxyName: nodeName, err: err}
	}
}

func (m *AppModel) testDelayCmd(nodeName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(defaultTestTimeoutMs+1000)*time.Millisecond)
		defer cancel()
		delay, err := m.client.TestDelay(ctx, nodeName, defaultTestURL, defaultTestTimeoutMs)
		return delayResultMsg{proxyName: nodeName, delay: delay, err: err}
	}
}

func (m *AppModel) updateProxiesTab(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Left):
		m.proxiesState.FocusPane = views.PaneGroups

	case key.Matches(msg, m.keys.Right):
		if len(m.groups) > 0 {
			m.proxiesState.FocusPane = views.PaneNodes
		}

	case key.Matches(msg, m.keys.Up):
		if m.proxiesState.FocusPane == views.PaneGroups {
			if m.proxiesState.GroupCursor > 0 {
				m.proxiesState.GroupCursor--
				m.proxiesState.NodeCursor = 0
			}
		} else {
			if m.proxiesState.NodeCursor > 0 {
				m.proxiesState.NodeCursor--
			}
		}

	case key.Matches(msg, m.keys.Down):
		if m.proxiesState.FocusPane == views.PaneGroups {
			if m.proxiesState.GroupCursor < len(m.groups)-1 {
				m.proxiesState.GroupCursor++
				m.proxiesState.NodeCursor = 0
			}
		} else {
			grp := m.currentGroup()
			if grp != nil && m.proxiesState.NodeCursor < len(grp.All)-1 {
				m.proxiesState.NodeCursor++
			}
		}

	case key.Matches(msg, m.keys.Enter):
		if m.proxiesState.FocusPane == views.PaneNodes {
			grp := m.currentGroup()
			node := m.currentSelectedNode()
			if grp != nil && node != "" {
				return m.selectProxyCmd(grp.Name, node)
			}
		} else {
			m.proxiesState.FocusPane = views.PaneNodes
		}

	case key.Matches(msg, m.keys.TestDelay):
		node := m.currentSelectedNode()
		if node != "" {
			if item, ok := m.proxies[node]; ok {
				item.TestingDelay = true
				m.proxies[node] = item
			}
			m.statusMsg = fmt.Sprintf("Testing latency for %s...", node)
			return m.testDelayCmd(node)
		}

	case key.Matches(msg, m.keys.TestAll):
		grp := m.currentGroup()
		if grp != nil && len(grp.All) > 0 {
			m.statusMsg = fmt.Sprintf("Testing all %d nodes in [%s]...", len(grp.All), grp.Name)
			for _, node := range grp.All {
				if item, ok := m.proxies[node]; ok {
					item.TestingDelay = true
					m.proxies[node] = item
				}
			}
			nodesToTest := append([]string(nil), grp.All...)
			return func() tea.Msg {
				var wg sync.WaitGroup
				sem := make(chan struct{}, 8)
				for _, node := range nodesToTest {
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
				return m.fetchDataCmd()()
			}
		}
	}

	return nil
}
