package installer

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jackie8tao/mihotui/internal/config"
)

func TestResolveAssetName(t *testing.T) {
	tests := []struct {
		goos    string
		goarch  string
		version string
		want    string
		wantErr bool
	}{
		{"linux", "amd64", "v1.19.33", "mihomo-linux-amd64-compatible-v1.19.33.gz", false},
		{"linux", "arm64", "v1.19.33", "mihomo-linux-arm64-v1.19.33.gz", false},
		{"linux", "arm", "v1.19.33", "mihomo-linux-armv7-v1.19.33.gz", false},
		{"linux", "386", "v1.19.33", "mihomo-linux-386-v1.19.33.gz", false},
		{"darwin", "arm64", "v1.19.33", "mihomo-darwin-arm64-v1.19.33.gz", false},
		{"darwin", "amd64", "v1.19.33", "mihomo-darwin-amd64-compatible-v1.19.33.gz", false},
		{"unknown", "arch", "v1.19.33", "", true},
	}

	for _, tt := range tests {
		got, err := resolveAssetName(tt.goos, tt.goarch, tt.version)
		if (err != nil) != tt.wantErr {
			t.Errorf("ResolveAssetName(%s, %s) error = %v, wantErr %v", tt.goos, tt.goarch, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ResolveAssetName(%s, %s) = %s, want %s", tt.goos, tt.goarch, got, tt.want)
		}
	}
}

func TestDownloadAndExtract_MockServer(t *testing.T) {
	// Create mock binary payload
	mockBinary := "#!/bin/sh\necho 'Mihomo Mock Version v1.0.0'\n"

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write([]byte(mockBinary)); err != nil {
		t.Fatalf("failed to gzip mock binary: %v", err)
	}
	_ = gw.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}))
	defer ts.Close()

	tmpDir := t.TempDir()
	targetBin := filepath.Join(tmpDir, "mihomo")

	ctx := context.Background()
	if err := downloadAndExtract(ctx, ts.URL, targetBin); err != nil {
		t.Fatalf("downloadAndExtract failed: %v", err)
	}

	content, err := os.ReadFile(targetBin)
	if err != nil {
		t.Fatalf("failed to read extracted binary: %v", err)
	}
	if string(content) != mockBinary {
		t.Errorf("extracted content mismatch. got %q, want %q", string(content), mockBinary)
	}
}

func TestInstall_Validation(t *testing.T) {
	_, err := Install(context.Background(), Options{})
	if err == nil {
		t.Fatal("expected error when opts.Paths is nil, got nil")
	}

	_, err = Install(context.Background(), Options{Paths: &config.AppPaths{}})
	if err == nil {
		t.Fatal("expected error when opts.Paths.AppRoot is empty, got nil")
	}
}

func TestInstall_MockFlow(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("skipping install mock flow on non-UNIX platform: " + runtime.GOOS)
	}

	mockBinary := "#!/bin/sh\necho 'Mihomo Mock Core v1.19.33'\n"
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, _ = gw.Write([]byte(mockBinary))
	_ = gw.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}))
	defer ts.Close()

	tmpDir := t.TempDir()
	opts := Options{
		Paths:   config.NewAppPaths(tmpDir),
		Version: "v1.19.33",
		Mirror:  ts.URL + "?target=", // will result in ts.URL?target=/https://...
		Force:   true,
	}

	res, err := Install(context.Background(), opts)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if res.MihomoBin == "" || !strings.HasSuffix(res.MihomoBin, "mihomo") {
		t.Errorf("unexpected MihomoBin: %s", res.MihomoBin)
	}
	if _, err := os.Stat(res.MihomoConfig); err != nil {
		t.Errorf("expected config file to exist at %s", res.MihomoConfig)
	}
}

func TestInstall_LocalArchive_Gzip(t *testing.T) {
	mockBinary := "mock-mihomo-gz-binary"
	archivePath := filepath.Join(t.TempDir(), "mihomo-linux-amd64-v1.19.33.gz")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	if _, err := gw.Write([]byte(mockBinary)); err != nil {
		t.Fatal(err)
	}
	_ = gw.Close()
	_ = f.Close()

	targetRoot := t.TempDir()
	res, err := Install(context.Background(), Options{
		Paths:       config.NewAppPaths(targetRoot),
		ArchivePath: archivePath,
	})
	if err != nil {
		t.Fatalf("Install from .gz archive failed: %v", err)
	}

	got, err := os.ReadFile(res.MihomoBin)
	if err != nil {
		t.Fatalf("failed to read installed binary: %v", err)
	}
	if string(got) != mockBinary {
		t.Errorf("binary content = %q, want %q", string(got), mockBinary)
	}
	if _, err := os.Stat(res.MihomoConfig); err != nil {
		t.Errorf("expected config file at %s: %v", res.MihomoConfig, err)
	}
	if _, err := os.Stat(res.MihotuiConfig); err != nil {
		t.Errorf("expected mihotui config file at %s: %v", res.MihotuiConfig, err)
	}
}

func TestInstall_LocalArchive_Errors(t *testing.T) {
	tmpDir := t.TempDir()
	paths := config.NewAppPaths(tmpDir)

	// 1. Nonexistent file
	if _, err := Install(context.Background(), Options{
		Paths:       paths,
		ArchivePath: filepath.Join(tmpDir, "missing.gz"),
	}); err == nil {
		t.Error("expected error for missing archive file, got nil")
	}

	// 2. Directory passed as archive
	if _, err := Install(context.Background(), Options{
		Paths:       paths,
		ArchivePath: tmpDir,
	}); err == nil {
		t.Error("expected error when archive path is a directory, got nil")
	}

	// 3. Invalid gzip file content
	badGz := filepath.Join(tmpDir, "corrupt.gz")
	_ = os.WriteFile(badGz, []byte("not-a-gzip-stream"), 0644)
	if _, err := Install(context.Background(), Options{
		Paths:       paths,
		ArchivePath: badGz,
	}); err == nil {
		t.Error("expected error for invalid gzip stream, got nil")
	}
}
