package model

import "time"

// Version represents the Mihomo version response.
type Version struct {
	Version string `json:"version"`
	Meta    bool   `json:"meta,omitempty"`
}

// Config represents runtime configurations of Mihomo.
type Config struct {
	Port        int    `json:"port"`
	SocksPort   int    `json:"socks-port"`
	RedirPort   int    `json:"redir-port"`
	TproxyPort  int    `json:"tproxy-port"`
	MixedPort   int    `json:"mixed-port"`
	Tun         TunCfg `json:"tun"`
	Mode        string `json:"mode"`
	LogLevel    string `json:"log-level"`
	AllowLan    bool   `json:"allow-lan"`
	BindAddress string `json:"bind-address"`
}

// TunCfg represents the tun config part.
type TunCfg struct {
	Enable bool   `json:"enable"`
	Device string `json:"device"`
	Stack  string `json:"stack"`
}

// DelayHistory represents delay test history for a proxy.
type DelayHistory struct {
	Time  time.Time `json:"time"`
	Delay int       `json:"delay"`
}

// ProxyItem represents a proxy node or proxy group.
type ProxyItem struct {
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	Now     string         `json:"now,omitempty"`
	All     []string       `json:"all,omitempty"`
	History []DelayHistory `json:"history,omitempty"`
	Udp     bool           `json:"udp,omitempty"`
	Extra   map[string]any `json:"extra,omitempty"`

	// Client-side helper fields
	TestingDelay bool `json:"-"`
}

// ProxiesResponse represents the response from GET /proxies.
type ProxiesResponse struct {
	Proxies map[string]ProxyItem `json:"proxies"`
}

// DelayResponse represents the response from GET /proxies/:name/delay.
type DelayResponse struct {
	Delay   int    `json:"delay"`
	Message string `json:"message,omitempty"`
}

// Traffic represents real-time bandwidth data from /traffic WS.
type Traffic struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

// Memory represents memory usage from /memory.
type Memory struct {
	Inuse   int64 `json:"inuse"`
	Osalloc int64 `json:"osalloc"`
}

// ReloadConfigPayload is payload for PUT /configs?force=true.
type ReloadConfigPayload struct {
	Path    string `json:"path"`
	Payload string `json:"payload,omitempty"`
}

// SubUserInfo represents subscription traffic quota & expiration header.
// Header format: upload=1073741824; download=21474836480; total=107374182400; expire=1735689600
type SubUserInfo struct {
	Upload   int64     `json:"upload" yaml:"upload"`
	Download int64     `json:"download" yaml:"download"`
	Total    int64     `json:"total" yaml:"total"`
	Expire   time.Time `json:"expire" yaml:"expire"`
}

// Subscription represents a managed subscription source.
type Subscription struct {
	ID        string       `json:"id" yaml:"id"`
	Name      string       `json:"name" yaml:"name"`
	URL       string       `json:"url" yaml:"url"`
	FileName  string       `json:"file_name" yaml:"file_name"`
	UpdatedAt time.Time    `json:"updated_at" yaml:"updated_at"`
	UserInfo  *SubUserInfo `json:"user_info,omitempty" yaml:"user_info,omitempty"`
}

// Profile represents a local configuration file (.yaml).
type Profile struct {
	Name         string        `json:"name"`
	FileName     string        `json:"file_name"`
	FullPath     string        `json:"full_path"`
	Size         int64         `json:"size"`
	ModTime      time.Time     `json:"mod_time"`
	IsActive     bool          `json:"is_active"`
	Subscription *Subscription `json:"subscription,omitempty"`
}

