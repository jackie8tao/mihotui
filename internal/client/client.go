package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mihomo-tui/internal/model"

	"github.com/gorilla/websocket"
)

// Client communicates with Mihomo's External Controller API.
type Client struct {
	baseURL    string
	secret     string
	httpClient *http.Client
}

// New creates a new Client instance.
func New(baseURL, secret string) *Client {
	// Normalize baseURL (remove trailing slash)
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL: baseURL,
		secret:  secret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// setHeaders adds common headers like Authorization.
func (c *Client) setHeaders(req *http.Request) {
	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}
	req.Header.Set("Content-Type", "application/json")
}

// GetVersion fetches the Mihomo version.
func (c *Client) GetVersion(ctx context.Context) (*model.Version, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/version", nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var v model.Version
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

// GetConfig fetches current runtime configs.
func (c *Client) GetConfig(ctx context.Context) (*model.Config, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/configs", nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var cfg model.Config
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// UpdateMode switches the proxy mode (rule, global, direct).
func (c *Client) UpdateMode(ctx context.Context, mode string) error {
	payload, _ := json.Marshal(map[string]string{"mode": mode})
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.baseURL+"/configs", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update mode (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// ReloadConfig requests Mihomo to hot-reload configuration from a given file path.
func (c *Client) ReloadConfig(ctx context.Context, configPath string, force bool) error {
	payload, _ := json.Marshal(map[string]string{"path": configPath})
	endpoint := c.baseURL + "/configs"
	if force {
		endpoint += "?force=true"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to reload config (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// GetProxies fetches all proxies and proxy groups.
func (c *Client) GetProxies(ctx context.Context) (*model.ProxiesResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/proxies", nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var res model.ProxiesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

// SelectProxy switches the selected proxy node inside a group.
func (c *Client) SelectProxy(ctx context.Context, groupName, proxyName string) error {
	payload, _ := json.Marshal(map[string]string{"name": proxyName})
	escapedGroup := url.PathEscape(groupName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"/proxies/"+escapedGroup, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to select proxy (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// TestDelay tests the latency of a proxy node.
func (c *Client) TestDelay(ctx context.Context, proxyName, testURL string, timeoutMs int) (int, error) {
	escapedProxy := url.PathEscape(proxyName)
	endpoint := fmt.Sprintf("%s/proxies/%s/delay?url=%s&timeout=%d",
		c.baseURL,
		escapedProxy,
		url.QueryEscape(testURL),
		timeoutMs,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	c.setHeaders(req)

	// Delay testing should allow timeoutMs + slight margin
	client := &http.Client{
		Timeout: time.Duration(timeoutMs+1500) * time.Millisecond,
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var res model.DelayResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, err
	}
	return res.Delay, nil
}

// GetMemory fetches memory usage statistics.
func (c *Client) GetMemory(ctx context.Context) (*model.Memory, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/memory", nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var m model.Memory
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// SubscribeTraffic opens a WebSocket connection to stream real-time bandwidth.
func (c *Client) SubscribeTraffic(ctx context.Context, trafficChan chan<- model.Traffic) error {
	wsURL := c.baseURL
	if strings.HasPrefix(wsURL, "http://") {
		wsURL = "ws://" + strings.TrimPrefix(wsURL, "http://")
	} else if strings.HasPrefix(wsURL, "https://") {
		wsURL = "wss://" + strings.TrimPrefix(wsURL, "https://")
	}

	u, err := url.Parse(wsURL + "/traffic")
	if err != nil {
		return err
	}

	if c.secret != "" {
		q := u.Query()
		q.Set("token", c.secret)
		u.RawQuery = q.Encode()
	}

	header := http.Header{}
	if c.secret != "" {
		header.Set("Authorization", "Bearer "+c.secret)
	}

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, u.String(), header)
	if err != nil {
		return err
	}
	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var t model.Traffic
			if err := conn.ReadJSON(&t); err != nil {
				return err
			}
			select {
			case trafficChan <- t:
			default:
				// Avoid blocking if consumer is busy
			}
		}
	}
}
