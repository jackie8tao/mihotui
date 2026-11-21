package installer

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jackie8tao/mihotui/internal/config"
)

const (
	DefaultMihomoVersion = "v1.19.33"
	DefaultMihomoRepo    = "MetaCubeX/mihomo"
)

// Options represents parameters for installing Mihomo core.
type Options struct {
	AppPaths    *config.AppPaths // Resolved suite runtime paths (single source of truth)
	ArchivePath string           // Optional local gzip-compressed Mihomo release (.gz)
	Force       bool             // Force overwrite existing binary
}

// Install installs the Mihomo core binary and initializes the suite configuration directories.
func Install(ctx context.Context, opts Options) error {
	if opts.AppPaths == nil || strings.TrimSpace(opts.AppPaths.AppRoot) == "" {
		return errors.New("installer: invalid app paths")
	}

	// 1. Ensure suite directories exist
	needCreatedDirs := []string{
		opts.AppPaths.BinDir,
		opts.AppPaths.EtcDir,
		filepath.Join(opts.AppPaths.EtcDir, "mihomo"),
		filepath.Join(opts.AppPaths.EtcDir, "mihotui"),
	}
	for _, dir := range needCreatedDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("installer: create dir %q: %w", dir, err)
		}
	}

	// 2. Install core binary from local archive or remote release
	if err := installMihomoBinary(ctx, opts); err != nil {
		return fmt.Errorf("installer: %w", err)
	}

	// 3. Generate and install systemd service unit on Linux
	if err := installSystemdUnit(ctx, opts); err != nil {
		return fmt.Errorf("installer: %w", err)
	}
	return nil
}

func installMihomoBinary(ctx context.Context, opts Options) error {
	mihomoBinPath := opts.AppPaths.MihomoBinPath()
	if !opts.Force {
		if fi, err := os.Stat(mihomoBinPath); err == nil && !fi.IsDir() {
			fmt.Printf("Mihomo core binary already exists at %s (use --force to overwrite)\n", mihomoBinPath)
			return nil
		}
	}

	if archivePath := strings.TrimSpace(opts.ArchivePath); archivePath != "" {
		if err := installLocalArchive(archivePath, mihomoBinPath); err != nil {
			return err
		}
		return chmodExecutable(mihomoBinPath)
	}

	if err := installRemoteArchive(ctx, mihomoBinPath); err != nil {
		return err
	}
	return chmodExecutable(mihomoBinPath)
}

func chmodExecutable(path string) error {
	if err := os.Chmod(path, 0755); err != nil {
		return fmt.Errorf("chmod %q: %w", path, err)
	}
	return nil
}

var assetTargets = map[string]string{
	"linux/amd64":  "linux-amd64-compatible",
	"linux/arm64":  "linux-arm64",
	"linux/arm":    "linux-armv7",
	"linux/386":    "linux-386",
	"darwin/arm64": "darwin-arm64",
	"darwin/amd64": "darwin-amd64-compatible",
}

func installRemoteArchive(ctx context.Context, targetBinPath string) error {
	// Resolve the latest release tag via GitHub's /releases/latest HTTP redirect.
	// GitHub responds with 302 Found and a Location header ending in /releases/tag/<version>.
	latestURL := fmt.Sprintf("https://github.com/%s/releases/latest", DefaultMihomoRepo)
	latestReq, err := http.NewRequestWithContext(ctx, http.MethodHead, latestURL, nil)
	if err != nil {
		return fmt.Errorf("build latest request: %w", err)
	}
	latestReq.Header.Set("User-Agent", "mihotui-installer")

	latestClient := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	latestResp, err := latestClient.Do(latestReq)
	if err != nil {
		return fmt.Errorf("query latest release: %w", err)
	}
	latestResp.Body.Close()

	location := latestResp.Header.Get("Location")
	if location == "" {
		return fmt.Errorf("missing redirect location (HTTP %d)", latestResp.StatusCode)
	}

	_, version, ok := strings.Cut(location, "/releases/tag/")
	version = strings.TrimRight(strings.SplitN(version, "?", 2)[0], "/")
	if !ok || version == "" {
		return fmt.Errorf("parse release tag from %q", location)
	}

	target, ok := assetTargets[runtime.GOOS+"/"+runtime.GOARCH]
	if !ok {
		return fmt.Errorf("unsupported platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	assetName := fmt.Sprintf("mihomo-%s-%s.gz", target, version)
	downloadURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", DefaultMihomoRepo, version, assetName)

	downloadReq, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("build download request: %w", err)
	}
	downloadReq.Header.Set("User-Agent", "mihotui-installer")

	downloadClient := &http.Client{
		Timeout: 3 * time.Minute,
	}

	downloadResp, err := downloadClient.Do(downloadReq)
	if err != nil {
		return fmt.Errorf("download binary: %w", err)
	}
	defer downloadResp.Body.Close()

	if downloadResp.StatusCode != http.StatusOK {
		return fmt.Errorf("download binary: unexpected HTTP %d", downloadResp.StatusCode)
	}

	return extractGzip(downloadResp.Body, targetBinPath)
}

func installLocalArchive(archivePath, targetBinPath string) error {
	info, err := os.Stat(archivePath)
	if err != nil {
		return fmt.Errorf("stat archive %q: %w", archivePath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected .gz file, got directory %q", archivePath)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive %q: %w", archivePath, err)
	}
	defer f.Close()

	return extractGzip(f, targetBinPath)
}

func extractGzip(r io.Reader, targetBinPath string) error {
	// Mihomo releases for Linux/macOS are single gzip-compressed binaries.
	gzReader, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("read gzip: %w", err)
	}
	defer gzReader.Close()

	tmpPath := targetBinPath + ".tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("create temp binary: %w", err)
	}
	defer func() {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := io.Copy(tmpFile, gzReader); err != nil {
		return fmt.Errorf("extract binary: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp binary: %w", err)
	}

	if err := os.Rename(tmpPath, targetBinPath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}
	return nil
}
