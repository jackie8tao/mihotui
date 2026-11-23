package profile

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/jackie8tao/mihotui/internal/config"
)

// parseUserinfo parses the Subscription-Userinfo header.
func parseUserinfo(raw string) *config.SubUserInfo {
	if raw == "" {
		return nil
	}

	info := &config.SubUserInfo{}
	parts := strings.Split(raw, ";")
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.ToLower(strings.TrimSpace(kv[0]))
		val, _ := strconv.ParseInt(strings.TrimSpace(kv[1]), 10, 64)

		switch k {
		case "upload":
			info.Upload = val
		case "download":
			info.Download = val
		case "total":
			info.Total = val
		case "expire":
			if val > 0 {
				info.Expire = time.Unix(val, 0)
			}
		}
	}

	return info
}

// downloadAndValidate downloads config from URL, verifies YAML, and writes atomically.
func downloadAndValidate(ctx context.Context, rawURL, targetPath string) (*config.SubUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("User-Agent", "Clash.Meta; mihomo/1.18.0; mihotui")

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}

	var userInfo *config.SubUserInfo
	if rawUI := resp.Header.Get("subscription-userinfo"); rawUI != "" {
		userInfo = parseUserinfo(rawUI)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var testMap map[string]any
	if err := yaml.Unmarshal(content, &testMap); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}

	tmpPath := targetPath + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return nil, fmt.Errorf("write temp file %q: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("replace file %q: %w", targetPath, err)
	}

	return userInfo, nil
}

func readYAMLMap(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}

	out := make(map[string]any)
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse YAML %q: %w", path, err)
	}
	return out, nil
}

func writeYAMLAtomically(targetPath string, data map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("create dir %q: %w", filepath.Dir(targetPath), err)
	}

	raw, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal YAML: %w", err)
	}

	tmpPath := targetPath + ".tmp"
	if err := os.WriteFile(tmpPath, raw, 0644); err != nil {
		return fmt.Errorf("write temp file %q: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace file %q: %w", targetPath, err)
	}
	return nil
}

// deepMergeMaps recursively merges src into dst. Nested maps are merged key-by-key;
// scalar values and slices in src overwrite those in dst.
func deepMergeMaps(dst, src map[string]any) {
	for k, srcVal := range src {
		srcMap, srcIsMap := srcVal.(map[string]any)
		if srcIsMap {
			if dstMap, dstIsMap := dst[k].(map[string]any); dstIsMap && dstMap != nil {
				deepMergeMaps(dstMap, srcMap)
				continue
			}
			copied := make(map[string]any, len(srcMap))
			deepMergeMaps(copied, srcMap)
			dst[k] = copied
			continue
		}
		dst[k] = srcVal
	}
}
