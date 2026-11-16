package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_ExplicitFile_NotFound(t *testing.T) {
	_, err := LoadConfig("non_existent_file.yaml", filepath.Join(t.TempDir(), "default.yaml"))
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

	cfg, err := LoadConfig(cfgFile, filepath.Join(tmpDir, "default.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.ApiURL != "http://192.168.1.1:9090" {
		t.Errorf("expected ApiURL 'http://192.168.1.1:9090', got %q", cfg.ApiURL)
	}
	if cfg.Secret != "test-secret" {
		t.Errorf("expected Secret 'test-secret', got %q", cfg.Secret)
	}
	if LoadedConfigPath() != cfgFile {
		t.Errorf("expected LoadedConfigPath %q, got %q", cfgFile, LoadedConfigPath())
	}
}

func TestLoadConfig_EmptyDefaultPath_FailsFast(t *testing.T) {
	if _, err := LoadConfig("", ""); err == nil {
		t.Fatalf("expected error when both explicitPath and defaultPath are empty, got nil")
	}
}

func TestLoadConfig_DefaultPath_NotExist_UsesDefaultsAndBindsSavePath(t *testing.T) {
	defaultPath := filepath.Join(t.TempDir(), "etc", "mihotui", "config.yaml")

	t.Setenv("MIHOMO_API_URL", "http://env-host:9090")
	t.Setenv("MIHOMO_SECRET", "env-secret")

	cfg, err := LoadConfig("", defaultPath)
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
	if LoadedConfigPath() != defaultPath {
		t.Errorf("expected LoadedConfigPath %q, got %q", defaultPath, LoadedConfigPath())
	}
}

func TestLoadConfig_ExplicitFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "broken.yaml")
	if err := os.WriteFile(cfgFile, []byte("invalid: [yaml: broken"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadConfig(cfgFile, filepath.Join(tmpDir, "default.yaml"))
	if err == nil {
		t.Fatalf("expected error for invalid yaml, got nil")
	}
}

func TestLoadConfig_DefaultPath_InvalidYAML(t *testing.T) {
	defaultFile := filepath.Join(t.TempDir(), "etc", "mihotui", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(defaultFile), 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(defaultFile, []byte("api_url: [invalid yaml"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadConfig("", defaultFile)
	if err == nil {
		t.Fatalf("expected error when default suite config has invalid yaml, got nil")
	}
}

func TestSaveConfig_And_Subscriptions(t *testing.T) {
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "saved_config.yaml")

	cfg := &AppConfig{
		ApiURL:        "http://10.0.0.1:9090",
		Secret:        "my-secret",
		ActiveProfile: "work.yaml",
		Subscriptions: []Subscription{
			{ID: "sub1", Name: "Sub1", URL: "http://example.com/sub"},
		},
	}

	SetLoadedConfigPath(cfgFile)
	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	if LoadedConfigPath() != cfgFile {
		t.Errorf("expected LoadedConfigPath %q, got %q", cfgFile, LoadedConfigPath())
	}

	loaded, err := LoadConfig(cfgFile, cfgFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if loaded.ActiveProfile != "work.yaml" {
		t.Errorf("expected ActiveProfile 'work.yaml', got %q", loaded.ActiveProfile)
	}
	if loaded.ApiURL != "http://10.0.0.1:9090" {
		t.Errorf("expected ApiURL 'http://10.0.0.1:9090', got %q", loaded.ApiURL)
	}
	if len(loaded.Subscriptions) != 1 || loaded.Subscriptions[0].Name != "Sub1" {
		t.Errorf("unexpected subscriptions: %+v", loaded.Subscriptions)
	}
}

func TestSaveConfigTo(t *testing.T) {
	if err := SaveConfigTo(DefaultConfig(), ""); err == nil {
		t.Fatalf("expected error when targetPath is empty, got nil")
	}

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "custom", "test_config.yaml")

	cfg := &AppConfig{
		ApiURL: "http://127.0.0.1:9090",
		Secret: "secret123",
	}

	if err := SaveConfigTo(cfg, targetPath); err != nil {
		t.Fatalf("SaveConfigTo failed: %v", err)
	}

	if LoadedConfigPath() != targetPath {
		t.Errorf("expected LoadedConfigPath %q, got %q", targetPath, LoadedConfigPath())
	}

	loaded, err := LoadConfig(targetPath, targetPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if loaded.Secret != "secret123" {
		t.Errorf("expected secret123, got %s", loaded.Secret)
	}
}
