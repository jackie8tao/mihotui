# Error Handling & Fail-Fast Rules

## 1. Zero Tolerance for Silent Fallbacks

- When an operation fails (system call, executable path resolution, file I/O, network request, parsing, symlink evaluation), **NEVER** swallow the error or return a hardcoded/arbitrary default fallback value.
- Any error must be wrapped and returned: `fmt.Errorf("...: %w", err)` to visibly surface root causes to the caller and user.
- Graceful fallbacks are forbidden unless there is an explicitly verified, well-documented business reason.

## 2. No Function Signature Compromise for `init()`

- Never omit an `error` return type from a function just to call it inside package `init()`.
- Functions that perform runtime discovery, I/O, or OS inspections must return `(T, error)`.
- Command flags must not perform dynamic environment detection in `init()`; defer resolution to `PreRunE` or `RunE`.

## 3. No Unverified Environment Assumptions

- Never silently assume or hardcode environment paths (such as `/usr/local/mihotui` or `/etc/mihomo`) when automatic environment detection fails.
- If environment detection fails, fail immediately with an explicit error explaining what could not be determined.

## 4. Concise & Uniform Error Formatting

- **Single Package-Boundary Prefix**: Apply a package prefix (e.g., `installer: ...`) only at exported package entrypoints, never inside unexported helpers. Callers (such as CLI `RunE` handlers) must not stack another redundant wrapper (e.g., `installation failed:`) on top of an already-prefixed package error.
- **Short Verb-First Context (`<action> [<target>]: %w`)**: Keep wrapped error strings lowercase, concise, and action-oriented (e.g., `create dir %q: %w`, `read gzip: %w`, `open archive %q: %w`). Avoid verbose filler such as `failed to ...`, `... failed`, or `unable to ...`.
- **No Bare System/I/O Returns**: Never return raw `os.*`, `io.*`, or `http.Client.Do` errors unwrapped; wrap every failure point with its short action context so the error chain reads as a clean, non-repetitive causal trace.
