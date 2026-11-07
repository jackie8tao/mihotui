#!/usr/bin/env bash
# ========================================================
# Mihomo TUI Linux Package Builder
# ========================================================

set -e

VERSION=${1:-"v0.1.0"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "Building Linux packages with Go Packager..."
go run ./scripts/package.go -v "$VERSION"
