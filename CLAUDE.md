# CLAUDE.md — Water Project

> Visual brain of MCP agents — knowledge graphs, reasoning paths, and token flow for Claude Code.

---

## Project Overview

**Water** is a lightweight, self-hosted visualization tool that captures and visualizes what Claude Code agents are "thinking". It runs locally, stores data in DuckDB, and exposes a web dashboard at `http://localhost:3141`.

**Module path**: `github.com/water-viz/water`  
**Go version**: 1.22+  
**Status**: Alpha (Week 1 of 8)

---

## Repository Structure

```
water/
├── CLAUDE.md                    ← You are here
├── Makefile
├── go.mod
├── go.sum
├── cmd/
│   └── water/
│       ├── main.go              ← Entry point
│       ├── root.go              ← Root Cobra command
│       ├── init.go              ← `water init`
│       ├── serve.go             ← `water serve`
│       ├── watch.go             ← `water watch`
│       ├── export.go            ← `water export`
│       ├── config.go            ← `water config`
│       └── install.go           ← `water install`
├── internal/
│   ├── capture/
│   │   ├── event.go             ← Canonical event schema
│   │   ├── writer.go            ← Write events to .jsonl
│   │   └── reader.go            ← Tail/read events.jsonl
│   ├── graph/
│   │   ├── client.go            ← DuckDB client wrapper
│   │   ├── schema.go            ← SQL schema + migrations
│   │   ├── nodes.go             ← CRUD for knowledge_nodes
│   │   ├── edges.go             ← CRUD for edges
│   │   └── traces.go            ← CRUD for reasoning_traces
│   ├── server/
│   │   ├── server.go            ← HTTP server + router
│   │   ├── handlers.go          ← REST API handlers
│   │   ├── websocket.go         ← WebSocket live updates
│   │   └── middleware.go        ← CORS, logging, recovery
│   ├── metrics/
│   │   ├── aggregator.go        ← Daily metrics aggregation
│   │   ├── knn.go               ← K-Nearest Neighbors
│   │   ├── louvain.go           ← Community detection
│   │   └── salience.go          ← Decay: exp(-t/tau)
│   ├── config/
│   │   └── config.go            ← Viper config management
│   └── logger/
│       └── logger.go            ← slog wrapper
├── pkg/
│   ├── duckdb/
│   │   └── pool.go              ← Connection pool
│   └── embedding/
│       ├── local.go             ← all-minilm-l6-v2 via ONNX
│       └── api.go               ← Anthropic embeddings API
├── web/                         ← Svelte frontend
│   ├── src/
│   │   ├── App.svelte
│   │   ├── components/
│   │   │   ├── Graph.svelte     ← Cytoscape.js knowledge graph
│   │   │   ├── Timeline.svelte  ← Reasoning trace timeline
│   │   │   ├── Metrics.svelte   ← Token + retention charts
│   │   │   └── Sidebar.svelte   ← Node detail panel
│   │   ├── stores/              ← Svelte writable stores
│   │   └── types/               ← TypeScript interfaces
│   ├── package.json
│   └── vite.config.ts
├── test/
│   ├── fixtures/
│   │   └── sample_events.jsonl
│   └── integration/
│       └── graph_test.go
├── scripts/
│   └── cross-compile.sh
├── .github/
│   ├── workflows/
│   │   ├── tests.yml
│   │   └── build.yml
│   └── ISSUE_TEMPLATE/
│       └── bug_report.md
├── agents/
│   ├── README.md
│   ├── orchestrator.md          ← Coordinates all sub-agents
│   ├── backend-agent.md         ← Go code generation
│   ├── frontend-agent.md        ← Svelte UI generation
│   ├── schema-agent.md          ← DuckDB schema & queries
│   └── devops-agent.md          ← CI/CD, Makefile, Homebrew
└── skills/
    ├── README.md
    ├── go-cobra-cli.md          ← Cobra CLI patterns
    ├── duckdb-go.md             ← DuckDB Go driver usage
    ├── svelte-cytoscape.md      ← Graph visualization
    ├── event-capture.md         ← Event schema & JSONL
    └── cross-compile.md         ← Multi-platform builds
```

---

## Core Concepts

### Event Flow
```
Claude Code Agent
  → Water Hook (SDK intercept)
  → events.jsonl (append-only log)
  → DuckDB (nodes, edges, traces, metrics)
  → WebSocket → Web Dashboard (live)
```

### Data Model
- **knowledge_nodes**: Chunks of information the agent learned (with embeddings)
- **edges**: Semantic/causal/retrieval relationships between nodes
- **reasoning_traces**: Ordered paths of decisions across tool calls
- **daily_metrics**: Aggregated token costs and retention rates

### Key Algorithms
- **KNN** (k=5 default): Connect semantically similar nodes via cosine similarity on embeddings
- **Louvain**: Cluster nodes into communities (color-coded in graph)
- **Salience Decay**: `weight = base_weight * exp(-Δt / tau)` — edges fade over time

---

## CLI Commands

| Command | Description |
|---------|-------------|
| `water init` | Create `.water/` folder with DuckDB schema + config |
| `water serve` | Start HTTP server, open `http://localhost:3141` |
| `water watch` | Live TUI tail of incoming events |
| `water export` | Dump graph to JSON/CSV/Parquet |
| `water config` | Get/set config values |
| `water install` | Register as macOS LaunchAgent or Linux systemd service |

---

## REST API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/nodes` | List all knowledge nodes |
| `GET` | `/api/nodes/:id` | Get single node + neighbors |
| `GET` | `/api/edges` | List all edges |
| `GET` | `/api/stats` | Aggregate metrics |
| `POST` | `/api/events` | Ingest a new event |
| `GET` | `/api/graph` | Full graph export (nodes + edges) |
| `GET` | `/api/traces` | List reasoning traces |
| `WS` | `/ws` | WebSocket live event stream |

---

## Configuration

`.water/config.json` (created by `water init`):

```json
{
  "db_path": ".water",
  "host": "127.0.0.1",
  "port": 3141,
  "embedding_mode": "local",
  "log_level": "info",
  "enable_websocket": true,
  "enable_analytics": false
}
```

Environment variable overrides:
```bash
ANTHROPIC_API_KEY   # For Anthropic embeddings (optional)
WATER_DB_PATH       # Override .water directory path
WATER_PORT          # Override port (default: 3141)
WATER_LOG_LEVEL     # debug | info | warn | error
```

---

## Development Workflow

```bash
# Setup
make setup          # go mod download + tidy

# Build
make build          # → bin/water
make build-all      # → dist/ (all platforms)

# Run locally
make run            # init .water-test + serve

# Test
make test           # go test -v -cover ./...
make test-integration

# Quality
make lint           # golangci-lint
make fmt            # gofmt -w .
```

---

## Go Conventions

- **Error handling**: Always return `error`. Never `panic` in library code.
- **Context**: Every DB query and HTTP handler receives `ctx context.Context`.
- **Logging**: Use `logger.Info/Error/Debug/Warn` (never `fmt.Println` in production paths).
- **Tests**: Use `testify/assert` and `testify/require`. Prefer table-driven tests.
- **Interfaces**: Define interfaces close to where they're consumed, not where implemented.
- **Naming**: Follow standard Go conventions (`camelCase` private, `PascalCase` exported).

### Package Rules
- `cmd/water/` — CLI wiring only. No business logic here.
- `internal/` — All business logic. Not importable externally.
- `pkg/` — Reusable utilities that could be extracted as libraries.

---

## Frontend Conventions (Svelte)

- **State**: Svelte stores (`writable`, `derived`) — no prop-drilling
- **Graph**: Cytoscape.js instance lives in `Graph.svelte`; expose events via `dispatch`
- **API calls**: Centralize in `src/api.ts` (Axios-based)
- **Types**: All TypeScript interfaces in `src/types/index.ts`
- **Styling**: Tailwind utility classes only — no custom CSS unless unavoidable

---

## Phase Checklist

### Week 1 (Now)
- [ ] `internal/config` — LoadConfig, Save, defaults
- [ ] `internal/logger` — slog wrapper
- [ ] `internal/capture/event.go` — Event struct + all sub-types
- [ ] `internal/graph/schema.go` — SQL CREATE TABLE statements
- [ ] `internal/graph/client.go` — NewClient, Close, Ping
- [ ] `cmd/water/init.go` — create `.water/` + schema + config
- [ ] Basic `make build` green

### Week 2
- [ ] `internal/graph/nodes.go` — InsertNode, GetNode, ListNodes
- [ ] `internal/graph/edges.go` — InsertEdge, GetEdges
- [ ] `internal/server/server.go` — HTTP router + CORS middleware
- [ ] `internal/server/handlers.go` — GET /api/nodes, POST /api/events
- [ ] `cmd/water/serve.go` — start server, open browser

### Week 3
- [ ] Svelte scaffold connected to backend
- [ ] Static Cytoscape graph rendering
- [ ] WebSocket live updates
- [ ] `make test` passing with integration tests

### Week 4–6
- [ ] Embeddings (local ONNX)
- [ ] KNN edge creation
- [ ] Louvain community detection
- [ ] Salience decay
- [ ] Metrics aggregation + charts

### Week 7–8
- [ ] Homebrew formula
- [ ] GitHub Actions cross-compile
- [ ] Public launch

---

## Common Pitfalls

1. **DuckDB concurrency**: DuckDB is single-writer. Use a mutex or connection pool (`pkg/duckdb/pool.go`).
2. **Embedding dimensions**: `all-minilm-l6-v2` = 384 dims. Anthropic API = 1536 dims. Don't mix.
3. **JSONL format**: Each event is a single JSON object per line — no trailing commas or arrays.
4. **Frontend build embedding**: Run `cd web && npm run build` before `make build` to embed assets via `//go:embed`.
5. **Port conflict**: Default 3141. If busy, use `--port` flag or `WATER_PORT` env.

---

## Key Dependencies

| Dependency | Version | Purpose |
|------------|---------|---------|
| `github.com/spf13/cobra` | v2 | CLI framework |
| `github.com/spf13/viper` | latest | Config management |
| `github.com/marcboeker/go-duckdb` | latest | Embedded DB |
| `github.com/gorilla/websocket` | latest | WebSocket server |
| `github.com/google/uuid` | latest | ID generation |
| `github.com/stretchr/testify` | latest | Test assertions |
| `golang.org/x/sync` | latest | errgroup, semaphore |
| `log/slog` | stdlib (Go 1.21+) | Structured logging |

---

## Resources

- [Technical Spec](./WATER_TECHNICAL_SPEC.md)
- [Go Implementation Guide](./WATER_GO_IMPLEMENTATION_GUIDE.md)
- [GitHub Launch Guide](./WATER_GITHUB_LAUNCH.md)
- [Cobra docs](https://cobra.dev)
- [DuckDB Go driver](https://github.com/marcboeker/go-duckdb)
- [Cytoscape.js](https://js.cytoscape.org)
- [MCP Spec](https://modelcontextprotocol.io)
- [Anthropic SDK Go](https://github.com/anthropics/anthropic-sdk-go)