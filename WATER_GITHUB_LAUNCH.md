# Water: GitHub Repository Template & Launch Guide

---

## PART 1: README.md (For GitHub Repo)

```markdown
# Water 💧

**Visual brain of MCP agents** — understand knowledge retention, reasoning paths, and token flow.

[![GitHub License](https://img.shields.io/badge/license-MIT-blue)](./LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.22+-blue)](https://go.dev)
[![Status](https://img.shields.io/badge/status-Alpha-yellow)]()

## What is Water?

Water captures and visualizes what your Claude Code agent is thinking:

- **Knowledge Graphs**: See what your agent remembers (and what it forgets)
- **Reasoning Paths**: Trace decision-making chains across tool calls
- **Token Economics**: Understand cost and efficiency per knowledge chunk
- **Team Insights**: Share snapshots to debug and learn together

Perfect for developers building with [Claude Code](https://claude.ai) and [MCP](https://modelcontextprotocol.io).

## Quick Start

### Installation

#### macOS / Linux
```bash
brew tap water-viz/water https://github.com/water-viz/homebrew-water
brew install water
```

#### Windows (Scoop - Coming Soon)
```powershell
scoop bucket add water https://github.com/water-viz/scoop-water
scoop install water
```

#### From Source
```bash
git clone https://github.com/water-viz/water.git
cd water
make build
./bin/water --help
```

### Initialize a Project

```bash
cd your-claude-code-project
water init
```

This creates a `.water/` folder to store your agent's brain data.

### Start the Dashboard

```bash
water serve
# Opens http://localhost:3141 automatically
```

## Features

### 🧠 Knowledge Graphs
- Visual nodes representing knowledge chunks
- Edges showing semantic relationships
- Community detection (similar knowledge clusters together)
- Salience decay (see what the agent is forgetting)

### 📊 Metrics Dashboard
- Token usage per knowledge chunk
- Memory retention curves
- Tool effectiveness scores
- Reasoning depth tracking

### 🔍 Decision Trees
- Step-by-step reasoning path visualization
- Alternative paths not taken (counterfactuals)
- Confidence scores at each decision

### 📈 Analytics
- Time-series metrics (daily aggregates)
- Performance bottleneck detection
- Knowledge utility analysis

## How It Works

1. **Claude Code Agent** runs and calls MCP tools
2. **Water SDK Hook** (coming) intercepts API calls transparently
3. **Event Stream** flows to `.water/` folder
4. **DuckDB** stores nodes, edges, metrics
5. **Web Dashboard** visualizes in real-time (http://localhost:3141)

## Architecture

```
Claude Code Agent
        ↓
   Water Hook (SDK integration)
        ↓
   Event Capture (JSON Lines)
        ↓
   DuckDB Storage (.water/database.duckdb)
        ↓
   Graph Analysis (KNN, Louvain, Salience)
        ↓
   Web Dashboard (React + D3/Cytoscape)
```

## CLI Commands

```bash
water init              # Initialize .water/ folder
water serve             # Start web server + dashboard
water watch             # Tail live events (terminal UI)
water export [format]   # Export snapshot (json, csv, parquet)
water config [key]      # Get/set configuration
water install           # Install as background service
```

## Configuration

Water stores configuration in `.water/config.json`:

```json
{
  "db_path": ".water",
  "host": "127.0.0.1",
  "port": 3141,
  "embedding_mode": "local",
  "log_level": "info",
  "enable_websocket": true
}
```

### Environment Variables

```bash
ANTHROPIC_API_KEY        # For embeddings API (optional)
WATER_DB_PATH           # Override default .water directory
WATER_PORT              # Override default port 3141
WATER_LOG_LEVEL         # Debug, Info, Warn, Error
```

## Data Privacy

✅ **All data stays on your laptop** — Water is fully local-first.
- No cloud sync (by default)
- No telemetry
- No account required
- Export snapshots manually to share with teammates

Optional: Use `water export --anonymize` to redact sensitive data before sharing.

## Requirements

- **Go 1.22+** (for building from source)
- **macOS 10.15+**, **Linux** (any distro), or **Windows 10+**
- **2GB RAM** (minimum), **50GB disk** (for large projects)

## Roadmap

### Phase 1: MVP (March 31 - April 20, 2026)
- ✅ CLI (`water init`, `water serve`)
- ✅ DuckDB storage
- ✅ Basic web dashboard (static visualization)
- ✅ Homebrew distribution

### Phase 2: Intelligence (April 21 - May 11, 2026)
- ✅ Vector embeddings (local `all-minilm-l6-v2`)
- ✅ Semantic clustering (KNN, Louvain)
- ✅ Salience decay curves
- ✅ Advanced metrics & analytics

### Phase 3: Integration (May 12 - May 31, 2026)
- ✅ Official Anthropic SDK integration
- ✅ VSCode extension
- ✅ Team collaboration features
- ✅ Multi-agent visualization

### Phase 4: Scale (Q3 2026)
- ✅ PostgreSQL backend (for remote servers)
- ✅ Cloud export (optional)
- ✅ API for custom integrations
- ✅ Plugins ecosystem

## Contributing

We're in **Alpha** and actively building! Contributions are welcome.

See [CONTRIBUTING.md](./CONTRIBUTING.md) for guidelines.

### Quick Contributions
- 🐛 Bug reports (GitHub Issues)
- 💡 Feature requests (GitHub Discussions)
- 📖 Documentation improvements (PRs)
- 🧪 Tests & examples (PRs)

### What We Need
- Go developers (backend)
- React developers (frontend)
- DevOps (cross-platform builds, Homebrew, Docker)
- Documentation writers
- Beta testers (real projects!)

## Getting Help

- **GitHub Discussions**: Ask questions, share ideas
- **GitHub Issues**: Report bugs
- **Documentation**: [ARCHITECTURE.md](./ARCHITECTURE.md), [API.md](./API.md)
- **Discord** (coming soon): Real-time chat with maintainers

## License

MIT License — see [LICENSE](./LICENSE)

## Acknowledgments

Water was inspired by:
- [Langsmith](https://smith.langchain.com) (LLM observability)
- [Replicate](https://replicate.com) (model introspection)
- [Cursor Composer](https://cursor.sh) (IDE + agent integration)
- The MCP ecosystem (Claude, Anthropic team)

## Status

🟡 **Alpha** — Feature-complete MVP, ready for brave beta testers.

Not recommended for production yet. Breaking changes expected.

---

**Let's build the brain of MCP agents together.** 🧠

Questions? Open an issue or start a discussion!
```

---

## PART 2: CONTRIBUTING.md

```markdown
# Contributing to Water

Thanks for your interest in Water! We're excited to work with you.

## Code of Conduct

Be respectful, inclusive, and constructive. We're all here to learn and build something cool.

## Getting Started

### Prerequisites
- Go 1.22+
- Node.js 18+
- Git
- macOS, Linux, or Windows 10+ with WSL2

### Local Development

1. **Fork & clone**
   ```bash
   git clone https://github.com/YOUR_USERNAME/water.git
   cd water
   ```

2. **Install dependencies**
   ```bash
   make setup
   ```

3. **Run tests**
   ```bash
   make test
   ```

4. **Build**
   ```bash
   make build
   ```

5. **Try it out**
   ```bash
   ./bin/water init --db-path .water-dev
   ./bin/water serve --db-path .water-dev
   ```

### Project Structure

```
water/
├── cmd/water/          # CLI entry points (Cobra commands)
├── internal/
│   ├── capture/        # Event capture & streaming
│   ├── graph/          # DuckDB client & queries
│   ├── server/         # HTTP handlers
│   ├── metrics/        # Graph analysis (KNN, Louvain, embeddings)
│   ├── config/         # Configuration management
│   └── logger/         # Logging
├── web/                # React frontend
│   ├── src/
│   │   ├── components/ # React components
│   │   ├── pages/      # Page components
│   │   ├── hooks/      # Custom hooks
│   │   └── types/      # TypeScript types
│   └── package.json
├── test/               # Tests & fixtures
└── Makefile
```

## Development Workflow

### Branch Naming

```
feature/my-feature           # New feature
bugfix/my-bug               # Bug fix
docs/improve-readme         # Documentation
test/add-integration-tests  # Tests
chore/update-deps           # Maintenance
```

### Commit Messages

```
[type] Short description

Longer explanation if needed.

Fixes #123
```

Types: `feat`, `fix`, `docs`, `test`, `chore`, `refactor`

### Pull Request Process

1. Create a branch: `git checkout -b feature/my-feature`
2. Make changes
3. Write tests: `go test ./...`
4. Update docs if needed
5. Push: `git push origin feature/my-feature`
6. Open a PR on GitHub

**PR Guidelines**:
- Title: `[type] Short description` (e.g., `[feat] Add vector embeddings`)
- Description: Why? What changed? How to test?
- Link related issues: `Fixes #123`
- Request review from maintainers

### Code Style

**Go**:
```bash
make fmt        # Format code
make lint       # Check style
go test ./...   # Run tests
```

Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

**TypeScript/React**:
```bash
cd web
npm run lint
npm run format
npm run test
```

### Testing

**Go Tests**:
```bash
# All tests
make test

# Specific package
go test ./internal/graph

# With coverage
go test -cover ./...

# Integration tests
make test-integration
```

**React Tests**:
```bash
cd web
npm test
```

### Documentation

Update docs when:
- Adding a feature
- Changing behavior
- Adding CLI flags
- Modifying APIs

## Areas We Need Help

### Backend (Go)
- [ ] Embeddings service (ONNX Runtime integration)
- [ ] Louvain community detection algorithm
- [ ] KNN edge builder optimization
- [ ] WebSocket live streaming
- [ ] More tests (unit & integration)

### Frontend (React)
- [ ] Graph visualization (Cytoscape UI)
- [ ] Timeline component
- [ ] Metrics dashboard
- [ ] Decision tree view
- [ ] Search & filter UI

### DevOps
- [ ] Homebrew formula & GitHub Actions
- [ ] Docker image
- [ ] Windows binary & scoop formula
- [ ] Release automation

### Documentation
- [ ] Architecture guide (detailed)
- [ ] API documentation
- [ ] Tutorial: "Building your first agent with Water"
- [ ] Video walkthrough
- [ ] Blog posts

### Community
- [ ] Beta testing (real projects)
- [ ] Bug reports with examples
- [ ] Feature requests & discussions
- [ ] Ecosystem integrations (VSCode, etc.)

## Asking Questions

- **How do I...?** → Start a discussion in GitHub Discussions
- **I found a bug** → Open an issue with reproduction steps
- **I have an idea** → GitHub Discussions or Discussions tab
- **Real-time chat** → Join our Discord (link coming)

## Review Process

1. Maintainers review your PR
2. Feedback: address comments, push updates
3. Approval: ✅ Looks good!
4. Merge: Your code is live

We aim to review within 2-3 days.

## Recognition

We'll acknowledge contributors:
- In [CONTRIBUTORS.md](./CONTRIBUTORS.md)
- In GitHub releases
- In project README
- In Anthropic's Water documentation (for major contributions)

## Code of Conduct

Please be respectful:
- No harassment, discrimination, or hate speech
- Welcome diverse perspectives
- Assume good intent
- Address conflicts constructively

## Questions?

Open a GitHub issue or discussion. We're here to help!

---

**Happy coding!** 🎉

For maintainers: see [MAINTAINERS.md](./MAINTAINERS.md)
```

---

## PART 3: Repository Files

### `.github/workflows/tests.yml`

```yaml
name: Tests

on:
  push:
    branches: [main, dev]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: 1.22
      - run: make setup
      - run: make test
      - run: make lint

  test-frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: 18
      - run: cd web && npm ci
      - run: cd web && npm run lint
      - run: cd web && npm run test
```

### `.github/workflows/build.yml`

```yaml
name: Build & Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
            filename: water-linux-amd64
          - os: ubuntu-latest
            goos: linux
            goarch: arm64
            filename: water-linux-arm64
          - os: macos-latest
            goos: darwin
            goarch: arm64
            filename: water-darwin-arm64
          - os: macos-13
            goos: darwin
            goarch: amd64
            filename: water-darwin-amd64
          - os: windows-latest
            goos: windows
            goarch: amd64
            filename: water-windows-amd64.exe
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: 1.22
      - run: |
          GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} \
          go build -o dist/${{ matrix.filename }} ./cmd/water
      - uses: softprops/action-gh-release@v1
        with:
          files: dist/*
```

### `.github/ISSUE_TEMPLATE/bug_report.md`

```markdown
---
name: Bug Report
about: Report a bug
title: '[BUG] '
labels: bug
---

## Description
Brief description of the bug.

## Reproduction Steps
1. ...
2. ...
3. ...

## Expected Behavior
What should happen?

## Actual Behavior
What actually happens?

## Environment
- OS: [macOS, Linux, Windows]
- Go version: [output of `go version`]
- Water version: [output of `water --version`]
- Error message/logs: [paste here]

## Additional Context
Screenshots, logs, etc.
```

### `LICENSE`

```
MIT License

Copyright (c) 2026 Water Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

[Standard MIT license text...]
```

---

## PART 4: GitHub Repository Setup Checklist

### Before First Commit

- [ ] Create new GitHub repository: `github.com/water-viz/water`
- [ ] Set description: "Visual brain of MCP agents — knowledge graphs for Claude Code"
- [ ] Set homepage: "https://water-viz.github.io" (or link to docs)
- [ ] Add topics: `mcp`, `claude`, `llm`, `visualization`, `agent`, `debugging`
- [ ] Make repository **public**
- [ ] Initialize README.md, LICENSE, .gitignore

### Branch Protection

Settings → Branches → Add rule for `main`:
- [ ] Require a pull request before merging
- [ ] Require code reviews: 1
- [ ] Require branches to be up to date
- [ ] Require status checks to pass (tests, lint)
- [ ] Dismiss stale reviews

### GitHub Actions

- [ ] Create `.github/workflows/tests.yml` (above)
- [ ] Create `.github/workflows/build.yml` (above)
- [ ] Create `.github/workflows/pages.yml` (for docs)

### Project Visibility

- [ ] Create GitHub Project board for tracking issues
- [ ] Enable Discussions (for Q&A, ideas)
- [ ] Enable Wiki (optional, for detailed docs)
- [ ] Set up GitHub Pages (point to `/docs` or Jekyll)

### Homebrew Setup

1. Create new repo: `github.com/water-viz/homebrew-water`
2. Add formula: `Formula/water.rb`
3. Update with each release

```bash
# In main repo, on release:
./scripts/release-homebrew.sh v0.1.0
```

### Initial Release

1. Create a tag: `git tag v0.1.0`
2. Push tag: `git push origin v0.1.0`
3. GitHub Actions builds and releases binaries
4. Update Homebrew formula with new checksums

---

## PART 5: Day-1 Marketing

### Announce on:

- **GitHub**: First issue pinned (project status, roadmap)
- **Hacker News**: "Show HN: Water — Visualize MCP Agent Brains"
- **Dev.to**: "Introducing Water: Debug Your Claude Code Agent"
- **Twitter/X**: Screenshot GIF of graph UI + tagline
- **Claude/Anthropic Discussions**: Link & call for beta testers
- **Reddit**: r/OpenAI, r/MachineLearning (if relevant)

### Beta Tester Sign-Up

Create: `BETA_TESTERS.md`
- Form link (Google Form)
- What we're looking for (real projects, feedback)
- What you get (early access, credit in docs)

---

## Success Metrics (First 30 Days)

- ⭐ 100+ GitHub stars
- 📥 50+ brew installs
- 👥 10+ beta testers using on real projects
- 💬 5+ active issues / discussions
- 📺 1+ video walkthrough or tutorial

---

**Ready to launch?** 🚀 Let's ship it!
```

---

## Final Checklist Before Launch

```markdown
# Water Launch Checklist ✅

## Code
- [ ] All 3 docs complete (spec, impl guide, github setup)
- [ ] Go code stubs reviewed
- [ ] React scaffold ready
- [ ] Makefile works
- [ ] Tests pass locally

## GitHub
- [ ] Repository created & public
- [ ] README.md, LICENSE, CONTRIBUTING.md committed
- [ ] GitHub Actions workflows set up
- [ ] Branch protection configured
- [ ] Issues/Discussions enabled

## Documentation
- [ ] ARCHITECTURE.md complete
- [ ] API.md (REST endpoints) drafted
- [ ] ROADMAP.md published
- [ ] Quickstart tested end-to-end

## Homebrew
- [ ] homebrew-water repo created
- [ ] Formula template ready (signed pending)
- [ ] CI/CD for auto-release configured

## Marketing
- [ ] Press release drafted
- [ ] Tweet/announcement ready
- [ ] Beta tester form published
- [ ] Video walkthrough script ready

## Legal
- [ ] MIT License in repo ✅
- [ ] Copyright assigned (or clear)
- [ ] No license conflicts with deps

## Go Live
- [ ] First release tagged v0.1.0-alpha
- [ ] Binaries built and uploaded
- [ ] Homebrew formula published
- [ ] Announcement posted

Estimated time to launch: **2-3 weeks** (depending on team size)
```