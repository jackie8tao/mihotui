# Architecture & Engineering Rules

## 1. Presentation Layer Separation (`internal/app/views`)

- Views must NEVER accept `*AppModel` directly to prevent circular dependencies and state leakage.
- Views must be pure, stateless functions accepting only read-only `Props` structs (e.g., `views.RenderOverview(p OverviewProps) string`).
- Presentation logic must be unit testable without mocking HTTP clients or Bubble Tea message loops.

## 2. Theme Isolation (`internal/app/theme`)

- `theme` contains design tokens and Lipgloss styles with zero internal dependencies.
- Both `app` and `views` may import `theme`. `theme` never imports other internal packages.

## 3. Configuration Discovery Standards (Self-Contained Suite)

- All suite runtime configuration and core profiles live inside the self-contained bundle root (`<appRoot>/etc/`):
  1. Explicit flag (`-c`): Strict read from the specified path. Fail fast with an error if missing or invalid.
  2. Default suite config: `<appRoot>/etc/mihotui/config.yaml` (via `filepath.Join(appPaths.AppEtcDir(), "config.yaml")`). Strict read; fail fast if missing or invalid.
- **Anti-Pattern Warning**: Never search current working directory (`./config.yaml`) or split state across user XDG directories (`~/.config/mihotui`) and system bundle directories (`<appRoot>/etc/mihomo`).

## 4. Managed Core & Suite Directories (`AppPaths`)

- `mihotui` operates as a self-contained suite manager, not a passive host-path probe.
- Do NOT heuristic-guess host paths (`/etc/mihomo`, `~/.config/mihomo`, `./`).
- Resolve runtime layout deterministically from executable location via `config.ResolveAppPaths()` (`<appRoot>/bin`, `<appRoot>/etc/mihotui`, `<appRoot>/etc/mihotui/subscriptions`, `<appRoot>/etc/mihomo`).
- Active Mihomo configuration is managed via atomic symlink replacement (`<appRoot>/etc/mihomo/config.yaml` -> `<appRoot>/etc/mihotui/subscriptions/<subName>.yaml`).

## 5. Code Hygiene & Comments

- **Document Implicit Assumptions (CRITICAL)**:
  Whenever code relies on implicit assumptions (such as pre-assumed directory hierarchies, filesystem layouts, runtime environments, external tool conventions, or protocol contracts), **clear and detailed explanatory comments MUST be documented directly alongside the relevant code**. Always document the exact assumed structure, mental model, or ASCII diagram so the rationale behind the logic is immediately transparent.
- **English Only**: All code comments, docstrings, git commit messages, and YAML configuration comments must be in standard English.
- **Domain Entity Ownership**: No generic `model` or `utils` packages. Persistent configuration schemas (`AppConfig`, `Subscription`, `SubUserInfo`) live in `internal/config`; local profile entities (`Profile`) and lifecycle management live in `internal/profile`.
- **Conventional Commits**: Format commit messages as `<type>(<scope>): <description>`.

## 6. Control Flow & Early Returns

- **Prefer Early Returns over `if-else` Nesting**:
  Handle guard conditions, errors, and special cases early and return immediately instead of wrapping main logic inside `if ... { ... } else { ... }` blocks.
- **Flatten Control Flow via Helper Functions**:
  When mutually exclusive branches contain substantial logic, extract them into dedicated single-purpose helper functions so callers stay flat, linear, and shallowly nested.
