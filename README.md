<div align="center">
  <img src="assets/water-icon.svg" alt="Water"  height="128" />

  **Visual brain of MCP agents** — understand knowledge retention, reasoning paths, and token flow.

  [![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)
  [![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
  [![Status](https://img.shields.io/badge/status-Alpha-yellow)]()
  [![GitHub Stars](https://img.shields.io/github/stars/water-viz/water?style=flat)](https://github.com/water-viz/water/stargazers)

  [Quick Start](#quick-start) · [Features](#features) · [Architecture](#architecture) · [CLI Reference](#cli-reference) · [Contributing](#contributing)
</div>

---

## What is Water?

Water is a **lightweight, self-hosted visualization tool** for Claude Code agents and MCP-based systems. It captures everything your agent does and turns it into an interactive knowledge graph you can explore in your browser.

No cloud. No account. No telemetry. Everything stays on your machine.

```
Claude Code Agent  →  Water Hook  →  DuckDB (.water/)  →  Dashboard (localhost:3141)
```

## Quick Start

### Install

**macOS / Linux (Homebrew)**
```bash
brew tap water-viz/water https://github.com/water-viz/homebrew-water
brew install water
```

**From source**
```bash
git clone https://github.com/water-viz/water.git
cd water
make build
./bin/water --help
```

**Windows (Scoop — coming soon)**
```powershell
scoop bucket add water https://github.com/water-viz/scoop-water
scoop install water
```

### Run

```bash
# 1. Initialize in your project
cd your-claude-code-project
water init

# 2. Start the dashboard
water serve
# → Opens http://localhost:3141 automatically
```

That's it. Water creates a `.water/` folder, starts a local HTTP server, and opens the dashboard. As your agent runs, the knowledge graph updates in real time.

---

## Features

### 🧠 Knowledge Graphs
See every piece of information your agent learned — as an interactive node graph.

- Nodes = knowledge chunks captured from MCP tool outputs
- Edges = semantic, causal, or retrieval relationships between nodes
- Node size scales with access frequency — busy nodes appear larger
- Edge opacity decays over time (`salience = exp(-Δt/τ)`) — you can literally watch the agent forget

### 📊 Token Economics
Understand the real cost and efficiency of every knowledge chunk:
- Token usage per node (input + output)
- Memory retention rate — what percentage of nodes are still "active"?
- Daily aggregates: token cost trends, retention curves, community growth

### 🔍 Reasoning Paths
Trace the agent's decision-making from query to answer:
- Step-by-step trace of tool calls and decisions
- Confidence scores at each decision point
- Alternative paths not taken (counterfactuals)
- Click any trace step to highlight the relevant graph region

### 🏘️ Community Detection
Nodes cluster automatically using the Louvain algorithm — similar knowledge groups together by color. Useful for spotting knowledge gaps and redundant information.

### 📡 Live Updates
The dashboard updates in real time via WebSocket as your agent runs. No refresh needed.

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Claude Code Agent                     │
└────────────────────────┬────────────────────────────────┘
                         │ SDK hook (intercepts API calls)
                         ▼
┌─────────────────────────────────────────────────────────┐
│              Event Capture  (.water/events.jsonl)       │
│   mcp_tool_call │ context_window │ decision │ error     │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│            DuckDB  (.water/database.duckdb)             │
│   knowledge_nodes │ edges │ reasoning_traces │ metrics  │
└────────────────────────┬────────────────────────────────┘
                         │
              ┌──────────┴──────────┐
              │  Graph Analysis     │
              │  KNN  │  Louvain    │
              │  Salience Decay     │
              └──────────┬──────────┘
                         │ HTTP + WebSocket
                         ▼
┌─────────────────────────────────────────────────────────┐
│         Web Dashboard  (localhost:3141)                 │
│   Cytoscape.js graph │ Timeline │ Metrics │ Traces      │
└─────────────────────────────────────────────────────────┘
```

**Tech stack:**
- **Backend**: Go 1.22, Cobra CLI, DuckDB (embedded), standard `net/http`
- **Frontend**: Svelte 4, Cytoscape.js, Tailwind CSS, Chart.js
- **Embeddings**: `all-minilm-l6-v2` via ONNX Runtime (local, no API key needed)
- **Distribution**: Single static binary — frontend embedded via `//go:embed`

---

## CLI Reference

```bash
water init              # Initialize .water/ in current project
water serve             # Start dashboard at http://localhost:3141
water watch             # Live event tail in terminal (TUI)
water export [format]   # Export snapshot: json | csv | parquet
water config [key]      # Get or set a config value
water install           # Register as background service (macOS / Linux)
```

**Common flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--db-path` | `.water` | Path to the `.water` directory |
| `--port` | `3141` | Web server port |
| `--host` | `127.0.0.1` | Bind address |
| `--open-browser` | `true` | Auto-open dashboard on `serve` |
| `--embedding-mode` | `local` | `local` (ONNX) or `api` (Anthropic) |

---

## Configuration

Water stores its config in `.water/config.json` (created by `water init`):

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

**Environment variable overrides:**

```bash
ANTHROPIC_API_KEY    # For Anthropic embeddings (optional)
WATER_DB_PATH        # Override .water directory path
WATER_PORT           # Override port
WATER_LOG_LEVEL      # debug | info | warn | error
```

---

## Data Privacy

Water is **fully local-first**:

- ✅ All data stays on your machine
- ✅ No cloud sync by default
- ✅ No telemetry or analytics
- ✅ No account required
- ✅ Open source (MIT)

Share snapshots manually with teammates using `water export --anonymize`.

---

## Project Structure

```
water/
├── cmd/water/          # CLI entry points (Cobra commands)
├── internal/
│   ├── capture/        # Event schema & JSONL streaming
│   ├── graph/          # DuckDB client, nodes, edges, traces
│   ├── server/         # HTTP handlers, WebSocket
│   ├── metrics/        # KNN, Louvain, salience decay
│   ├── config/         # Viper config management
│   └── logger/         # Structured logging (slog)
├── pkg/
│   └── embedding/      # Local ONNX + Anthropic API embeddings
├── web/                # Svelte frontend (embedded into binary)
│   └── src/
│       ├── components/ # Graph, Timeline, Metrics, Sidebar
│       ├── stores/     # Svelte state management
│       └── types/      # TypeScript interfaces
├── agents/             # Claude Code agent definitions
├── skills/             # Reusable coding patterns for agents
├── test/               # Integration tests & fixtures
├── CLAUDE.md           # Agent instructions & project conventions
└── Makefile
```

---

## Requirements

| | Minimum |
|---|---|
| OS | macOS 10.15+, Linux (any), Windows 10+ |
| RAM | 2 GB |
| Disk | 500 MB (for binary + database) |
| Go | 1.22+ *(build from source only)* |

No Node.js required to run Water — the frontend is pre-built and embedded in the binary.

---

## Roadmap

### ✅ Phase 1 — MVP (Week 1–3)
- `water init` + `water serve`
- DuckDB storage (nodes, edges, traces)
- Basic Cytoscape.js graph visualization
- Homebrew distribution

### 🔨 Phase 2 — Intelligence (Week 4–6)
- Local embeddings (`all-minilm-l6-v2` via ONNX)
- KNN edge creation (k=5, cosine similarity)
- Louvain community detection
- Salience decay curves
- Token efficiency metrics + Chart.js dashboard

### 🚀 Phase 3 — Integration (Week 7–8)
- Official Anthropic SDK hook (`anthropic-sdk-go`)
- VSCode extension
- Team collaboration (snapshot sharing)
- Multi-agent visualization

### 🌐 Phase 4 — Scale (Q3 2026)
- PostgreSQL backend option (for team servers)
- Optional cloud export
- Plugin ecosystem

---

## Contributing

Water is in **Alpha** and actively being built. All contributions are welcome.

### Getting started

```bash
# Fork & clone
git clone https://github.com/YOUR_USERNAME/water.git
cd water

# Install dependencies
make setup

# Run tests
make test

# Build
make build

# Try it
./bin/water init --db-path .water-dev
./bin/water serve --db-path .water-dev
```

### What we need

- **Go developers** — backend, graph algorithms, CLI
- **Svelte / TypeScript developers** — dashboard UI, Cytoscape.js
- **DevOps** — cross-platform builds, Docker, Homebrew
- **Beta testers** — run Water on a real project and share feedback

### Branch naming

```
feature/my-feature
bugfix/fix-description
docs/improve-readme
test/add-integration-tests
```

### Commit style

```
feat: add salience decay to edge rendering
fix: resolve DuckDB concurrent write panic
docs: update CLI reference in README
```

See [CONTRIBUTING.md](./CONTRIBUTING.md) for the full guide.

---

## Community

| Channel | Link |
|---------|------|
| 🐛 Bug reports | [GitHub Issues](https://github.com/water-viz/water/issues) |
| 💡 Feature requests | [GitHub Discussions](https://github.com/water-viz/water/discussions) |
| 📖 Documentation | [ARCHITECTURE.md](./ARCHITECTURE.md) · [API.md](./API.md) |
| 💬 Real-time chat | Discord *(coming soon)* |

---

## Acknowledgments

Water was inspired by:
- [LangSmith](https://smith.langchain.com) — LLM observability done right
- [Cursor Composer](https://cursor.sh) — IDE + agent tightly integrated
- The [MCP ecosystem](https://modelcontextprotocol.io) — for making agents composable

---

## License

MIT — see [LICENSE](./LICENSE)

---

<div align="center">
  <img src="assets/water-icon.svg" alt="Water" width="48" />
  <br/>
  <sub>Built for developers who want to understand what their agents are thinking.</sub>
  <br/><br/>
  <strong>Give it a ⭐ if Water helps you debug your agents!</strong>
</div>