package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// AppPaths holds resolved runtime paths for the self-contained suite layout.
type AppPaths struct {
	AppRoot   string // <bundle_root>
	BinDir    string // <bundle_root>/bin
	EtcDir    string // <bundle_root>/etc
	MihomoDir string // <bundle_root>/etc/mihomo
	MihomoBin string // <bundle_root>/bin/mihomo
}

// ResolveAppPaths derives the suite directory paths based on the executable's real location.
// Assumed self-contained bundle directory layout:
//   <appRoot>/
//   ├── bin/
//   │   ├── mihotui (realExecPath)
//   │   └── mihomo  (MihomoBin)
//   └── etc/
//       └── mihomo/ (MihomoDir)
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

	return &AppPaths{
		AppRoot:   appRoot,
		BinDir:    filepath.Join(appRoot, "bin"),
		EtcDir:    filepath.Join(appRoot, "etc"),
		MihomoDir: filepath.Join(appRoot, "etc", "mihomo"),
		MihomoBin: filepath.Join(appRoot, "bin", "mihomo"),
	}, nil
}

