# Water: MCP Brain Visualization Tool
## Complete Technical Specification v1.0

---

## 1. PROJECT OVERVIEW

**Name**: Water  
**Tagline**: Visual brain of MCP agents—understand knowledge retention, reasoning paths, and token flow.  
**Status**: Pre-Alpha (Design Phase)  
**License**: MIT (fully free, public GitHub)  

### Core Vision
Water is a lightweight, self-hosted visualization tool for Claude Code agents and MCP-based systems. It captures, analyzes, and visualizes:
- **Memory & Knowledge Graphs**: What does your agent remember? What has it forgotten?
- **Reasoning Paths**: Trace decision-making chains across tool calls and context windows.
- **Token Economics**: Understand cost and efficiency per knowledge chunk.
- **Team Insights**: Share `.water/` snapshots with teammates to debug and learn together.

### Why Now?
- MCP ecosystem is exploding (GitHub, Slack, Stripe integrations, etc.)
- No visual "brain introspection" tools for agents yet
- Developers urgently need to understand "what is my agent thinking?"

---

## 2. TECH STACK

### Backend
- **Language**: Go 1.22+
- **CLI Framework**: Cobra + Viper (flag parsing, config management)
- **Database**: DuckDB 1.0+ with pgvector extension (in-process, zero-config)
- **HTTP Server**: Go standard library (net/http) for MVP, upgrade to Axum (Rust) if needed
- **Embeddings**: `all-minilm-l6-v2` (ONNX Runtime) or Anthropic embeddings API
- **Graph Analysis**: Custom Go implementation (KNN, Louvain clustering, salience decay)

### Frontend
- **Framework**: Svelte 4 (TypeScript)
- **Graph Visualization**: Cytoscape.js
- **Styling**: Tailwind CSS + Svelte stores
- **Build**: Vite (lightning-fast HMR)
- **State Management**: Svelte Stores (writable, derived)
- **Charts**: Chart.js for metrics visualization

### Distribution
- **macOS**: Homebrew formula (cross-compile arm64 + amd64)
- **Linux**: Homebrew (Linuxbrew), static binary on GitHub Releases
- **Windows**: Binary executable, scoop formula (later)
- **Packaging**: GitHub Actions for cross-platform builds

### Dependencies (Minimal)
```
Go:
  - duckdb-go (database)
  - anthropic-sdk-go (future: official SDK)
  - cobra (CLI)
  - viper (config)
  - onnx-runtime-go (local embeddings)

Node (frontend only):
  - react, react-dom
  - cytoscape, cytoscape.js
  - shadcn/ui, tailwindcss
  - tanstack/react-query
```

---

## 3. CORE DATA MODEL

### Canonical Event Schema

All events captured from agents are normalized into this schema:

```json
{
  "id": "evt-abc123def456",
  "timestamp": "2026-03-30T14:22:31.456Z",
  "session_id": "sess-xyz789",
  "agent_id": "claude-code-instance-1",
  "event_type": "mcp_tool_call|context_window|memory_access|decision|error",
  
  "mcp_tool_call": {
    "server_name": "github",
    "tool_name": "search_repositories",
    "input": { "query": "golang orm", "limit": 10 },
    "output": { "results": [...], "count": 25 },
    "input_tokens": 245,
    "output_tokens": 1204,
    "execution_ms": 1850,
    "success": true,
    "error_message": null
  },
  
  "context_window": {
    "role": "user|assistant",
    "prompt_tokens": 8192,
    "completion_tokens": 512,
    "cached_tokens": 2048,
    "model": "claude-opus-4-6",
    "temperature": 0.7,
    "top_p": 1.0
  },
  
  "memory_access": {
    "chunk_id": "mem-456",
    "content_preview": "type User struct { ID string ... }",
    "access_type": "retrieve|update|create|delete",
    "importance_score": 0.87,
    "retention_confidence": 0.92,
    "age_seconds": 3600
  },
  
  "decision": {
    "node_id": "dec-789",
    "description": "Choose tool: github.search vs local_cache",
    "options": ["option_a", "option_b"],
    "chosen": "option_a",
    "reasoning": "API faster, fresher results",
    "confidence": 0.94
  },
  
  "error": {
    "error_type": "tool_execution_error|context_overflow",
    "message": "API rate limit exceeded",
    "stack_trace": null
  },
  
  "metadata": {
    "source": "sdk_hook|manual_log|inference",
    "environment": "development|production",
    "project_path": "/Users/me/project",
    "tags": ["debug", "test"]
  }
}
```

### Graph Database Schema (DuckDB + pgvector)

#### Table: `knowledge_nodes`
```sql
CREATE TABLE knowledge_nodes (
  node_id TEXT PRIMARY KEY,
  content TEXT NOT NULL,              -- Original text chunk
  content_hash TEXT,                  -- For dedup
  embedding FLOAT32[],                -- pgvector (384-768 dims)
  source_type TEXT,                   -- 'mcp_output' | 'context' | 'memory'
  source_tool TEXT,                   -- 'github' | 'slack' | null
  tokens_in BIGINT,                   -- Input tokens from source
  tokens_out BIGINT,                  -- Output tokens containing this node
  created_at TIMESTAMP,
  first_accessed_at TIMESTAMP,
  last_accessed_at TIMESTAMP,
  access_count BIGINT DEFAULT 0,
  importance_score FLOAT,             -- Manual or computed
  retention_confidence FLOAT,         -- How likely to be remembered
  tags TEXT[]                         -- User-defined labels
);
CREATE INDEX idx_embedding ON knowledge_nodes USING hnsw (embedding);
```

#### Table: `edges`
```sql
CREATE TABLE edges (
  edge_id TEXT PRIMARY KEY,
  from_node_id TEXT REFERENCES knowledge_nodes(node_id),
  to_node_id TEXT REFERENCES knowledge_nodes(node_id),
  edge_type TEXT,                     -- 'semantic' | 'causal' | 'retrieval'
  weight FLOAT,                       -- 0.0 - 1.0 (KNN similarity, salience, etc.)
  salience FLOAT,                     -- Decay over time: exp(-t/tau)
  reasoning_path TEXT,                -- "why connected?" for debugging
  community_id INT,                   -- Louvain clustering result
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
CREATE INDEX idx_from ON edges(from_node_id);
CREATE INDEX idx_to ON edges(to_node_id);
```

#### Table: `reasoning_traces`
```sql
CREATE TABLE reasoning_traces (
  trace_id TEXT PRIMARY KEY,
  session_id TEXT,
  timestamp TIMESTAMP,
  nodes_path TEXT[],                  -- [node1, node2, node3, ...] ordered
  edge_path TEXT[],                   -- Connecting edges
  depth INT,                          -- Number of hops
  decision_points JSONB,              -- Decisions made at each step
  total_tokens INT,
  latency_ms BIGINT,
  tool_calls TEXT[],                  -- Tools invoked along the path
  outcome TEXT                        -- 'success' | 'partial' | 'failed'
);
```

#### Table: `daily_metrics` (aggregates)
```sql
CREATE TABLE daily_metrics (
  date DATE,
  node_id TEXT,
  access_count BIGINT,
  avg_latency_ms FLOAT,
  token_cost INT,
  retention_rate FLOAT,               -- % still "remembered" today
  importance_trend FLOAT,             -- Delta from yesterday
  community_id INT,
  PRIMARY KEY (date, node_id)
);
```

---

## 4. CLI STRUCTURE (Cobra)

### Commands

#### `water init`
```bash
$ water -init

Initialize Water in current project.

Flags:
  --api-key STRING           Anthropic API key (optional, for embeddings)
  --embedding-mode local|api (default: local)
  --db-path PATH             Custom path for .water/ (default: ./.water)
  --port INT                 Web server port (default: 3141)

Output:
  Creates:
    .water/
    ├── database.duckdb
    ├── config.json
    ├── events.jsonl
    └── .gitignore
```

#### `water serve`
```bash
$ water serve

Start the HTTP server and open the web dashboard.

Flags:
  --db-path PATH             Path to .water (auto-detect)
  --port INT                 (default: 3141)
  --host STRING              (default: 127.0.0.1)
  --open-browser             Auto-open browser (default: true)
  --watch                    Watch .water/events.jsonl for live updates
```

#### `water watch`
```bash
$ water watch

Tail live events as they arrive. TUI-based live graph visualization.

Flags:
  --db-path PATH
  --refresh INT              Update interval in ms (default: 1000)
  --format json|text|graph   (default: text)
```

#### `water export`
```bash
$ water export

Export graph snapshot for sharing or analysis.

Flags:
  --format json|csv|parquet  (default: json)
  --include-vectors          Include embeddings in export
  --date-range START,END     ISO 8601 dates
  --output PATH              Write to file (default: stdout)
  --anonymize                Redact sensitive content
```

#### `water install`
```bash
$ water install

Install Water as a background service (macOS: LaunchAgent, Linux: systemd).

Flags:
  --daemon-name STRING       Service name (default: com.water.agent)
  --auto-start               Enable at boot
```

#### `water config`
```bash
$ water config [key] [value]

Get/set configuration.

Examples:
  water config show
  water config embedding-mode api
  water config anthropic-api-key sk-...
```

---

## 5. GOLANG IMPLEMENTATION ROADMAP

### Phase 1: Foundation (Weeks 1-3)

#### Milestone 1a: CLI scaffold
- [x] Initialize Cobra CLI with basic commands
- [x] Create Go module structure (cmd/, internal/)
- [x] Config loading (viper): .water/config.json
- [x] DuckDB connection + schema creation

#### Milestone 1b: Event capture
- [x] Define `Event` struct matching canonical schema
- [x] JSON marshaling/unmarshaling
- [x] Channel-based event streaming (producer-consumer)
- [x] Append-only event log (events.jsonl in .water/)

#### Milestone 1c: Basic web server
- [x] HTTP endpoints: `GET /api/nodes`, `GET /api/edges`, `GET /api/stats`
- [x] CORS headers for local development
- [x] Static file serving for React build

#### Milestone 1d: Graph storage
- [x] DuckDB client wrapper (pkg/graph/)
- [x] Node/edge insertion logic
- [x] Simple queries: "find node by ID", "get neighbors"

**Deliverable**: `water init` + `water serve` with static graph visualization.

---

### Phase 2: Intelligence (Weeks 4-6)

#### Milestone 2a: Vector embeddings
- [x] Integrate `all-minilm-l6-v2` via ONNX Runtime
- [x] Embed each node's content on insert
- [x] Store vectors in DuckDB (pgvector extension)
- [x] OR fallback to Anthropic embeddings API (optional, flagged)

#### Milestone 2b: Graph analysis
- [x] KNN edge builder (cosine similarity)
- [x] Salience decay calculation (exp(-t/tau))
- [x] Community detection (Louvain algorithm)
  - Use existing Go lib: `github.com/james-bowman/nlp` or implement from scratch
- [x] Daily aggregation: compute `daily_metrics` table

#### Milestone 2c: Query service
- [x] GraphQL endpoint (optional, start with REST)
- [x] Advanced queries: "find most important knowledge", "memory retention curve"
- [x] WebSocket endpoint for live updates

**Deliverable**: Graph with semantic clustering, decay visualization, metrics dashboard.

---

### Phase 3: Polish & Distribution (Weeks 7-8)

#### Milestone 3a: Brew packaging
- [x] Create `water.rb` formula for Homebrew
- [x] GitHub Actions workflow for cross-compilation (darwin-amd64, darwin-arm64, linux-amd64, linux-arm64, windows-amd64)
- [x] Codesigning (macOS) — optional but recommended
- [x] Release artifacts on GitHub Releases

#### Milestone 3b: Documentation
- [x] README.md with quickstart
- [x] Architecture guide
- [x] API docs (Swagger/OpenAPI)
- [x] Video walkthrough (5 min)

#### Milestone 3c: VSCode extension
- [x] Minimal sidebar showing current session graph
- [x] Link to web dashboard

**Deliverable**: Shipped on Homebrew, public GitHub, ready for beta testing.

---

### Phase 4: Official Integration (Weeks 9+)

- Work with Anthropic to integrate Water hooks into Claude Code SDK
- Official documentation on anthropic.com
- Standardized event protocol (RFC)

---

## 6. GOLANG PROJECT STRUCTURE

```
water/
├── .github/workflows/
│   ├── build.yml            # Cross-platform builds
│   ├── tests.yml            # Go tests + coverage
│   └── release.yml          # Auto-release on tag
│
├── cmd/water/
│   └── main.go              # Cobra CLI entry
│
├── internal/
│   ├── capture/
│   │   ├── event.go         # Event struct, marshaling
│   │   ├── stream.go        # Channel-based streaming
│   │   └── decoder.go       # JSON → Event
│   │
│   ├── graph/
│   │   ├── client.go        # DuckDB connection
│   │   ├── node.go          # Node CRUD
│   │   ├── edge.go          # Edge CRUD
│   │   ├── schema.go        # SQL schema initialization
│   │   └── query.go         # Advanced queries
│   │
│   ├── server/
│   │   ├── http.go          # HTTP handler setup
│   │   ├── handlers.go      # Endpoint implementations
│   │   ├── websocket.go     # WebSocket for live updates
│   │   └── middleware.go    # CORS, logging
│   │
│   ├── metrics/
│   │   ├── embedding.go     # Vector operations
│   │   ├── salience.go      # Decay calculations
│   │   ├── louvain.go       # Community detection
│   │   ├── knn.go           # KNN edge builder
│   │   └── daily.go         # Aggregation job
│   │
│   ├── config/
│   │   └── config.go        # Viper config management
│   │
│   └── logger/
│       └── logger.go        # Structured logging (slog)
│
├── pkg/
│   ├── duckdb/              # Reusable DuckDB utilities
│   ├── embedding/           # Reusable embedding logic
│   └── math/                # Linear algebra, similarity
│
├── web/                     # React frontend (separate package.json)
│   ├── src/
│   │   ├── components/
│   │   │   ├── Graph.tsx
│   │   │   ├── Timeline.tsx
│   │   │   ├── Metrics.tsx
│   │   │   └── ...
│   │   ├── pages/
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── vite.config.ts
│   └── package.json
│
├── scripts/
│   ├── embed_web.sh         # Embed React build into Go binary
│   ├── setup-db.sh          # Initialize DuckDB + schema
│   └── cross-compile.sh     # Multi-platform builds
│
├── test/
│   ├── fixtures/            # Sample events, snapshots
│   └── integration/         # End-to-end tests
│
├── Formula/
│   ├── water.rb             # Homebrew formula (darwin)
│   └── water-linux.rb       # Homebrew formula (linux)
│
├── Dockerfile               # Optional: containerized deployment
├── go.mod
├── go.sum
├── Makefile                 # Build targets
├── README.md
├── LICENSE
└── ARCHITECTURE.md
```

---

## 7. GOLANG KEY MODULES (Pseudocode)

### `internal/capture/event.go`

```go
package capture

import "time"

// Event is the canonical event type
type Event struct {
    ID            string                 `json:"id"`
    Timestamp     time.Time              `json:"timestamp"`
    SessionID     string                 `json:"session_id"`
    AgentID       string                 `json:"agent_id"`
    EventType     string                 `json:"event_type"` // mcp_tool_call, etc.
    
    MCPToolCall   *MCPToolCallEvent      `json:"mcp_tool_call,omitempty"`
    ContextWindow *ContextWindowEvent    `json:"context_window,omitempty"`
    MemoryAccess  *MemoryAccessEvent     `json:"memory_access,omitempty"`
    Decision      *DecisionEvent         `json:"decision,omitempty"`
    Error         *ErrorEvent            `json:"error,omitempty"`
    
    Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type MCPToolCallEvent struct {
    ServerName    string                 `json:"server_name"`
    ToolName      string                 `json:"tool_name"`
    Input         map[string]interface{} `json:"input"`
    Output        map[string]interface{} `json:"output"`
    InputTokens   int64                  `json:"input_tokens"`
    OutputTokens  int64                  `json:"output_tokens"`
    ExecutionMs   int64                  `json:"execution_ms"`
    Success       bool                   `json:"success"`
    ErrorMessage  *string                `json:"error_message,omitempty"`
}

// ... other event types
```

### `internal/graph/client.go`

```go
package graph

import (
    "database/sql"
    _ "github.com/marcboeker/go-duckdb"
)

type Client struct {
    db *sql.DB
}

func NewClient(dbPath string) (*Client, error) {
    // Open DuckDB connection
    // Run migrations
    return &c, nil
}

func (c *Client) InsertNode(node *Node) error {
    // INSERT INTO knowledge_nodes ...
}

func (c *Client) GetNode(nodeID string) (*Node, error) {
    // SELECT * FROM knowledge_nodes WHERE node_id = ?
}

func (c *Client) InsertEdge(edge *Edge) error {
    // INSERT INTO edges ...
}

func (c *Client) GetNeighbors(nodeID string, depth int) ([]Node, []Edge, error) {
    // Graph traversal
}
```

### `internal/metrics/embedding.go`

```go
package metrics

type EmbeddingService interface {
    Embed(text string) ([]float32, error)
}

type LocalEmbedder struct {
    // ONNX Runtime session
}

func NewLocalEmbedder() (*LocalEmbedder, error) {
    // Load all-minilm-l6-v2.onnx
    // Initialize session
}

func (e *LocalEmbedder) Embed(text string) ([]float32, error) {
    // Tokenize, run inference, return vector
}

type AnthropicEmbedder struct {
    client *anthropic.Client
}

func (e *AnthropicEmbedder) Embed(text string) ([]float32, error) {
    // Call Anthropic embeddings API
}
```

### `internal/metrics/louvain.go`

```go
package metrics

import "gonum/graph/community"

func (g *GraphStore) DetectCommunities() (map[string]int, error) {
    // Build gonum graph from DuckDB nodes + edges
    // Run Louvain algorithm (gonum/graph/community/louvain)
    // Return node_id → community_id mapping
    // Store in edges table
}
```

### `internal/server/handlers.go`

```go
package server

import "net/http"

func (s *Server) HandleGetNodes(w http.ResponseWriter, r *http.Request) {
    // GET /api/nodes?session_id=...
    // Return JSON array of nodes
}

func (s *Server) HandleGetEdges(w http.ResponseWriter, r *http.Request) {
    // GET /api/edges?from_node=...
    // Return JSON array of edges
}

func (s *Server) HandleGetStats(w http.ResponseWriter, r *http.Request) {
    // GET /api/stats?date=...
    // Return metrics summary
}

func (s *Server) HandleExport(w http.ResponseWriter, r *http.Request) {
    // POST /api/export?format=json&anonymize=true
    // Return snapshot JSON
}
```

---

## 8. FRONTEND ARCHITECTURE

### React Components

```
web/src/
├── components/
│   ├── Graph.tsx              # Cytoscape wrapper
│   ├── Timeline.tsx           # Reasoning path timeline
│   ├── Metrics.tsx            # Stats cards
│   ├── TokenHeatmap.tsx       # Token cost visualization
│   ├── DecisionTree.tsx       # Agent decision branching
│   ├── MetadataPanel.tsx      # Node/edge details
│   └── Controls.tsx           # Filters, date range
│
├── pages/
│   ├── Dashboard.tsx          # Main view (graph + timeline + metrics)
│   ├── Analysis.tsx           # Deep analysis (communities, retention curves)
│   └── Settings.tsx           # Config, export
│
├── hooks/
│   ├── useGraphData.ts        # TanStack Query for /api/nodes, /api/edges
│   └── useWebSocket.ts        # Live updates
│
└── types/
    └── graph.ts               # TypeScript interfaces matching Go structs
```

### Key Features

**Graph Visualization**:
- Nodes = knowledge chunks
- Edges = semantic relationships (colored by type)
- Node size = access frequency
- Node color = community
- Edge opacity = salience (fades with time)
- Hover = show content preview

**Timeline View**:
- Vertical timeline of reasoning steps
- Each step shows: decision, tools used, tokens, outcome
- Click = zoom into graph at that moment

**Metrics Dashboard**:
- Token efficiency (tokens in vs. useful output)
- Memory retention rate (% nodes still "active")
- Tool effectiveness (success rate, latency)
- Community insights (cluster sizes, intra-cluster vs. inter-cluster edges)

**Reasoning Path**:
- Highlight the chain of decisions leading to a result
- Show alternative paths not taken (counterfactuals)

---

## 9. DISTRIBUTION STRATEGY

### Homebrew (Primary)

#### `Formula/water.rb` (macOS)
```ruby
class Water < Formula
  desc "Visual brain of MCP agents"
  homepage "https://github.com/water-viz/water"
  url "https://github.com/water-viz/water/releases/download/v0.1.0/water-0.1.0-darwin-amd64.tar.gz"
  sha256 "..."
  
  def install
    bin.install "water"
  end
  
  service do
    run [opt_bin/"water", "serve"]
    keep_alive true
    log_path var/"log/water.log"
  end
end
```

#### Installation
```bash
brew tap water-viz/water https://github.com/water-viz/homebrew-water
brew install water
water -init
water serve
```

### GitHub Releases

Cross-platform binaries:
- `water-0.1.0-darwin-amd64.tar.gz`
- `water-0.1.0-darwin-arm64.tar.gz` (Apple Silicon)
- `water-0.1.0-linux-amd64.tar.gz`
- `water-0.1.0-linux-arm64.tar.gz`
- `water-0.1.0-windows-amd64.zip`

### CI/CD (GitHub Actions)

```yaml
# .github/workflows/release.yml
on:
  push:
    tags: ['v*']

jobs:
  build-and-release:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, macos-13]  # Intel + ARM
        include:
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
          - os: macos-latest
            goos: darwin
            goarch: arm64
          - os: macos-13
            goos: darwin
            goarch: amd64
    
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: make release
      - uses: softprops/action-gh-release@v1
        with:
          files: dist/*
```

---

## 10. ANTHROPIC INTEGRATION PLAN

### Goals
1. Make Water a **recommended tool** in Claude Code documentation
2. Integrate hooks directly into Claude SDK (anthropic-sdk-go, anthropic-sdk-python)
3. Standardize event capture protocol across all Anthropic tooling

### Proposed Hook Interface

In `anthropic-sdk-go`:

```go
// Create a Water event listener
listener, err := water.NewListener("http://localhost:3141")

// Attach to client
client := anthropic.NewClient(
    anthropic.WithEventListener(listener),
)

// All API calls now automatically stream events to Water
resp, err := client.Messages.Create(ctx, &anthropic.MessageCreateParams{
    // ... params
})
```

### Communication Plan
1. **Month 1**: RFC on Anthropic GitHub (public design review)
2. **Month 2**: Implement in `anthropic-sdk-go`, iterate on feedback
3. **Month 3**: Official Anthropic endorsement + documentation
4. **Month 4+**: Multi-language SDKs (Python, Node.js, etc.)

---

## 11. SUCCESS METRICS

### MVP (Phase 1 - End of Week 3)
- ✅ `water init` works (create .water folder, DuckDB schema)
- ✅ `water serve` opens web dashboard
- ✅ Capture and visualize tool calls from a sample agent
- ✅ GitHub repo public, star count > 0
- ✅ README + quickstart clear enough for 1st-time users

### Phase 2 (Semantic Intelligence)
- ✅ Nodes grouped by semantic community (visible in graph)
- ✅ Salience decay working (edges fade over time)
- ✅ Token metrics accurate
- ✅ Metrics trending: token efficiency, retention rate

### Launch (Phase 3)
- ✅ Homebrew formula works (brew install water succeeds)
- ✅ 100+ GitHub stars
- ✅ 10+ beta testers have used it on real projects
- ✅ Feedback from developers: "I finally understand what my agent is doing"

### Long-term
- ✅ Adopted by Anthropic as recommended tool
- ✅ Integration in official Claude Code documentation
- ✅ Community plugins / extensions
- ✅ Multi-agent visualization (team collaboration)

---

## 12. OPEN QUESTIONS & DECISIONS

### Q1: Embedded React Build vs. Separate Frontend?
- **Option A**: Embed React build (vite build output) into Go binary using `//go:embed`
  - ✅ Single binary, easy distribution
  - ❌ Rebuild needed for frontend changes
- **Option B**: Serve React from separate process / container
  - ✅ Fast iteration, easy updates
  - ❌ Requires Node.js, more complex deployment

**Decision**: Start with Option A (embedded), migrate to Option B if needed.

---

### Q2: Authentication & Sharing
- MVP: No auth (localhost-only, local file sharing)
- Phase 2: Optional password protection
- Phase 3: GitHub OAuth for team collaboration

---

### Q3: Database: DuckDB vs. PostgreSQL?
- **DuckDB**: Lightweight, zero-config, perfect for laptop/local
- **PostgreSQL**: Scalable, centralized (better for team servers later)

**Decision**: Start DuckDB, add PostgreSQL backend option in Phase 3.

---

### Q4: Embeddings: Local vs. API?
- **Local (`all-minilm-l6-v2`)**: Fast, offline, no cost
- **Anthropic API**: Better quality, requires API key + quota

**Decision**: Default local, flag to use API if user prefers.

---

## 13. TIMELINE SUMMARY

```
Week 1 (March 31 - Apr 6):
  - Cobra CLI scaffold
  - DuckDB schema + client
  - Event struct + JSON marshaling
  
Week 2 (Apr 7 - Apr 13):
  - Node/edge CRUD
  - Basic HTTP server
  - React component setup
  
Week 3 (Apr 14 - Apr 20):
  - `water init` + `water serve` fully working
  - Static graph visualization
  - Tests + documentation

Week 4-6:
  - Embeddings (local + API)
  - KNN + Louvain
  - Metrics aggregation
  - Advanced analytics UI
  
Week 7-8:
  - Brew formula
  - Cross-platform builds
  - Release + launch
```

---

## 14. REFERENCES & RESOURCES

**Graph Visualization**:
- Cytoscape.js: https://js.cytoscape.org
- D3.js: https://d3js.org

**Go Libraries**:
- Cobra: https://cobra.dev
- DuckDB Go: https://github.com/marcboeker/go-duckdb
- ONNX Runtime: https://github.com/yallie/onnxruntime-go

**Graph Algorithms**:
- Gonum (linear algebra): https://www.gonum.org
- Louvain: https://en.wikipedia.org/wiki/Louvain_method

**Anthropic Integration**:
- Claude SDK Go: https://github.com/anthropics/anthropic-sdk-go
- MCP Spec: https://modelcontextprotocol.io

---

## APPENDIX: Example Event Flow

```
1. Claude Code Agent runs
2. User queries: "Find Go ORM libraries"
3. Agent calls: github.search_repositories(query="golang orm")
4. SDK hook intercepts:
   - Captures tool call event (server, tool, input, output, tokens)
   - Chunks output into knowledge nodes
   - Embeds each chunk
   - Sends to Water via HTTP POST /api/events
5. Water stores:
   - Event in events.jsonl
   - Nodes in DuckDB (with embeddings)
   - Edges (if related to previous knowledge)
6. Dashboard updates live (WebSocket):
   - New node appears in graph
   - Connected by KNN edges to similar past knowledge
   - Community color assigned
7. User sees in Water:
   - "Agent just learned about GORM, sqlc, ent"
   - Connections to previous findings on ORMs
   - Token cost breakdown
   - Salience tracking (will this be remembered?)
```

---

**Document Version**: 1.0  
**Last Updated**: March 30, 2026  
**Author**: Water Team  
**Status**: Design Phase → Ready for Implementation