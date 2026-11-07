package profile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseUserinfo(t *testing.T) {
	raw := "upload=1073741824; download=21474836480; total=107374182400; expire=1735689600"
	info := ParseUserinfo(raw)
	if info == nil {
		t.Fatal("expected non-nil SubUserInfo")
	}

	if info.Upload != 1073741824 {
		t.Errorf("expected upload 1073741824, got %d", info.Upload)
	}
	if info.Download != 21474836480 {
		t.Errorf("expected download 21474836480, got %d", info.Download)
	}
	if info.Total != 107374182400 {
		t.Errorf("expected total 107374182400, got %d", info.Total)
	}
	if info.Expire.Unix() != 1735689600 {
		t.Errorf("expected expire 1735689600, got %d", info.Expire.Unix())
	}
}

func TestManager_AddUpdateDelete(t *testing.T) {
	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "mihomo-profile-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Mock Subscription HTTP server
	validYAML := `
port: 7890
mode: rule
proxies:
  - name: "Node-Mock"
    type: ss
    server: 1.1.1.1
    port: 8388
    cipher: aes-128-gcm
    password: pass
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("expected non-empty User-Agent")
		}
		w.Header().Set("subscription-userinfo", "upload=1000; download=2000; total=100000; expire=1800000000")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(validYAML))
	}))
	defer server.Close()

	mgr, err := NewManager(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	mgr.metaPath = filepath.Join(tempDir, "subscriptions.yaml")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Add subscription
	sub, err := mgr.AddSubscription(ctx, "Test Provider", server.URL)
	if err != nil {
		t.Fatalf("AddSubscription failed: %v", err)
	}

	if sub.Name != "Test Provider" {
		t.Errorf("expected name Test Provider, got %s", sub.Name)
	}
	if sub.UserInfo == nil || sub.UserInfo.Total != 100000 {
		t.Errorf("unexpected userInfo: %+v", sub.UserInfo)
	}

	// Verify file was written
	filePath := filepath.Join(tempDir, sub.FileName)
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("target config file does not exist: %v", err)
	}

	// 2. List Profiles
	profiles, err := mgr.ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles failed: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(profiles))
	}
	if profiles[0].Subscription == nil {
		t.Error("expected correlated subscription in profile")
	}

	// 3. Update subscription
	updated, err := mgr.UpdateSubscription(ctx, sub.ID)
	if err != nil {
		t.Fatalf("UpdateSubscription failed: %v", err)
	}
	if updated.UpdatedAt.IsZero() {
		t.Error("expected updated timestamp")
	}

	// 4. Delete subscription
	err = mgr.DeleteSubscription(sub.ID, true)
	if err != nil {
		t.Fatalf("DeleteSubscription failed: %v", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("expected target file to be removed")
	}
}
