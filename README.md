# Mihomo TUI

专为 Headless Linux（服务器 / 软路由 / VPS）打造的现代化轻量级 **mihomo (Clash.Meta)** 终端交互式管理工具。

通过 SSH 终端即可实现媲美桌面 GUI 的实时监控、节点测速、多配置文件与订阅管理，无需开放 Web 控制台公网端口，零运行依赖。

---

## 🌟 核心特性

- **1. 实时状态仪表盘 (Overview)**：
  - 实时上下行传输速率（B/s, KB/s, MB/s）与总吞吐量统计（通过 WebSocket 流式推送）。
  - 内核版本与 Meta 标识、混合端口、Socks5、Tun 模式及状态、内存占用监测。
  - 路由模式（Rule / Global / Direct）一键循环切换。
- **2. 策略组与节点管理 (Proxies)**：
  - 双栏交互视图：左侧策略组列表，右侧组内节点与延迟信息。
  - 实时延迟测速（`t` 键单节点测试，`T` 键并发批量测速），支持按延迟高低梯度着色。
  - 一键切换当前生效节点。
- **3. 配置与订阅管理 (Configs & Subs)**：
  - **多 Profile 扫描与平滑切换**：自动扫描配置目录下的全部 `.yaml` 配置，按 `Enter` 触发 `PUT /configs?force=true` 平滑热重载，无需重启服务。
  - **订阅导入与更新**：按 `a` 调出内置交互表单输入名称与订阅 URL，自动拉取并解析；按 `u` 更新单个订阅，按 `U` 批量更新全部订阅。
  - **流量与到期时间解析**：自动解析响应头中的 `subscription-userinfo`，呈现可视化流量使用进度条、已用/总量配额与到期倒计时。
  - **原子安全覆盖**：通过临时文件和 YAML 合法性校验后原子重命名，防止网络中断或非法 HTML 损坏本地配置。
- **4. 原生单二进制**：
  - 基于 Go + Bubble Tea (Charmbracelet)，编译为单一静态文件，通过 `scp/curl` 上传即用。
  - 原生支持 Linux x86_64、ARM64（树莓派/软路由等）。

---

## 🚀 快捷键指南

### 通用快捷键
| 按键 | 功能 |
| :--- | :--- |
| `Tab` | 循环切换 **Overview** / **Proxies** / **Configs & Subs** 视图 |
| `m` | 快速切换出站模式（`RULE` ➔ `GLOBAL` ➔ `DIRECT`） |
| `r` | 手动刷新全部 API 与配置文件数据 |
| `q` / `Ctrl+C` | 退出程序 |

### 节点管理视图 (Proxies)
| 按键 | 功能 |
| :--- | :--- |
| `← / h` | 将焦点移动到左侧 **策略组** 列表 |
| `→ / l` | 将焦点移动到右侧 **节点** 列表 |
| `↑ / k`, `↓ / j` | 上下移动光标 |
| `Enter` | 将当前光标所在的节点设为活动节点 |
| `t` | 对当前选中的单个节点测试延迟（Ping） |
| `T` | 对当前策略组内的所有节点进行**并发批量延迟测速** |

### 配置与订阅视图 (Configs & Subs)
| 按键 | 功能 |
| :--- | :--- |
| `↑ / k`, `↓ / j` | 上下移动选择配置文件/订阅项 |
| `Enter` | **应用并热重载**当前选中的配置文件到 Mihomo 核心 |
| `u` | 重新拉取并更新当前选中的订阅（自动更新流量配额） |
| `U` | 批量重新拉取并更新所有已添加的订阅 |
| `a` | 弹出表单新增订阅（输入订阅名称与 URL） |
| `d` | 删除当前选中的订阅及其本地文件 |
| `Esc` | 在添加订阅窗口中取消退出 |

---

## 📦 安装与运行

### 方式 1：Linux 在线一键安装 (推荐)

在 Linux 服务器终端直接运行一键安装脚本（自动检测当前系统 CPU 架构，拉取对应构建包并注册至 `/usr/local/bin`）：

```bash
curl -fsSL https://raw.githubusercontent.com/<your-username>/mihomo-tui/master/scripts/install.sh | bash
```

### 方式 2：下载 Release 手动安装

从 GitHub Releases 页面下载对应架构的 `.tar.gz`：

```bash
# 解压并进入目录
tar -zxvf mihomo-tui-v0.1.0-linux-amd64.tar.gz

# 执行安装
sudo ./install.sh
```

---

## 🛠️ 本地构建与打包

### 1. 本地一键打包

项目自带跨平台的自动化发布打包工具，支持在 Windows / macOS / Linux 任意环境下执行：

```bash
# 构建默认版本 (v0.1.0)
go run ./scripts/package.go

# 或指定版本标签
go run ./scripts/package.go -v v0.1.0
```

或者使用 Makefile / 脚本：
```bash
# Linux / macOS
make package
# 或
./scripts/build-linux.sh v0.1.0

# Windows PowerShell
.\scripts\build-linux.ps1 -Version v0.1.0
```

构建完成后，产物将存放在 `./dist/` 目录下：
- `mihomo-tui-v0.1.0-linux-amd64.tar.gz` (x86_64 服务器 / VPS)
- `mihomo-tui-v0.1.0-linux-arm64.tar.gz` (ARM64 软路由 / 树莓派)
- `mihomo-tui-v0.1.0-linux-armv7.tar.gz` (ARMv7 嵌入式路由器)
- `mihomo-tui-v0.1.0-linux-386.tar.gz` (32位 x86 系统)
- `sha256sums.txt` (全量哈希校验和)

每个发布包内已内置：
1. 编译优化后的二进制文件（自动剔除调试符号，体积仅约 3MB）
2. Linux 一键安装脚本 `install.sh`（自动设置 0755 可执行权限）
3. 配置模板目录 `configs/`（含 `config.example.yaml` 与 `subscriptions.example.yaml`）
4. Systemd 服务单元模板 `mihomo.service`
5. 说明文档 `README.md`

### 2. Linux 服务器上一键安装

解压发布的归档包后，直接执行内置的安装脚本即可：

```bash
tar -zxvf mihomo-tui-v0.1.0-linux-amd64.tar.gz
cd mihomo-tui-*
sudo ./install.sh
```

脚本将自动：
- 将二进制安装到 `/usr/local/bin/mihomo-tui`
- 赋予可执行权限
- 自动创建 `/etc/mihomo` 与默认用户配置目录

### 3. 运行参数

```bash
# 默认连接本地 http://127.0.0.1:9090，自动探测 /etc/mihomo 或 ~/.config/mihomo
mihomo-tui

# 指定外部控制器地址和密钥
mihomo-tui -u http://127.0.0.1:9090 -s "your_secret_here"

# 指定配置与订阅保存目录
mihomo-tui -d /etc/mihomo
```

也可通过环境变量覆盖：
```bash
export MIHOMO_API_URL="http://127.0.0.1:9090"
export MIHOMO_SECRET="your_secret"
export MIHOMO_CONFIG_DIR="/etc/mihomo"
mihomo-tui
```
