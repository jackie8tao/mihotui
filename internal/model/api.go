package model

import (
	"time"

	"mihomo-tui/pkg/mihomo"
)

// Re-export core Mihomo SDK types for internal use
type (
	Version         = mihomo.Version
	Config          = mihomo.Config
	TunCfg          = mihomo.TunCfg
	DelayHistory    = mihomo.DelayHistory
	ProxyItem       = mihomo.ProxyItem
	ProxiesResponse = mihomo.ProxiesResponse
	DelayResponse   = mihomo.DelayResponse
	Traffic         = mihomo.Traffic
	Memory          = mihomo.Memory
)

// SubUserInfo represents subscription traffic quota & expiration header.
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
