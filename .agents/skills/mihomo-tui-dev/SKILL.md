---
name: mihomo-tui-dev
description: >-
  Development guidelines, architectural patterns, and standards for MihomoTui.
  Use when designing or refactoring TUI components, managing Mihomo core directories,
  organizing Go packages, configuring distribution paths, or adding new views.
---

# MihomoTui Architecture & Development Guidelines

This skill encapsulates the core design principles, architecture standards, and development workflows established for the `mihotui` project.

---

## 1. Directory Structure & Layered Architecture

The project strictly follows a layered, unidirectional architecture:

```text
cmd/mihotui/             # CLI Entrypoint (Cobra root, start, and install subcommands)
configs/                 # Default suite configuration template (English comments only)
internal/
├── app/                 # TUI Application Container & Page Routing
│   ├── app.go           # Root Bubble Tea Model (Page interface, tab routing, global keys)
│   ├── events/          # Cross-component & inter-page Bubble Tea event messages
│   │   └── events.go
│   ├── keymaps/         # Read-only global and page keybindings
│   │   └── keymaps.go
│   ├── pages/           # Stateful page sub-models (Overview, Proxies, Configs, Subscriptions)
│   │   ├── overview.go
│   │   ├── proxies.go
│   │   ├── configs.go
│   │   └── subscriptions.go
│   ├── theme/           # Design system & styles
│   │   └── theme.go     # Lipgloss colors (Color*) and styles (*Style) - zero internal deps
│   └── views/           # Pure presentation layer (Stateless rendering)
│       ├── types.go     # Props definitions (OverviewProps, ProxiesProps, ConfigsProps, SubscriptionsProps)
│       ├── overview.go  # Dashboard view (RenderOverview)
│       ├── proxies.go   # Proxy groups & nodes view (RenderProxies)
│       ├── configs.go   # Profiles view (RenderConfigs)
│       └── subscriptions.go # Subscriptions view (RenderSubscriptions)
├── config/              # Self-contained suite path resolution & AppConfig persistence
│   ├── config.go        # AppConfig, Subscription, SubUserInfo schemas & atomic YAML I/O
│   └── paths.go         # AppPaths & ResolveAppPaths via executable symlink evaluation
├── installer/           # Mihomo core binary & systemd service installer
│   ├── installer.go     # Directory initialization, remote/local .gz extraction
│   └── systemd.go       # Linux systemd unit serialization & D-Bus daemon-reload
└── profile/             # Profile & subscription lifecycle management
    ├── manager.go       # Profile listing, active profile symlink switching, subscription CRUD
    ├── types.go         # Local Profile domain entity
    └── utils.go         # Atomic subscription download, YAML validation, symlink replacement
pkg/
└── mihomo/              # Public, standalone Go SDK for Mihomo External Controller
```

---

## 2. Page & Presentation Layer Separation (`internal/app/pages` & `internal/app/views`)

### Rules

1. **Independent Page Sub-Models**: Each tab implements the `app.Page` interface in `internal/app/pages` and communicates across pages strictly via `internal/app/events`.
2. **Stateless Views**: Views in `internal/app/views` must NEVER receive `*AppModel` or page sub-models directly.
3. **Props Pattern**: Every view accepts a read-only `Props` struct (e.g. `views.RenderOverview(p OverviewProps) string`).
4. **Theme Isolation**: `internal/app`, `internal/app/pages`, and `internal/app/views` may import `internal/app/theme`. `theme` never imports other internal packages.

---

## 3. Self-Contained Suite Layout & Configuration Discovery

`mihotui` operates as a self-contained suite derived from the executable's real filesystem path (`filepath.EvalSymlinks` on `os.Executable()`):

```text
<appRoot>/                       # e.g. /usr/local/mihotui
├── bin/
│   ├── mihotui                  # Symlinked from /usr/local/bin/mihotui
│   └── mihomo                   # Installed by `mihotui install`
└── etc/
    ├── mihotui/
    │   ├── config.yaml          # Persistent AppConfig & subscriptions metadata
    │   └── subscriptions/       # Downloaded & local profile .yaml files
    └── mihomo/
        └── config.yaml          # Symlink -> ../mihotui/subscriptions/<active>.yaml
```

| Precedence | Scope | Path Resolution |
| :--- | :--- | :--- |
| **1 (Highest)** | Explicit Flag (`-c`) | Strict read from specified path. Fail fast with clear error if not found or invalid YAML. |
| **2** | Default Suite Config | `<appRoot>/etc/mihotui/config.yaml` (`filepath.Join(appPaths.AppEtcDir(), "config.yaml")`). Strict read; fail fast if missing or invalid. |

> **Anti-Pattern Warning**: Never search the current working directory (`./config.yaml`) or split state across user XDG directories (`~/.config/mihotui`) and host paths (`/etc/mihomo`).

---

## 4. Managed Mihomo Core & Profile Activation

1. **Deterministic Suite Paths**:
   - All runtime paths derive from `config.ResolveAppPaths()` (or `config.NewAppPaths(targetDir)` during `mihotui install -d`).
   - Do NOT probe random host paths (`/etc/mihomo`, `~/.config/mihomo`, `./`).
2. **Symlink-Driven Profile Switching**:
   - Subscription YAML files live in `<appRoot>/etc/mihotui/subscriptions/<subName>.yaml`.
   - `Manager.SetActiveProfile(subName)` atomically points `<appRoot>/etc/mihomo/config.yaml` to the target subscription YAML file and persists `active_subscription_name` in `<appRoot>/etc/mihotui/config.yaml`.
   - In `ConfigsModel`, hot-reloading via Mihomo API rolls back the active symlink if the reload fails.

---

## 5. Go Package Design & Code Hygiene

1. **Eliminate Generic Packages**:
   - Do NOT create generic `model`, `utils`, or `common` packages.
   - Persistent config schemas (`AppConfig`, `Subscription`, `SubUserInfo`) live in `internal/config`; local `Profile` lives in `internal/profile`.
   - Mihomo protocol models and byte/speed formatting helpers live directly in `pkg/mihomo`.
2. **Language & Comments**:
   - **Document Implicit Assumptions**: Whenever code relies on implicit assumptions (e.g. directory layouts, runtime environments, external conventions), clear explanatory comments and diagrams MUST be added directly in the code.
   - All code comments, docstrings, and configuration comments MUST be in standard English.
   - Public declarations must have descriptive Go doc comments following Effective Go style.
3. **Commit Standards**:
   - Follow Conventional Commits: `refactor(...)`, `feat(...)`, `fix(...)`, `docs(...)`.

---

## 6. Error Handling & Fail-Fast Principles

1. **No Silent Error Swallowing**:
   - When an operation fails (system call, executable path resolution, file I/O, network request, parsing), **NEVER** swallow the error or return a hardcoded default fallback.
   - Any failure must be explicitly surfaced with `fmt.Errorf("...: %w", err)`.
2. **Concise & Uniform Error Formatting**:
   - Apply a single package-boundary prefix (`config: ...`, `profile: ...`, `installer: ...`) only at exported package entrypoints, never inside unexported helpers.
   - Use short verb-first context (`<action> [<target>]: %w`) and avoid verbose filler (`failed to ...`).
3. **Never Compromise Signatures for `init()`**:
   - Do NOT omit `error` from a function return just to call it in `init()`.
   - Execute runtime inspection or I/O dynamically during command execution (`RunE`), not during package initialization.
