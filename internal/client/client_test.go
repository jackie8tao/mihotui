package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mihomo-tui/internal/model"
)

func TestClient_REST(t *testing.T) {
	mux := http.NewServeMux()

	// Mock /version
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(model.Version{Version: "v1.18.0", Meta: true})
	})

	// Mock /configs
	mux.HandleFunc("/configs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(model.Config{
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
		res := model.ProxiesResponse{
			Proxies: map[string]model.ProxyItem{
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
					History: []model.DelayHistory{
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
		_ = json.NewEncoder(w).Encode(model.DelayResponse{Delay: 66})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	cli := New(server.URL, "test-secret")
	ctx := context.Background()

	// 1. Test GetVersion
	ver, err := cli.GetVersion(ctx)
	if err != nil {
		t.Fatalf("GetVersion failed: %v", err)
	}
	if ver.Version != "v1.18.0" || !ver.Meta {
		t.Errorf("unexpected version result: %+v", ver)
	}

	// 2. Test GetConfig
	cfg, err := cli.GetConfig(ctx)
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if cfg.Mode != "rule" || cfg.MixedPort != 7890 {
		t.Errorf("unexpected config result: %+v", cfg)
	}

	// 3. Test UpdateMode
	err = cli.UpdateMode(ctx, "global")
	if err != nil {
		t.Fatalf("UpdateMode failed: %v", err)
	}

	// 4. Test GetProxies
	proxies, err := cli.GetProxies(ctx)
	if err != nil {
		t.Fatalf("GetProxies failed: %v", err)
	}
	if len(proxies.Proxies) != 3 {
		t.Errorf("expected 3 proxies, got %d", len(proxies.Proxies))
	}

	// 5. Test SelectProxy
	err = cli.SelectProxy(ctx, "ProxyGroup", "Node-2")
	if err != nil {
		t.Fatalf("SelectProxy failed: %v", err)
	}

	// 6. Test TestDelay
	delay, err := cli.TestDelay(ctx, "Node-1", "http://test", 2000)
	if err != nil {
		t.Fatalf("TestDelay failed: %v", err)
	}
	if delay != 66 {
		t.Errorf("expected delay 66, got %d", delay)
	}

	// 7. Test ReloadConfig
	err = cli.ReloadConfig(ctx, "/etc/mihomo/config.yaml", true)
	if err != nil {
		t.Fatalf("ReloadConfig failed: %v", err)
	}
}
