package config

import "time"

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
