package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// AppPaths holds resolved runtime paths relative to mihotui installation.
type AppPaths struct {
	AppRoot   string // e.g. /usr/local/mihotui
	BinDir    string // e.g. /usr/local/mihotui/bin
	EtcDir    string // e.g. /usr/local/mihotui/etc
	MihomoDir string // e.g. /usr/local/mihotui/etc/mihomo
	MihomoBin string // e.g. /usr/local/mihotui/bin/mihomo
	IsBundle  bool   // Whether current running binary is within a self-contained bundle
}

// ResolveAppPaths determines relative paths based on the executable's real location.
func ResolveAppPaths() (AppPaths, error) {
	execPath, err := os.Executable()
	if err != nil {
		return AppPaths{}, fmt.Errorf("failed to determine executable path: %w", err)
	}

	// Resolve symlinks (e.g. /usr/local/bin/mihotui -> /usr/local/mihotui/bin/mihotui)
	realExecPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return AppPaths{}, fmt.Errorf("failed to resolve symlink for executable %q: %w", execPath, err)
	}

	binDir := filepath.Dir(realExecPath)
	appRoot := filepath.Dir(binDir)

	// Check if appRoot conforms to the self-contained suite layout
	etcDir := filepath.Join(appRoot, "etc")
	isBundle := false
	if fi, err := os.Stat(etcDir); err == nil && fi.IsDir() {
		isBundle = true
	} else if filepath.Base(binDir) == "bin" {
		// Even if etc doesn't exist yet, if binary is in a /bin folder, treat as bundle root
		isBundle = true
	}

	return AppPaths{
		AppRoot:   appRoot,
		BinDir:    filepath.Join(appRoot, "bin"),
		EtcDir:    filepath.Join(appRoot, "etc"),
		MihomoDir: filepath.Join(appRoot, "etc", "mihomo"),
		MihomoBin: filepath.Join(appRoot, "bin", "mihomo"),
		IsBundle:  isBundle,
	}, nil
}

