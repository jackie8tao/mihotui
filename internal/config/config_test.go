package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_ExplicitFile_NotFound(t *testing.T) {
	_, err := LoadConfig("non_existent_file.yaml")
	if err == nil {
		t.Fatalf("expected error for non-existent explicit config path, got nil")
	}
}

func TestLoadConfig_ExplicitFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "custom.yaml")
	content := `
api_url: "http://192.168.1.1:9090"
secret: "test-secret"
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg, err := LoadConfig(cfgFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.ApiURL != "http://192.168.1.1:9090" {
		t.Errorf("expected ApiURL 'http://192.168.1.1:9090', got %q", cfg.ApiURL)
	}
	if cfg.Secret != "test-secret" {
		t.Errorf("expected Secret 'test-secret', got %q", cfg.Secret)
	}
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	t.Setenv("MIHOMO_API_URL", "http://env-host:9090")
	t.Setenv("MIHOMO_SECRET", "env-secret")

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.ApiURL != "http://env-host:9090" {
		t.Errorf("expected ApiURL 'http://env-host:9090', got %q", cfg.ApiURL)
	}
	if cfg.Secret != "env-secret" {
		t.Errorf("expected Secret 'env-secret', got %q", cfg.Secret)
	}
}

