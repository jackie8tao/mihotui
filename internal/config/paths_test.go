package config

import (
	"testing"
)

func TestResolveAppPaths(t *testing.T) {
	paths, err := ResolveAppPaths()
	if err != nil {
		t.Fatalf("unexpected error resolving app paths: %v", err)
	}
	if paths == nil {
		t.Fatalf("expected non-nil paths")
	}
	if paths.AppRoot == "" {
		t.Errorf("expected non-empty AppRoot")
	}
	if paths.BinDir == "" {
		t.Errorf("expected non-empty BinDir")
	}
	if paths.MihomoBin == "" {
		t.Errorf("expected non-empty MihomoBin")
	}
}

func TestNewAppPaths(t *testing.T) {
	root := "/opt/test-suite"
	paths := NewAppPaths(root)
	if paths == nil {
		t.Fatalf("expected non-nil AppPaths")
	}
	if paths.AppRoot != root {
		t.Errorf("expected AppRoot %q, got %q", root, paths.AppRoot)
	}
	if paths.BinDir == "" || paths.EtcDir == "" || paths.MihomoDir == "" || paths.MihomoBin == "" {
		t.Errorf("expected all path fields to be populated, got %+v", paths)
	}
}

