package mihomo

import (
	"fmt"
	"time"
)

// Version represents the Mihomo version information.
type Version struct {
	Version string `json:"version"`
	Meta    bool   `json:"meta,omitempty"`
}

// Config represents runtime configurations of the Mihomo core.
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

// TunCfg represents the TUN interface configuration.
type TunCfg struct {
	Enable bool   `json:"enable"`
	Device string `json:"device"`
	Stack  string `json:"stack"`
}

// DelayHistory represents a record of latency test result.
type DelayHistory struct {
	Time  time.Time `json:"time"`
	Delay int       `json:"delay"`
}

// ProxyItem represents a proxy node or selector group.
type ProxyItem struct {
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	Now     string         `json:"now,omitempty"`
	All     []string       `json:"all,omitempty"`
	History []DelayHistory `json:"history,omitempty"`
	Udp     bool           `json:"udp,omitempty"`
	Extra   map[string]any `json:"extra,omitempty"`

	// TestingDelay indicates whether latency test is in progress.
	TestingDelay bool `json:"-"`
}

// ProxiesResponse represents the response payload for GET /proxies.
type ProxiesResponse struct {
	Proxies map[string]ProxyItem `json:"proxies"`
}

// DelayResponse represents the response payload for GET /proxies/:name/delay.
type DelayResponse struct {
	Delay   int    `json:"delay"`
	Message string `json:"message,omitempty"`
}

// Traffic represents real-time bandwidth statistics (bytes/s).
type Traffic struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

// Memory represents memory usage statistics in bytes.
type Memory struct {
	Inuse   int64 `json:"inuse"`
	Osalloc int64 `json:"osalloc"`
}

// LogMessage represents a real-time log stream message.
type LogMessage struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

// FormatBytes formats byte counts into human-readable strings (e.g. 1.25 MB).
func FormatBytes(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
		tb = gb * 1024
	)

	switch {
	case bytes >= tb:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(tb))
	case bytes >= gb:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatSpeed formats byte rate into human-readable transfer rate (e.g. 2.40 MB/s).
func FormatSpeed(bytesPerSec int64) string {
	return FormatBytes(bytesPerSec) + "/s"
}

// FormatDelay formats latency in ms into readable string.
func FormatDelay(delay int) string {
	if delay <= 0 {
		return "Timeout"
	}
	return fmt.Sprintf("%d ms", delay)
}
