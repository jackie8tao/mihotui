package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// AppPaths holds resolved runtime paths for the self-contained suite layout.
type AppPaths struct {
	AppRoot string // <bundle_root>
	BinDir  string // <bundle_root>/bin
	EtcDir  string // <bundle_root>/etc
}

// MihomoBinPath returns the path to the bundled Mihomo core binary.
func (a AppPaths) MihomoBinPath() string {
	return filepath.Join(a.BinDir, "mihomo")
}

// MihomoEtcDir returns the configuration directory for the Mihomo core.
func (a AppPaths) MihomoEtcDir() string {
	return filepath.Join(a.EtcDir, "mihomo")
}

// AppEtcDir returns the configuration directory for mihotui itself.
func (a AppPaths) AppEtcDir() string {
	return filepath.Join(a.EtcDir, "mihotui")
}

// NewAppPaths constructs standard AppPaths for any given suite root directory.
func NewAppPaths(appRoot string) *AppPaths {
	return &AppPaths{
		AppRoot: appRoot,
		BinDir:  filepath.Join(appRoot, "bin"),
		EtcDir:  filepath.Join(appRoot, "etc"),
	}
}

// ResolveAppPaths derives the suite directory paths based on the executable's real location.
// Assumed self-contained bundle directory layout:
//
//	<appRoot>/
//	├── bin/
//	│   ├── mihotui
//	│   └── mihomo
//	└── etc/
//	    ├── mihotui/
//	    │   ├── config.yaml
//	    │   └── subscriptions/
//	    └── mihomo/
//	        └── config.yaml
//
// If the executable is in a "bin" folder, appRoot is its parent folder.
// Otherwise, appRoot is the executable's directory.
func ResolveAppPaths() (*AppPaths, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("config: resolve executable path: %w", err)
	}

	// Resolve symlinks (e.g. /usr/local/bin/mihotui -> /usr/local/mihotui/bin/mihotui)
	realExecPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return nil, fmt.Errorf("config: eval executable symlink %q: %w", execPath, err)
	}

	dir := filepath.Dir(realExecPath)
	appRoot := dir
	if filepath.Base(dir) == "bin" {
		appRoot = filepath.Dir(dir)
	}

	return NewAppPaths(appRoot), nil
}
