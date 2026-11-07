package mihomo

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

	"github.com/gorilla/websocket"
)

// Client is the interface for communicating with Mihomo External Controller API.
type Client interface {
	// Core & Configs
	GetVersion(ctx context.Context) (*Version, error)
	GetConfig(ctx context.Context) (*Config, error)
	UpdateMode(ctx context.Context, mode string) error
	ReloadConfig(ctx context.Context, configPath string, force bool) error
	GetMemory(ctx context.Context) (*Memory, error)

	// Proxies
	GetProxies(ctx context.Context) (*ProxiesResponse, error)
	SelectProxy(ctx context.Context, groupName, proxyName string) error
	TestDelay(ctx context.Context, proxyName, testURL string, timeoutMs int) (int, error)

	// Streaming (WebSocket)
	SubscribeTraffic(ctx context.Context, trafficChan chan<- Traffic) error
	SubscribeLogs(ctx context.Context, level string, logChan chan<- LogMessage) error
}

type clientImpl struct {
	baseURL    string
	opts       clientOptions
	httpClient *http.Client
}

// New creates a new Client configured with functional options.
func New(baseURL string, opts ...Option) (Client, error) {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:9090"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	// Validate URL scheme
	u, err := url.Parse(baseURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("mihomo: invalid base URL: %s", baseURL)
	}

	options := clientOptions{
		timeout:   10 * time.Second,
		userAgent: "mihomo-sdk-go/v1",
	}
	for _, opt := range opts {
		opt(&options)
	}

	httpClient := options.httpClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: options.timeout,
		}
	}

	return &clientImpl{
		baseURL:    baseURL,
		opts:       options,
		httpClient: httpClient,
	}, nil
}

func (c *clientImpl) do(ctx context.Context, method, path string, body any, out any) error {
	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("mihomo: failed to marshal request payload: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	reqURL := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("mihomo: failed to build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.opts.userAgent != "" {
		req.Header.Set("User-Agent", c.opts.userAgent)
	}
	if c.opts.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.opts.secret)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		rawBody, _ := io.ReadAll(resp.Body)
		var apiErr struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(rawBody, &apiErr)
		msg := apiErr.Message
		if msg == "" {
			msg = strings.TrimSpace(string(rawBody))
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    msg,
		}
	}

	if out != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("mihomo: failed to decode response: %w", err)
		}
	}

	return nil
}

func (c *clientImpl) GetVersion(ctx context.Context) (*Version, error) {
	var v Version
	if err := c.do(ctx, http.MethodGet, "/version", nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (c *clientImpl) GetConfig(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := c.do(ctx, http.MethodGet, "/configs", nil, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *clientImpl) UpdateMode(ctx context.Context, mode string) error {
	payload := map[string]string{"mode": strings.ToLower(mode)}
	return c.do(ctx, http.MethodPatch, "/configs", payload, nil)
}

func (c *clientImpl) ReloadConfig(ctx context.Context, configPath string, force bool) error {
	path := "/configs"
	if force {
		path += "?force=true"
	}
	payload := map[string]string{"path": configPath}
	return c.do(ctx, http.MethodPut, path, payload, nil)
}

func (c *clientImpl) GetMemory(ctx context.Context) (*Memory, error) {
	var m Memory
	if err := c.do(ctx, http.MethodGet, "/memory", nil, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *clientImpl) GetProxies(ctx context.Context) (*ProxiesResponse, error) {
	var res ProxiesResponse
	if err := c.do(ctx, http.MethodGet, "/proxies", nil, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *clientImpl) SelectProxy(ctx context.Context, groupName, proxyName string) error {
	path := fmt.Sprintf("/proxies/%s", url.PathEscape(groupName))
	payload := map[string]string{"name": proxyName}
	return c.do(ctx, http.MethodPut, path, payload, nil)
}

func (c *clientImpl) TestDelay(ctx context.Context, proxyName, testURL string, timeoutMs int) (int, error) {
	path := fmt.Sprintf("/proxies/%s/delay?url=%s&timeout=%d",
		url.PathEscape(proxyName),
		url.QueryEscape(testURL),
		timeoutMs,
	)

	var res DelayResponse
	if err := c.do(ctx, http.MethodGet, path, nil, &res); err != nil {
		return 0, err
	}
	return res.Delay, nil
}

func (c *clientImpl) SubscribeTraffic(ctx context.Context, trafficChan chan<- Traffic) error {
	wsURL := c.baseURL
	if strings.HasPrefix(wsURL, "http://") {
		wsURL = "ws://" + strings.TrimPrefix(wsURL, "http://")
	} else if strings.HasPrefix(wsURL, "https://") {
		wsURL = "wss://" + strings.TrimPrefix(wsURL, "https://")
	}

	u, err := url.Parse(wsURL + "/traffic")
	if err != nil {
		return fmt.Errorf("mihomo: invalid websocket url: %w", err)
	}

	if c.opts.secret != "" {
		q := u.Query()
		q.Set("token", c.opts.secret)
		u.RawQuery = q.Encode()
	}

	header := http.Header{}
	if c.opts.secret != "" {
		header.Set("Authorization", "Bearer "+c.opts.secret)
	}
	if c.opts.userAgent != "" {
		header.Set("User-Agent", c.opts.userAgent)
	}

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, u.String(), header)
	if err != nil {
		return fmt.Errorf("mihomo: failed to dial traffic websocket: %w", err)
	}
	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var t Traffic
			if err := conn.ReadJSON(&t); err != nil {
				return err
			}
			select {
			case trafficChan <- t:
			default:
			}
		}
	}
}

func (c *clientImpl) SubscribeLogs(ctx context.Context, level string, logChan chan<- LogMessage) error {
	wsURL := c.baseURL
	if strings.HasPrefix(wsURL, "http://") {
		wsURL = "ws://" + strings.TrimPrefix(wsURL, "http://")
	} else if strings.HasPrefix(wsURL, "https://") {
		wsURL = "wss://" + strings.TrimPrefix(wsURL, "https://")
	}

	endpoint := "/logs"
	if level != "" {
		endpoint += "?level=" + url.QueryEscape(level)
	}

	u, err := url.Parse(wsURL + endpoint)
	if err != nil {
		return fmt.Errorf("mihomo: invalid websocket url: %w", err)
	}

	if c.opts.secret != "" {
		q := u.Query()
		q.Set("token", c.opts.secret)
		u.RawQuery = q.Encode()
	}

	header := http.Header{}
	if c.opts.secret != "" {
		header.Set("Authorization", "Bearer "+c.opts.secret)
	}
	if c.opts.userAgent != "" {
		header.Set("User-Agent", c.opts.userAgent)
	}

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, u.String(), header)
	if err != nil {
		return fmt.Errorf("mihomo: failed to dial logs websocket: %w", err)
	}
	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var msg LogMessage
			if err := conn.ReadJSON(&msg); err != nil {
				return err
			}
			select {
			case logChan <- msg:
			default:
			}
		}
	}
}
