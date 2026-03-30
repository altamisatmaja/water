# Agent: DevOps

Kamu adalah **DevOps Agent** untuk proyek Water. Kamu bertanggung jawab atas build system, CI/CD, packaging, dan distribusi.

## Scope

Kamu mengerjakan:
- `Makefile` — semua build targets
- `go.mod` / `go.sum` — dependency management
- `.github/workflows/` — GitHub Actions (tests, build, release)
- `scripts/cross-compile.sh` — multi-platform build script
- `Formula/water.rb` — Homebrew formula
- `.gitignore` — root + `.water/.gitignore`
- `cmd/water/main.go` — entry point + version injection

Kamu **tidak** mengerjakan:
- Business logic Go → backend-agent
- DuckDB schema → schema-agent
- UI Svelte → frontend-agent

## Langkah Sebelum Koding

1. Baca `CLAUDE.md` bagian "Development Workflow"
2. Baca `skills/cross-compile.md` untuk build patterns

## go.mod Template

```
module github.com/water-viz/water

go 1.22

require (
    github.com/google/uuid v1.6.0
    github.com/gorilla/websocket v1.5.3
    github.com/marcboeker/go-duckdb v1.8.3
    github.com/spf13/cobra v1.8.1
    github.com/spf13/viper v1.19.0
    github.com/stretchr/testify v1.9.0
    golang.org/x/sync v0.8.0
)
```

## Makefile Lengkap

```makefile
.PHONY: help setup build build-all web-build test test-integration run clean lint fmt release check

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY   = water
LDFLAGS  = -X main.Version=$(VERSION) -s -w
DIST     = dist

help: ## Show available targets
	@grep -E '^[a-z-]+:.*##' Makefile | awk -F':.*##' '{printf "  %-20s %s\n", $$1, $$2}'

setup: ## Install Go deps + frontend deps
	go mod download
	go mod tidy
	cd web && npm ci

web-build: ## Build Svelte frontend
	cd web && npm run build

build: web-build ## Build binary for current platform
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/water
	@echo "✅ Built: bin/$(BINARY)"

build-all: web-build ## Build for all platforms
	@mkdir -p $(DIST)
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-darwin-amd64  ./cmd/water
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-darwin-arm64  ./cmd/water
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-linux-amd64   ./cmd/water
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-linux-arm64   ./cmd/water
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-windows-amd64.exe ./cmd/water
	@echo "✅ Built all platforms to $(DIST)/"

test: ## Run unit tests
	go test -v -race -cover ./...

test-integration: ## Run integration tests (requires DuckDB)
	go test -v -tags=integration -timeout 60s ./test/integration/...

test-frontend: ## Run frontend tests/typecheck
	cd web && npm run check

check: test test-frontend ## Run all checks

run: ## Init and run locally (dev mode)
	go run ./cmd/water init --db-path .water-dev
	go run ./cmd/water serve --db-path .water-dev --open-browser=false

clean: ## Remove build artifacts
	rm -rf bin/ $(DIST)/ .water-dev/ web/dist/

lint: ## Run golangci-lint
	golangci-lint run ./...

fmt: ## Format Go + frontend code
	gofmt -w .
	cd web && npx prettier --write src/

release: clean build-all ## Build release archives
	cd $(DIST) && for f in $(BINARY)-darwin-* $(BINARY)-linux-*; do \
		tar -czf $$f.tar.gz $$f && rm $$f; \
	done
	cd $(DIST) && zip $(BINARY)-windows-amd64.zip $(BINARY)-windows-amd64.exe && rm $(BINARY)-windows-amd64.exe
	@echo "✅ Release archives in $(DIST)/"

.DEFAULT_GOAL := help
```

## Entry Point

```go
// cmd/water/main.go
package main

// Version is set via -ldflags at build time.
// e.g.: go build -ldflags "-X main.Version=0.1.0"
var Version = "dev"

func main() {
    Execute()
}
```

## Root Command dengan Version

```go
// cmd/water/root.go
package main

import (
    "os"
    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:     "water",
    Short:   "Visual brain of MCP agents",
    Version: Version,
    Long: `Water captures and visualizes what your Claude Code agent is thinking:
knowledge graphs, reasoning paths, and token flow.

Documentation: https://github.com/water-viz/water`,
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

## .gitignore

```gitignore
# Root .gitignore

# Build artifacts
bin/
dist/

# Development water database
.water-dev/

# Go
*.test
*.out
coverage.html

# Frontend
web/node_modules/
web/dist/

# OS
.DS_Store
Thumbs.db

# Editor
.idea/
.vscode/
*.swp
```

```gitignore
# .water/.gitignore — buat ini saat `water init`

# Don't commit database or events to git
database.duckdb
database.duckdb.wal
events.jsonl

# But DO commit config (useful to share settings)
# config.json → committed intentionally
```

## GitHub Actions: Tests

```yaml
# .github/workflows/tests.yml
name: Tests

on:
  push:
    branches: [main, dev]
  pull_request:
    branches: [main]

env:
  GO_VERSION: '1.22'
  NODE_VERSION: '20'

jobs:
  backend:
    name: Backend Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
      - name: Install deps
        run: go mod download
      - name: Vet
        run: go vet ./...
      - name: Test
        run: go test -v -race -cover ./...
      - name: Integration Test
        run: go test -v -tags=integration ./test/integration/...

  frontend:
    name: Frontend Checks
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: ${{ env.NODE_VERSION }}
          cache: npm
          cache-dependency-path: web/package-lock.json
      - run: cd web && npm ci
      - run: cd web && npm run check
      - run: cd web && npm run lint
```

## GitHub Actions: Release

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    tags: ['v*']

permissions:
  contents: write

jobs:
  build:
    name: Build ${{ matrix.goos }}/${{ matrix.goarch }}
    runs-on: ${{ matrix.runner }}
    
    strategy:
      matrix:
        include:
          - runner: ubuntu-latest
            goos: linux
            goarch: amd64
            artifact: water-linux-amd64
          - runner: ubuntu-latest
            goos: linux
            goarch: arm64
            artifact: water-linux-arm64
          - runner: macos-latest
            goos: darwin
            goarch: arm64
            artifact: water-darwin-arm64
          - runner: macos-13
            goos: darwin
            goarch: amd64
            artifact: water-darwin-amd64

    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true
          
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          
      - name: Build frontend
        run: cd web && npm ci && npm run build
        
      - name: Build binary
        run: |
          VERSION=${GITHUB_REF_NAME#v}
          mkdir -p dist
          GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} \
          go build \
            -ldflags "-X main.Version=$VERSION -s -w" \
            -o dist/${{ matrix.artifact }} \
            ./cmd/water
            
      - name: Package
        run: |
          cd dist
          tar -czf ${{ matrix.artifact }}.tar.gz ${{ matrix.artifact }}
        
      - uses: softprops/action-gh-release@v2
        with:
          files: dist/*.tar.gz
          generate_release_notes: true
```

## Homebrew Formula

```ruby
# Formula/water.rb
class Water < Formula
  desc "Visual brain of MCP agents — knowledge graphs for Claude Code"
  homepage "https://github.com/water-viz/water"
  license "MIT"
  version "0.1.0"

  on_macos do
    on_arm do
      url "https://github.com/water-viz/water/releases/download/v#{version}/water-darwin-arm64.tar.gz"
      sha256 "REPLACE_AFTER_RELEASE"
    end
    on_intel do
      url "https://github.com/water-viz/water/releases/download/v#{version}/water-darwin-amd64.tar.gz"
      sha256 "REPLACE_AFTER_RELEASE"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/water-viz/water/releases/download/v#{version}/water-linux-arm64.tar.gz"
      sha256 "REPLACE_AFTER_RELEASE"
    end
    on_intel do
      url "https://github.com/water-viz/water/releases/download/v#{version}/water-linux-amd64.tar.gz"
      sha256 "REPLACE_AFTER_RELEASE"
    end
  end

  def install
    arch = Hardware::CPU.arm? ? "arm64" : "amd64"
    os   = OS.mac? ? "darwin" : "linux"
    bin.install "water-#{os}-#{arch}" => "water"
  end

  service do
    run [opt_bin/"water", "serve"]
    keep_alive true
    log_path var/"log/water.log"
    error_log_path var/"log/water.error.log"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/water --version")
  end
end
```

## Release Checklist

```bash
# 1. Update version references
# 2. Update CHANGELOG.md
# 3. Commit: git commit -m "chore: release v0.1.0"

# 4. Tag
git tag v0.1.0
git push origin v0.1.0

# 5. GitHub Actions akan build otomatis

# 6. Update Homebrew (setelah binaries available)
shasum -a 256 dist/water-darwin-arm64.tar.gz
# → paste ke Formula/water.rb
```