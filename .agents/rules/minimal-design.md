# Minimal Design & Schema Discipline Rules

## 1. No Unsolicited Fields (Structs, Configs, and Payloads)

- **Explicit Necessity Required**: Do NOT add fields to Go structs, YAML/JSON schemas, or generated config templates out of a desire for "completeness", guesswork, or speculative future use.
- **Every Field Must Have a Direct Consumer**: Before introducing any field, verify where it is read or written in the current task. Fields without an active, concrete consumer or explicit user requirement are forbidden.
- **Keep Generated Configs Minimal & Conflict-Free**: When generating default configurations for external tools (such as Mihomo `config.yaml`), emit only the strictly required fields for integration/startup. NEVER copy-paste bloated boilerplate presets or introduce conflicting keys (e.g., defining both `port: 7890` and `mixed-port: 7890` simultaneously).

## 2. Strict Separation of Data Models vs. Runtime State

- **Pure Persistent Schemas**: Configuration structs (e.g., `AppConfig`) must contain only persistent user configuration fields. Do NOT embed process-wide runtime state (such as loaded file paths with `yaml:"-"`) or derived internal paths inside persistent config models.
- **No Redundant Pass-Through Wrapper Structs**: Do NOT create new result/wrapper structs solely to re-expose or rename fields that the caller already possesses (e.g., duplicating `AppPaths` fields into an installer `Result` struct just for logging).

## 3. Respect Module Boundaries (No Silent Scope Creep)

- **Stay Within Requested Responsibilities**: Do NOT silently expand a module's responsibility by adding auxiliary setup steps, new directories, or cross-domain fields that were not asked for (e.g., a core binary installer taking over unrelated application config scaffolding and expanding shared path/result structs to plumb it through).
- **Ask Before Expanding**: If a task appears to require new configuration keys, new directory layouts, or additional struct fields beyond what was explicitly discussed, propose the minimal option first and confirm with the user rather than inventing fields unilaterally.
