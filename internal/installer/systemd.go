package installer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/coreos/go-systemd/v22/unit"

	"github.com/jackie8tao/mihotui/internal/config"
)

const (
	// SystemdUnitName is the installed systemd service unit filename for Mihomo core.
	SystemdUnitName = "mihomo.service"
	// SystemdUnitDir is the standard Linux system-wide systemd unit directory.
	SystemdUnitDir = "/etc/systemd/system"
)

// installSystemdUnit generates and installs /etc/systemd/system/mihomo.service using go-systemd
// and triggers a systemd daemon-reload over D-Bus.
// Implicit assumption: systemd is only applicable on Linux; non-Linux hosts skip this step.
func installSystemdUnit(ctx context.Context, opts Options) error {
	if runtime.GOOS != "linux" {
		return nil
	}

	unitPath := filepath.Join(SystemdUnitDir, SystemdUnitName)
	if !opts.Force {
		if fi, err := os.Stat(unitPath); err == nil && !fi.IsDir() {
			fmt.Printf("Systemd unit already exists at %s (use --force to overwrite)\n", unitPath)
			return nil
		}
	}

	fi, err := os.Stat(SystemdUnitDir)
	if err != nil {
		return fmt.Errorf("systemd unit directory %q not accessible: %w", SystemdUnitDir, err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("systemd unit path %q is not a directory", SystemdUnitDir)
	}

	content, err := renderMihomoServiceUnit(opts.AppPaths)
	if err != nil {
		return fmt.Errorf("render systemd unit: %w", err)
	}

	if err := writeUnitFileAtomically(unitPath, content); err != nil {
		return err
	}

	return reloadSystemdDaemon(ctx)
}

// renderMihomoServiceUnit builds and serializes the Mihomo systemd unit using go-systemd/unit.
// systemd requires absolute executable and working paths in ExecStart.
func renderMihomoServiceUnit(paths *config.AppPaths) ([]byte, error) {
	mihomoBin, err := filepath.Abs(paths.MihomoBinPath())
	if err != nil {
		return nil, fmt.Errorf("resolve absolute mihomo binary path: %w", err)
	}
	mihomoEtcDir, err := filepath.Abs(paths.MihomoEtcDir())
	if err != nil {
		return nil, fmt.Errorf("resolve absolute mihomo config dir: %w", err)
	}

	sections := []*unit.UnitSection{
		{
			Section: "Unit",
			Entries: []*unit.UnitEntry{
				{Name: "Description", Value: "Mihomo Daemon, A rule-based proxy in Go."},
				{Name: "After", Value: "network.target network-online.target"},
				{Name: "Wants", Value: "network-online.target"},
			},
		},
		{
			Section: "Service",
			Entries: []*unit.UnitEntry{
				{Name: "Type", Value: "simple"},
				{Name: "User", Value: "root"},
				{Name: "LimitNPROC", Value: "500"},
				{Name: "LimitNOFILE", Value: "1000000"},
				{Name: "CapabilityBoundingSet", Value: "CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW"},
				{Name: "AmbientCapabilities", Value: "CAP_NET_ADMIN CAP_NET_BIND_SERVICE CAP_NET_RAW"},
				{Name: "Restart", Value: "always"},
				{Name: "RestartSec", Value: "3"},
				{Name: "ExecStartPre", Value: "/usr/bin/sleep 1"},
				{Name: "ExecStart", Value: fmt.Sprintf("%s -d %s", mihomoBin, mihomoEtcDir)},
				{Name: "ExecReload", Value: "/bin/kill -HUP $MAINPID"},
			},
		},
		{
			Section: "Install",
			Entries: []*unit.UnitEntry{
				{Name: "WantedBy", Value: "multi-user.target"},
			},
		},
	}

	data, err := io.ReadAll(unit.SerializeSections(sections))
	if err != nil {
		return nil, fmt.Errorf("serialize unit sections: %w", err)
	}
	return data, nil
}

// writeUnitFileAtomically writes a systemd unit file via a temporary file and atomic rename.
func writeUnitFileAtomically(targetPath string, data []byte) error {
	tmpPath := targetPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp unit file %q: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace unit file %q: %w", targetPath, err)
	}
	return nil
}

// reloadSystemdDaemon connects to the system D-Bus and triggers a systemd daemon-reload.
func reloadSystemdDaemon(ctx context.Context) error {
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return fmt.Errorf("connect systemd dbus: %w", err)
	}
	defer conn.Close()

	if err := conn.ReloadContext(ctx); err != nil {
		return fmt.Errorf("reload systemd daemon: %w", err)
	}
	return nil
}
