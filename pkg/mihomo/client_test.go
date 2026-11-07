package mihomo

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_REST(t *testing.T) {
	mux := http.NewServeMux()

	// Mock /version
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Unauthorized"})
			return
		}
		_ = json.NewEncoder(w).Encode(Version{Version: "v1.18.0", Meta: true})
	})

	// Mock /configs
	mux.HandleFunc("/configs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(Config{
				MixedPort: 7890,
				Mode:      "rule",
				LogLevel:  "info",
			})
			return
		}
		if r.Method == http.MethodPatch {
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["mode"] != "global" {
				t.Errorf("expected mode global, got %v", body["mode"])
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == http.MethodPut {
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["path"] != "/etc/mihomo/config.yaml" {
				t.Errorf("expected /etc/mihomo/config.yaml, got %v", body["path"])
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	})

	// Mock /proxies
	mux.HandleFunc("/proxies", func(w http.ResponseWriter, r *http.Request) {
		res := ProxiesResponse{
			Proxies: map[string]ProxyItem{
				"GLOBAL": {
					Name: "GLOBAL",
					Type: "Selector",
					Now:  "ProxyGroup",
					All:  []string{"ProxyGroup", "DIRECT"},
				},
				"ProxyGroup": {
					Name: "ProxyGroup",
					Type: "Selector",
					Now:  "Node-1",
					All:  []string{"Node-1", "Node-2"},
				},
				"Node-1": {
					Name: "Node-1",
					Type: "Shadowsocks",
					History: []DelayHistory{
						{Time: time.Now(), Delay: 80},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	// Mock /proxies/ProxyGroup
	mux.HandleFunc("/proxies/ProxyGroup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			var payload map[string]string
			_ = json.NewDecoder(r.Body).Decode(&payload)
			if payload["name"] != "Node-2" {
				t.Errorf("expected Node-2, got %s", payload["name"])
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	})

	// Mock /proxies/Node-1/delay
	mux.HandleFunc("/proxies/Node-1/delay", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(DelayResponse{Delay: 66})
	})

	// Mock /memory
	mux.HandleFunc("/memory", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(Memory{Inuse: 20480, Osalloc: 40960})
	})

	// Mock 404
	mux.HandleFunc("/notfound", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	ctx := context.Background()

	// 1. Test Client Initialization with Functional Options
	cli, err := New(server.URL,
		WithSecret("test-secret"),
		WithTimeout(5*time.Second),
		WithUserAgent("custom-agent"),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// 2. Test GetVersion
	ver, err := cli.GetVersion(ctx)
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}
	if ver.Version != "v1.18.0" || !ver.Meta {
		t.Errorf("unexpected version result: %+v", ver)
	}

	// 3. Test Unauthorized with wrong secret
	badCli, _ := New(server.URL, WithSecret("wrong-secret"))
	_, err = badCli.GetVersion(ctx)
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized match, got %v", err)
	}

	// 4. Test GetConfig
	cfg, err := cli.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if cfg.Mode != "rule" || cfg.MixedPort != 7890 {
		t.Errorf("unexpected config result: %+v", cfg)
	}

	// 5. Test UpdateMode
	err = cli.UpdateMode(ctx, "global")
	if err != nil {
		t.Fatalf("UpdateMode failed: %v", err)
	}

	// 6. Test ReloadConfig
	err = cli.ReloadConfig(ctx, "/etc/mihomo/config.yaml", true)
	if err != nil {
		t.Fatalf("ReloadConfig failed: %v", err)
	}

	// 7. Test GetProxies
	proxies, err := cli.GetProxies(ctx)
	if err != nil {
		t.Fatalf("GetProxies failed: %v", err)
	}
	if len(proxies.Proxies) != 3 {
		t.Errorf("expected 3 proxies, got %d", len(proxies.Proxies))
	}

	// 8. Test SelectProxy
	err = cli.SelectProxy(ctx, "ProxyGroup", "Node-2")
	if err != nil {
		t.Fatalf("SelectProxy failed: %v", err)
	}

	// 9. Test TestDelay
	delay, err := cli.TestDelay(ctx, "Node-1", "http://test", 2000)
	if err != nil {
		t.Fatalf("TestDelay failed: %v", err)
	}
	if delay != 66 {
		t.Errorf("expected delay 66, got %d", delay)
	}

	// 10. Test GetMemory
	mem, err := cli.GetMemory(ctx)
	if err != nil {
		t.Fatalf("GetMemory failed: %v", err)
	}
	if mem.Inuse != 20480 {
		t.Errorf("expected inuse 20480, got %d", mem.Inuse)
	}
}

func TestFormatHelpers(t *testing.T) {
	if s := FormatBytes(1024); s != "1.00 KB" {
		t.Errorf("expected 1.00 KB, got %s", s)
	}
	if s := FormatSpeed(2048); s != "2.00 KB/s" {
		t.Errorf("expected 2.00 KB/s, got %s", s)
	}
	if s := FormatDelay(150); s != "150 ms" {
		t.Errorf("expected 150 ms, got %s", s)
	}
	if s := FormatDelay(0); s != "Timeout" {
		t.Errorf("expected Timeout, got %s", s)
	}
}
