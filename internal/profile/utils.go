package profile

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/jackie8tao/mihotui/internal/config"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// genSubFilePath generate subscription filepath from subscription name
func genSubFilePath(subName string) string {
	subName = strings.TrimSpace(subName)
	return fmt.Sprintf("%s.yaml", subName)
}

func replaceSymlink(target, linkPath string) error {
	tmpLink := linkPath + ".tmp"
	if err := os.Remove(tmpLink); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove stale temp symlink: %w", err)
	}

	if err := os.Symlink(target, tmpLink); err != nil {
		return fmt.Errorf("create temp symlink: %w", err)
	}
	if err := os.Rename(tmpLink, linkPath); err != nil {
		_ = os.Remove(tmpLink)
		return fmt.Errorf("replace symlink: %w", err)
	}
	return nil
}

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
		return nil, err
	}

	// Use Clash.Meta / mihomo UA so subscription providers return standard mihomo YAML
	req.Header.Set("User-Agent", "Clash.Meta; mihomo/1.18.0; mihotui")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with HTTP status %d", resp.StatusCode)
	}

	// Parse Userinfo if header present
	var userInfo *config.SubUserInfo
	if rawUI := resp.Header.Get("subscription-userinfo"); rawUI != "" {
		userInfo = parseUserinfo(rawUI)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Validate basic YAML structure to prevent saving 404 HTML pages
	var testMap map[string]any
	if err := yaml.Unmarshal(content, &testMap); err != nil {
		return nil, fmt.Errorf("downloaded content is not valid YAML: %w", err)
	}

	// Write atomically via temporary file
	tmpPath := targetPath + ".tmp"
	if err := os.WriteFile(tmpPath, content, 0644); err != nil {
		return nil, err
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, err
	}

	return userInfo, nil
}
