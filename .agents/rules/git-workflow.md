# Git Workflow & Commit Rules

## 1. Automatic Commit After Code Changes
- **Mandatory Auto-Commit**: Whenever code changes (features, bug fixes, refactorings, style cleanups) are completed and verified via tests, they **MUST be automatically committed immediately**.
- Do not wait for the user to explicitly say "commit code" or prompt for committing.
- Ensure the git working directory is kept clean after each completed instruction.

## 2. Conventional Commits Standards
- Commit messages must strictly follow the Conventional Commits specification:
  - Format: `<type>(<scope>): <description>`
  - Types: `feat`, `fix`, `refactor`, `style`, `test`, `docs`, `chore`.
  - Scope: optional but recommended, e.g. `installer`, `config`, `views`, `cli`.
- Commit messages must be written in standard English.
- Commits must be atomic: do not bundle unrelated changes together.
