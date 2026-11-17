package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// AppPaths holds resolved runtime paths for the self-contained suite layout.
type AppPaths struct {
	AppRoot string // <bundle_root>
	BinDir  string // <bundle_root>/bin
	EtcDir  string // <bundle_root>/etc
}

func (a AppPaths) MihomoBinPath() string {
	return filepath.Join(a.BinDir, "mihomo")
}

func (a AppPaths) MihomoEtcDir() string {
	return filepath.Join(a.EtcDir, "mihomo")
}

// NewAppPaths constructs standard AppPaths for any given suite root directory.
func NewAppPaths(appRoot string) *AppPaths {
	binDir := filepath.Join(appRoot, "bin")
	etcDir := filepath.Join(appRoot, "etc")
	mihomoBin := filepath.Join(binDir, "mihomo")
	if runtime.GOOS == "windows" {
		mihomoBin += ".exe"
	}

	return &AppPaths{
		AppRoot: appRoot,
		BinDir:  binDir,
		EtcDir:  etcDir,
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
//	    └── mihomo/
//
// If the executable is in a "bin" folder, appRoot is its parent folder.
// Otherwise, appRoot is the executable's directory.
func ResolveAppPaths() (*AppPaths, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to determine executable path: %w", err)
	}

	// Resolve symlinks (e.g. /usr/local/bin/mihotui -> /usr/local/mihotui/bin/mihotui)
	realExecPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve symlink for executable %q: %w", execPath, err)
	}

	dir := filepath.Dir(realExecPath)
	appRoot := dir
	if filepath.Base(dir) == "bin" {
		appRoot = filepath.Dir(dir)
	}

	return NewAppPaths(appRoot), nil
}
