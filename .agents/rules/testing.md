# Testing Rules

## 1. No Unsolicited Unit Tests

- **Explicit Request Required**: Do NOT create, generate, or add `*_test.go` unit test files unless the user explicitly asks for unit tests.
- When implementing features, bug fixes, or refactorings without an explicit request for tests, verify correctness by compiling the project (`go build ./...`) rather than writing new unit test files.
