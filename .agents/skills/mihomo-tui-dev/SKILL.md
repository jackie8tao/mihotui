---
name: mihomo-tui-dev
description: >-
  Development guidelines, architectural patterns, and standards for MihomoTui.
  Use when designing or refactoring TUI components, managing Mihomo core directories,
  organizing Go packages, configuring distribution paths, or adding new views.
---

# MihomoTui Architecture & Development Guidelines

This skill encapsulates the core design principles, architecture standards, and development workflows established for the `mihomo-tui` project.

---

## 1. Directory Structure & Layered Architecture

The project strictly follows a layered, unidirectional architecture:

```
cmd/mihomo-tui/          # CLI Entrypoint (flag parsing, App bootstrap, tea.Program)
configs/                 # Default & sample configurations (English comments only)
internal/
├── app/                 # TUI Application State Machine & Coordination
│   ├── app.go          # Root Bubble Tea Model (Init, Update, View, event loops)
│   ├── keys.go         # Keybindings and Help mappings (KeyMap)
│   ├── theme/          # Design system & styles
│   │   └── theme.go    # Lipgloss colors (Color*) and styles (*Style) - zero internal deps
│   └── views/          # Pure presentation layer (Stateless rendering)
│       ├── types.go    # Props definitions (OverviewProps, ProxiesProps, ConfigsProps)
│       ├── overview.go # Dashboard view (RenderOverview)
│       ├── proxies.go  # Proxy groups & nodes view (RenderProxies)
│       ├── configs.go  # Profiles & subscriptions view (RenderConfigs)
│       └── views_test.go # Fast, pure presentation unit tests
├── config/              # Application configuration loading & environment overrides
└── profile/             # Profile & subscription lifecycle management
    ├── manager.go      # Profile & sub file I/O, hot-reloading orchestration
    └── types.go        # Domain entities (Profile, Subscription, SubUserInfo)
pkg/
└── mihomo/              # Public, standalone Go SDK for Mihomo External Controller
```

---

## 2. Presentation Layer Separation (`internal/app/views`)

### Rules
1. **Stateless Functions**: Views must NEVER receive `*AppModel` directly to prevent circular dependencies and state leakage.
2. **Props Pattern**: Every view must accept a read-only `Props` struct (e.g. `views.RenderOverview(p OverviewProps) string`).
3. **Pure Functions for Testability**: Presentation logic must be testable without mocking HTTP clients, WebSockets, or TEA message loops.
4. **Theme Isolation**: Both `internal/app` and `internal/app/views` import `internal/app/theme`. `theme` never imports other internal packages.

---

## 3. Configuration Discovery & Packaging Standards

Follow XDG Base Directory Specification and Linux FHS conventions:

| Precedence | Scope | Path Resolution |
| :--- | :--- | :--- |
| **1 (Highest)** | Explicit Flag (`-c`) | Strict read. Fail fast with clear error if not found or invalid YAML. |
| **2** | User-level Config | `$XDG_CONFIG_HOME/mihotui/config.yaml` (fallback `~/.config/mihotui/config.yaml` via `os.UserConfigDir()`). |
| **3** | System-wide Config | Linux: `/etc/mihotui/config.yaml` + `$XDG_CONFIG_DIRS`. Windows: `%ProgramData%\mihotui\config.yaml`. macOS: `/Library/Application Support/mihotui/config.yaml`. |
| **4 (Lowest)** | Built-in Defaults | `config.DefaultConfig()` |

> **Anti-Pattern Warning**: Never search the current working directory (`./config.yaml`) by default. Users run CLI tools from arbitrary terminal locations.

---

## 4. Managed Mihomo Core Directory (`mihomo_dir`)

1. **Managed Mode vs. Heuristic Guessing**:
   - `mihomo-tui` operates as a central manager. It does NOT probe random host paths (`/etc/mihomo`, `~/.config/mihomo`, `./`).
   - If `mihomo_dir` is specified in config/flags, use it directly and ensure directory existence (`os.MkdirAll(..., 0755)`).
   - If empty, default deterministically to the isolated application data directory (e.g. `$XDG_CONFIG_HOME/mihotui/mihomo`).
2. **Naming Convention**:
   - Use explicit, self-documenting naming: `resolveMihomoConfigDir` (not generic `resolveConfigDir`).
   - Config field: `mihomo_dir` (not ambiguous `config_dir`).

---

## 5. Go Package Design & Code Hygiene

1. **Eliminate Generic Packages**:
   - Do NOT create generic `model`, `utils`, or `common` packages.
   - Types belong to their domain owners (e.g., `Profile` and `Subscription` live in `internal/profile`).
   - Mihomo protocol models and byte/speed formatting helpers live directly in `pkg/mihomo`.
2. **Language & Comments**:
   - All code comments, docstrings, and configuration comments MUST be in standard English.
   - Public declarations must have descriptive Go doc comments following Effective Go style.
3. **Commit Standards**:
   - Follow Conventional Commits: `refactor(...)`, `feat(...)`, `fix(...)`, `docs(...)`.

