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

func TestLoadConfig_IgnoreEnvOverrides(t *testing.T) {
	t.Setenv("MIHOMO_API_URL", "http://env-host:9090")
	t.Setenv("MIHOMO_SECRET", "env-secret")

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Environment variables should NOT override configuration
	if cfg.ApiURL != "http://127.0.0.1:9090" {
		t.Errorf("expected ApiURL 'http://127.0.0.1:9090', got %q", cfg.ApiURL)
	}
	if cfg.Secret != "" {
		t.Errorf("expected empty Secret, got %q", cfg.Secret)
	}
}

func TestLoadConfig_ExplicitFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "broken.yaml")
	if err := os.WriteFile(cfgFile, []byte("invalid: [yaml: broken"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadConfig(cfgFile)
	if err == nil {
		t.Fatalf("expected error for invalid yaml, got nil")
	}
}

func TestLoadConfig_Discovery_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	// Override XDG_CONFIG_HOME or HOME to point to tmpDir
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, defaultConfigDir)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	brokenFile := filepath.Join(cfgDir, defaultConfigFile)
	if err := os.WriteFile(brokenFile, []byte("api_url: [invalid yaml"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadConfig("")
	if err == nil {
		t.Fatalf("expected error when candidate config has invalid yaml, got nil")
	}
}

