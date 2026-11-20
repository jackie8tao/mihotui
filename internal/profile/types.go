package profile

import (
	"time"

	"github.com/jackie8tao/mihotui/internal/config"
)

// Profile represents a local configuration file (.yaml).
type Profile struct {
	SubName      string               `json:"sub_name"`
	FileName     string               `json:"file_name"`
	FullPath     string               `json:"full_path"`
	Size         int64                `json:"size"`
	ModTime      time.Time            `json:"mod_time"`
	IsActive     bool                 `json:"is_active"`
	Subscription *config.Subscription `json:"subscription,omitempty"`
}
