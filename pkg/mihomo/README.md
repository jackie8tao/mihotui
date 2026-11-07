# Mihomo Go SDK (`pkg/mihomo`)

`pkg/mihomo` 是一个现代化、轻量级、类型安全的 Go 语言客户端 SDK，用于同 **Mihomo (Clash.Meta)** 的 External Controller RESTful API 与 WebSocket 流通信。

适合集成在各类管理工具、监控系统、定时测速自动化脚本及 CLI 应用中。

---

## 🌟 特性

- **零重型依赖**：不依赖庞大的 Mihomo 核心源码，仅基于标准库与 `gorilla/websocket`，体积极小。
- **现代化设计**：
  - 采用 **Functional Options Pattern（函数式选项模式）** 进行客户端初始化。
  - 严格遵守 `context.Context` 首参数设计。
  - 面向接口编程（暴露 `Client` 接口，便于外部做 Mock 测试）。
- **结构化错误匹配**：支持 `errors.Is(err, mihomo.ErrUnauthorized)` 及 `errors.Is(err, mihomo.ErrNotFound)`，附带详细的 `APIError` 状态码与远端返回信息。
- **功能覆盖完整**：
  - 内核版本 (`/version`)、运行时状态与内存 (`/memory`)
  - 动态切换模式（Rule / Global / Direct）与平滑配置热重载 (`/configs?force=true`)
  - 代理策略组与节点检索、节点切换 (`/proxies`)
  - 节点低延迟测试 (`/proxies/:name/delay`)
  - WebSocket 实时上下行带宽推送 (`/traffic`)
  - WebSocket 实时日志流监听 (`/logs`)

---

## 🚀 快速上手

### 1. 安装

```bash
go get mihomo-tui/pkg/mihomo
```

### 2. 初始化与使用

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"mihomo-tui/pkg/mihomo"
)

func main() {
	// 创建客户端实例
	client, err := mihomo.New("http://127.0.0.1:9090",
		mihomo.WithSecret("your-secret-token"),
		mihomo.WithTimeout(10*time.Second),
		mihomo.WithUserAgent("my-agent/1.0"),
	)
	if err != nil {
		log.Fatalf("Init client failed: %v", err)
	}

	ctx := context.Background()

	// 1. 获取版本
	version, err := client.GetVersion(ctx)
	if err != nil {
		log.Fatalf("Get version failed: %v", err)
	}
	fmt.Printf("Mihomo Version: %s (Meta: %v)\n", version.Version, version.Meta)

	// 2. 切换模式
	if err := client.UpdateMode(ctx, "rule"); err != nil {
		log.Printf("Update mode failed: %v", err)
	}

	// 3. 测速
	delay, err := client.TestDelay(ctx, "HK-Node-01", "http://www.gstatic.com/generate_204", 3000)
	if err != nil {
		log.Printf("Delay test failed: %v", err)
	} else {
		fmt.Printf("Delay: %s\n", mihomo.FormatDelay(delay))
	}
}
```

### 3. WebSocket 实时流量监听

```go
trafficChan := make(chan mihomo.Traffic, 20)
go func() {
	if err := client.SubscribeTraffic(ctx, trafficChan); err != nil {
		log.Printf("WebSocket traffic error: %v", err)
	}
}()

for traffic := range trafficChan {
	fmt.Printf("Up: %s | Down: %s\n", mihomo.FormatSpeed(traffic.Up), mihomo.FormatSpeed(traffic.Down))
}
```
