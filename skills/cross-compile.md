# Skill: Cross-Platform Build & Release

Panduan build multi-platform dan release otomatis untuk Water.

---

## Makefile

```makefile
.PHONY: help setup build build-all test test-integration run clean lint fmt release

VERSION ?= 0.1.0
BINARY  = water
LDFLAGS = -X main.Version=$(VERSION) -s -w

help:
	@grep -E '^[a-z-]+:' Makefile | sed 's/:.*//g' | column -t

setup:
	go mod download
	go mod tidy
	cd web && npm ci

build: web-build
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/water

build-all: web-build
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64  ./cmd/water
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64  ./cmd/water
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64   ./cmd/water
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64   ./cmd/water
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe ./cmd/water

web-build:
	cd web && npm run build

test:
	go test -v -race -cover ./...

test-integration:
	go test -v -tags=integration ./test/integration/...

run:
	go run ./cmd/water init --db-path .water-dev
	go run ./cmd/water serve --db-path .water-dev

clean:
	rm -rf bin/ dist/ .water-dev/ web/dist/

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	cd web && npm run format

release: clean build-all
	cd dist && for f in water-darwin-* water-linux-*; do tar -czf $$f.tar.gz $$f; done
	cd dist && zip water-windows-amd64.zip water-windows-amd64.exe
```

---

## GitHub Actions: Tests

```yaml
# .github/workflows/tests.yml
name: Tests

on:
  push:
    branches: [main, dev]
  pull_request:
    branches: [main]

jobs:
  test-backend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true
      - run: make setup
      - run: make test
      - run: make lint
      
  test-frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: npm
          cache-dependency-path: web/package-lock.json
      - run: cd web && npm ci
      - run: cd web && npm run lint
      - run: cd web && npm run check
```

---

## GitHub Actions: Build & Release

```yaml
# .github/workflows/build.yml
name: Build & Release

on:
  push:
    tags: ['v*']

jobs:
  build:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
            artifact: water-linux-amd64
          - os: ubuntu-latest
            goos: linux
            goarch: arm64
            artifact: water-linux-arm64
          - os: macos-latest
            goos: darwin
            goarch: arm64
            artifact: water-darwin-arm64
          - os: macos-13
            goos: darwin
            goarch: amd64
            artifact: water-darwin-amd64
          - os: windows-latest
            goos: windows
            goarch: amd64
            artifact: water-windows-amd64.exe

    runs-on: ${{ matrix.os }}
    
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
          GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} \
          go build \
            -ldflags "-X main.Version=$VERSION -s -w" \
            -o dist/${{ matrix.artifact }} \
            ./cmd/water
        shell: bash
        
      - name: Package (Unix)
        if: matrix.goos != 'windows'
        run: |
          cd dist
          tar -czf ${{ matrix.artifact }}.tar.gz ${{ matrix.artifact }}
          
      - name: Package (Windows)
        if: matrix.goos == 'windows'
        run: |
          cd dist
          Compress-Archive -Path ${{ matrix.artifact }} -DestinationPath water-windows-amd64.zip
        shell: pwsh
        
      - uses: softprops/action-gh-release@v2
        with:
          files: |
            dist/*.tar.gz
            dist/*.zip
```

---

## Embed Frontend ke Go Binary

```go
// cmd/water/main.go
package main

import "embed"

//go:embed all:web/dist
var webFS embed.FS

// Di server.go:
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
    // Strip leading slash
    path := strings.TrimPrefix(r.URL.Path, "/")
    if path == "" {
        path = "index.html"
    }
    
    content, err := webFS.ReadFile("web/dist/" + path)
    if err != nil {
        // Fallback ke index.html untuk SPA routing
        content, _ = webFS.ReadFile("web/dist/index.html")
        w.Header().Set("Content-Type", "text/html")
    } else {
        // Set content type berdasarkan extension
        ext := filepath.Ext(path)
        mimeTypes := map[string]string{
            ".js":  "application/javascript",
            ".css": "text/css",
            ".svg": "image/svg+xml",
        }
        if ct, ok := mimeTypes[ext]; ok {
            w.Header().Set("Content-Type", ct)
        }
    }
    
    w.Write(content)
}
```

---

## Version Injection

```go
// cmd/water/root.go
var Version = "dev" // overridden by -ldflags at build time

var rootCmd = &cobra.Command{
    Use:     "water",
    Version: Version,
}
```

Build dengan:
```bash
go build -ldflags "-X main.Version=0.1.0" -o bin/water ./cmd/water
water --version  # → water version 0.1.0
```

---

## Homebrew Formula

```ruby
# Formula/water.rb
class Water < Formula
  desc "Visual brain of MCP agents"
  homepage "https://github.com/water-viz/water"
  version "0.1.0"

  on_macos do
    on_arm do
      url "https://github.com/water-viz/water/releases/download/v#{version}/water-darwin-arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
    on_intel do
      url "https://github.com/water-viz/water/releases/download/v#{version}/water-darwin-amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/water-viz/water/releases/download/v#{version}/water-linux-arm64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
    on_intel do
      url "https://github.com/water-viz/water/releases/download/v#{version}/water-linux-amd64.tar.gz"
      sha256 "REPLACE_WITH_ACTUAL_SHA256"
    end
  end

  def install
    bin.install "water-#{OS.mac? ? 'darwin' : 'linux'}-#{Hardware::CPU.arm? ? 'arm64' : 'amd64'}" => "water"
  end

  service do
    run [opt_bin/"water", "serve"]
    keep_alive true
    log_path var/"log/water.log"
  end

  test do
    system "#{bin}/water", "--version"
  end
end
```

---

## Release Checklist

```bash
# 1. Update version di go.mod, package.json, CHANGELOG
# 2. Commit & push ke main
# 3. Tag
git tag v0.1.0
git push origin v0.1.0

# 4. GitHub Actions akan otomatis build + upload binaries

# 5. Update Homebrew formula (ganti sha256)
sha256sum dist/water-darwin-arm64.tar.gz
# Update Formula/water.rb dengan sha256 baru
git -C homebrew-water add . && git -C homebrew-water commit -m "Release v0.1.0"
```