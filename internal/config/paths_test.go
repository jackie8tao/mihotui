package config

import (
	"testing"
)

func TestResolveAppPaths(t *testing.T) {
	paths := ResolveAppPaths()
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

