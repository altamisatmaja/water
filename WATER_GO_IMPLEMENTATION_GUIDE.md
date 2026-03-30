# Water: Go Implementation Guide
## Detailed Build Instructions & Code Patterns

---

## PART 1: PROJECT INITIALIZATION

### Step 1: Create Go Module & Svelte Frontend

```bash
# Backend (Go)
mkdir water
cd water
git init
git config user.email "you@example.com"
git config user.name "Your Name"

go mod init github.com/water-viz/water

# Create directory structure
mkdir -p cmd/water internal/{capture,graph,server,metrics,config,logger} pkg/{duckdb,embedding,math} test/fixtures test/integration

# Frontend (Svelte)
npm create vite@latest web -- --template svelte
cd web
npm install
cd ..
```

### Step 2: Initialize Cobra CLI

```bash
go get -u github.com/spf13/cobra/v2
go get -u github.com/spf13/viper

# Create cobra commands
cobra-cli init
cobra-cli add init
cobra-cli add serve
cobra-cli add watch
cobra-cli add export
cobra-cli add config
cobra-cli add install
```

### Step 3: Add Core Go Dependencies

```bash
# Database
go get github.com/marcboeker/go-duckdb

# Configuration
go get github.com/spf13/viper
go get github.com/spf13/cobra/v2

# Logging
go get golang.org/x/exp/slog

# Embeddings (later phase)
go get github.com/yallie/onnxruntime-go

# HTTP / WebSocket
go get github.com/gorilla/websocket

# Testing
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/require

# Utilities
go get github.com/google/uuid
go get golang.org/x/sync
```

### Step 4: Setup Svelte Dependencies

```bash
cd web

# Core Svelte
npm install -D vite svelte svelte-check

# UI & Styling
npm install tailwindcss postcss autoprefixer
npm install -D @tailwindcss/forms @tailwindcss/typography

# Graph Visualization
npm install cytoscape cytoscape-elk

# Data & HTTP
npm install axios
npm install date-fns

# Charting
npm install chart.js

# Utilities
npm install clsx zustand

# Initialize Tailwind
npx tailwindcss init -p

cd ..
```

### Step 5: Create Makefile

```makefile
.PHONY: help setup build test run clean lint fmt

help:
	@echo "Available targets:"
	@grep -E '^[a-z-]+:' Makefile | sed 's/:.*//g'

setup:
	go mod download
	go mod tidy

build:
	go build -o bin/water ./cmd/water

build-all:
	GOOS=darwin GOARCH=amd64 go build -o dist/water-darwin-amd64 ./cmd/water
	GOOS=darwin GOARCH=arm64 go build -o dist/water-darwin-arm64 ./cmd/water
	GOOS=linux GOARCH=amd64 go build -o dist/water-linux-amd64 ./cmd/water
	GOOS=linux GOARCH=arm64 go build -o dist/water-linux-arm64 ./cmd/water
	GOOS=windows GOARCH=amd64 go build -o dist/water-windows-amd64.exe ./cmd/water

test:
	go test -v -cover ./...

test-integration:
	go test -v -tags=integration ./test/integration

run:
	go run ./cmd/water init --db-path=./.water-test
	go run ./cmd/water serve --db-path=./.water-test

clean:
	rm -rf bin/ dist/ .water-test/

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

mod-tidy:
	go mod tidy
```

---

## PART 2: CORE MODULES (Code Stubs)

### `internal/config/config.go`

```go
package config

import (
    "os"
    "path/filepath"
    
    "github.com/spf13/viper"
)

type Config struct {
    // Database
    DBPath string
    
    // Server
    Host string
    Port int
    
    // Embeddings
    EmbeddingMode string // "local" or "api"
    AnthropicAPIKey string
    
    // Logging
    LogLevel string
    
    // Feature flags
    EnableWebSocket bool
    EnableAnalytics bool
}

func LoadConfig(cfgPath string) (*Config, error) {
    v := viper.New()
    
    // Set defaults
    v.SetDefault("db_path", "./.water")
    v.SetDefault("host", "127.0.0.1")
    v.SetDefault("port", 3141)
    v.SetDefault("embedding_mode", "local")
    v.SetDefault("log_level", "info")
    v.SetDefault("enable_websocket", true)
    
    // Load from file if provided
    if cfgPath != "" {
        v.SetConfigFile(cfgPath)
        if err := v.ReadInConfig(); err != nil {
            return nil, err
        }
    }
    
    // Load from env
    v.BindEnv("anthropic_api_key", "ANTHROPIC_API_KEY")
    
    cfg := &Config{
        DBPath:          v.GetString("db_path"),
        Host:            v.GetString("host"),
        Port:            v.GetInt("port"),
        EmbeddingMode:   v.GetString("embedding_mode"),
        AnthropicAPIKey: v.GetString("anthropic_api_key"),
        LogLevel:        v.GetString("log_level"),
        EnableWebSocket: v.GetBool("enable_websocket"),
    }
    
    return cfg, nil
}

func (c *Config) Save(cfgPath string) error {
    v := viper.New()
    v.Set("db_path", c.DBPath)
    v.Set("host", c.Host)
    v.Set("port", c.Port)
    v.Set("embedding_mode", c.EmbeddingMode)
    v.Set("log_level", c.LogLevel)
    
    return v.WriteConfigAs(cfgPath)
}

func GetConfigPath(dbPath string) string {
    return filepath.Join(dbPath, "config.json")
}

func GetEventsPath(dbPath string) string {
    return filepath.Join(dbPath, "events.jsonl")
}
```

---

### `internal/logger/logger.go`

```go
package logger

import (
    "log/slog"
    "os"
)

var log *slog.Logger

func init() {
    log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
}

func SetLevel(level slog.Level) {
    log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: level,
    }))
}

func Info(msg string, args ...any) {
    log.Info(msg, args...)
}

func Error(msg string, args ...any) {
    log.Error(msg, args...)
}

func Debug(msg string, args ...any) {
    log.Debug(msg, args...)
}

func Warn(msg string, args ...any) {
    log.Warn(msg, args...)
}
```

---

### `internal/capture/event.go`

```go
package capture

import (
    "encoding/json"
    "time"
)

// EventType constants
const (
    EventTypeMCPToolCall   = "mcp_tool_call"
    EventTypeContextWindow = "context_window"
    EventTypeMemoryAccess  = "memory_access"
    EventTypeDecision      = "decision"
    EventTypeError         = "error"
)

// Event is the canonical event type
type Event struct {
    ID            string                 `json:"id"`
    Timestamp     time.Time              `json:"timestamp"`
    SessionID     string                 `json:"session_id"`
    AgentID       string                 `json:"agent_id"`
    EventType     string                 `json:"event_type"`
    
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
    Input         json.RawMessage        `json:"input"`
    Output        json.RawMessage        `json:"output"`
    InputTokens   int64                  `json:"input_tokens"`
    OutputTokens  int64                  `json:"output_tokens"`
    ExecutionMs   int64                  `json:"execution_ms"`
    Success       bool                   `json:"success"`
    ErrorMessage  *string                `json:"error_message,omitempty"`
}

type ContextWindowEvent struct {
    Role             string `json:"role"` // "user" or "assistant"
    PromptTokens     int64  `json:"prompt_tokens"`
    CompletionTokens int64  `json:"completion_tokens"`
    CachedTokens     int64  `json:"cached_tokens"`
    Model            string `json:"model"`
    Temperature      float64 `json:"temperature"`
    TopP             float64 `json:"top_p"`
}

type MemoryAccessEvent struct {
    ChunkID            string  `json:"chunk_id"`
    ContentPreview     string  `json:"content_preview"`
    AccessType         string  `json:"access_type"` // retrieve, update, create, delete
    ImportanceScore    float64 `json:"importance_score"`
    RetentionConfidence float64 `json:"retention_confidence"`
    AgeSeconds         int64   `json:"age_seconds"`
}

type DecisionEvent struct {
    NodeID     string   `json:"node_id"`
    Description string  `json:"description"`
    Options    []string `json:"options"`
    Chosen     string   `json:"chosen"`
    Reasoning  string   `json:"reasoning"`
    Confidence float64  `json:"confidence"`
}

type ErrorEvent struct {
    ErrorType  string  `json:"error_type"`
    Message    string  `json:"message"`
    StackTrace *string `json:"stack_trace,omitempty"`
}
```

---

### `internal/capture/stream.go`

```go
package capture

import (
    "bufio"
    "encoding/json"
    "os"
    "sync"
)

type EventStream struct {
    filePath string
    mu       sync.Mutex
    file     *os.File
}

func NewEventStream(filePath string) (*EventStream, error) {
    // Create file if not exists
    f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
    if err != nil {
        return nil, err
    }
    
    return &EventStream{
        filePath: filePath,
        file:     f,
    }, nil
}

func (es *EventStream) Write(event *Event) error {
    es.mu.Lock()
    defer es.mu.Unlock()
    
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
    _, err = es.file.WriteString(string(data) + "\n")
    return err
}

func (es *EventStream) ReadAll() ([]*Event, error) {
    es.mu.Lock()
    defer es.mu.Unlock()
    
    f, err := os.Open(es.filePath)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    
    var events []*Event
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        var evt Event
        if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
            // Log but continue
            continue
        }
        events = append(events, &evt)
    }
    
    return events, scanner.Err()
}

func (es *EventStream) Close() error {
    es.mu.Lock()
    defer es.mu.Unlock()
    
    if es.file != nil {
        return es.file.Close()
    }
    return nil
}
```

---

### `internal/graph/client.go`

```go
package graph

import (
    "context"
    "database/sql"
    "fmt"
    
    _ "github.com/marcboeker/go-duckdb"
    "github.com/water-viz/water/internal/logger"
)

type Client struct {
    db     *sql.DB
    dbPath string
}

func NewClient(ctx context.Context, dbPath string) (*Client, error) {
    // Connect to DuckDB
    connStr := fmt.Sprintf("file:%s/database.duckdb", dbPath)
    db, err := sql.Open("duckdb", connStr)
    if err != nil {
        return nil, err
    }
    
    // Test connection
    if err := db.PingContext(ctx); err != nil {
        return nil, err
    }
    
    c := &Client{
        db:     db,
        dbPath: dbPath,
    }
    
    // Initialize schema
    if err := c.initSchema(ctx); err != nil {
        return nil, err
    }
    
    logger.Info("DuckDB client initialized", "path", dbPath)
    return c, nil
}

func (c *Client) initSchema(ctx context.Context) error {
    schema := `
    CREATE TABLE IF NOT EXISTS knowledge_nodes (
        node_id TEXT PRIMARY KEY,
        content TEXT NOT NULL,
        content_hash TEXT,
        source_type TEXT,
        source_tool TEXT,
        tokens_in BIGINT,
        tokens_out BIGINT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        first_accessed_at TIMESTAMP,
        last_accessed_at TIMESTAMP,
        access_count BIGINT DEFAULT 0,
        importance_score FLOAT,
        retention_confidence FLOAT,
        tags TEXT[]
    );
    
    CREATE TABLE IF NOT EXISTS edges (
        edge_id TEXT PRIMARY KEY,
        from_node_id TEXT REFERENCES knowledge_nodes(node_id),
        to_node_id TEXT REFERENCES knowledge_nodes(node_id),
        edge_type TEXT,
        weight FLOAT,
        salience FLOAT,
        reasoning_path TEXT,
        community_id INT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );
    
    CREATE INDEX IF NOT EXISTS idx_from ON edges(from_node_id);
    CREATE INDEX IF NOT EXISTS idx_to ON edges(to_node_id);
    
    CREATE TABLE IF NOT EXISTS reasoning_traces (
        trace_id TEXT PRIMARY KEY,
        session_id TEXT,
        timestamp TIMESTAMP,
        nodes_path TEXT[],
        edge_path TEXT[],
        depth INT,
        decision_points JSON,
        total_tokens INT,
        latency_ms BIGINT,
        tool_calls TEXT[],
        outcome TEXT
    );
    `
    
    _, err := c.db.ExecContext(ctx, schema)
    return err
}

type Node struct {
    NodeID               string
    Content             string
    SourceType          string
    SourceTool          *string
    TokensIn            int64
    TokensOut           int64
    AccessCount         int64
    ImportanceScore     float64
    RetentionConfidence float64
}

func (c *Client) InsertNode(ctx context.Context, node *Node) error {
    _, err := c.db.ExecContext(ctx, `
        INSERT INTO knowledge_nodes (
            node_id, content, source_type, source_tool, tokens_in, tokens_out,
            access_count, importance_score, retention_confidence
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, node.NodeID, node.Content, node.SourceType, node.SourceTool,
       node.TokensIn, node.TokensOut, node.AccessCount,
       node.ImportanceScore, node.RetentionConfidence)
    
    return err
}

func (c *Client) GetNode(ctx context.Context, nodeID string) (*Node, error) {
    var node Node
    err := c.db.QueryRowContext(ctx, `
        SELECT node_id, content, source_type, source_tool, tokens_in, tokens_out,
               access_count, importance_score, retention_confidence
        FROM knowledge_nodes
        WHERE node_id = ?
    `, nodeID).Scan(&node.NodeID, &node.Content, &node.SourceType, &node.SourceTool,
                     &node.TokensIn, &node.TokensOut, &node.AccessCount,
                     &node.ImportanceScore, &node.RetentionConfidence)
    
    if err != nil {
        return nil, err
    }
    return &node, nil
}

func (c *Client) Close() error {
    return c.db.Close()
}
```

---

### `cmd/water/main.go`

```go
package main

import (
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    "github.com/water-viz/water/internal/logger"
)

var rootCmd = &cobra.Command{
    Use:   "water",
    Short: "Visual brain of MCP agents",
    Long: `Water is a visualization tool for Claude Code agents and MCP-based systems.
    
It captures, analyzes, and visualizes:
- Memory & knowledge graphs
- Reasoning paths
- Token economics
- Team insights`,
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        logger.Error("command failed", "error", err)
        os.Exit(1)
    }
}

func main() {
    Execute()
}
```

---

### `cmd/water/init.go`

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    
    "github.com/spf13/cobra"
    "github.com/water-viz/water/internal/config"
    "github.com/water-viz/water/internal/logger"
)

var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize Water in current project",
    RunE: func(cmd *cobra.Command, args []string) error {
        dbPath := cmd.Flag("db-path").Value.String()
        if dbPath == "" {
            dbPath = ".water"
        }
        
        // Create .water directory
        if err := os.MkdirAll(dbPath, 0755); err != nil {
            return fmt.Errorf("failed to create directory: %w", err)
        }
        
        // Create config.json
        cfg := &config.Config{
            DBPath:          dbPath,
            Host:            "127.0.0.1",
            Port:            3141,
            EmbeddingMode:   "local",
            LogLevel:        "info",
            EnableWebSocket: true,
        }
        
        cfgPath := config.GetConfigPath(dbPath)
        if err := cfg.Save(cfgPath); err != nil {
            return fmt.Errorf("failed to save config: %w", err)
        }
        
        // Create .gitignore
        gitignorePath := filepath.Join(dbPath, ".gitignore")
        if err := os.WriteFile(gitignorePath, []byte("database.duckdb\nevents.jsonl\n"), 0644); err != nil {
            return fmt.Errorf("failed to create .gitignore: %w", err)
        }
        
        logger.Info("Water initialized successfully", "path", dbPath)
        fmt.Printf("✓ Created %s/\n", dbPath)
        fmt.Printf("✓ Created %s\n", cfgPath)
        fmt.Printf("\nNext: water serve --db-path %s\n", dbPath)
        
        return nil
    },
}

func init() {
    rootCmd.AddCommand(initCmd)
    initCmd.Flags().String("db-path", ".water", "Path to .water directory")
    initCmd.Flags().String("embedding-mode", "local", "Embedding mode: local or api")
    initCmd.Flags().Int("port", 3141, "Web server port")
}
```

---

### `cmd/water/serve.go`

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os/exec"
    "runtime"
    
    "github.com/spf13/cobra"
    "github.com/water-viz/water/internal/config"
    "github.com/water-viz/water/internal/graph"
    "github.com/water-viz/water/internal/logger"
    "github.com/water-viz/water/internal/server"
)

var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start the Web server and dashboard",
    RunE: func(cmd *cobra.Command, args []string) error {
        dbPath := cmd.Flag("db-path").Value.String()
        if dbPath == "" {
            dbPath = ".water"
        }
        
        // Load config
        cfg, err := config.LoadConfig(config.GetConfigPath(dbPath))
        if err != nil {
            return err
        }
        
        // Initialize graph client
        ctx := context.Background()
        graphClient, err := graph.NewClient(ctx, dbPath)
        if err != nil {
            return err
        }
        defer graphClient.Close()
        
        // Create server
        srv := server.NewServer(cfg, graphClient)
        
        // Start server
        addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
        logger.Info("Starting server", "addr", addr)
        
        // Open browser
        openBrowser := cmd.Flag("open-browser").Value.String() == "true"
        if openBrowser {
            url := fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port)
            go openURL(url)
        }
        
        // Start serving
        return http.ListenAndServe(addr, srv.Router())
    },
}

func openURL(url string) {
    var cmd *exec.Cmd
    switch runtime.GOOS {
    case "darwin":
        cmd = exec.Command("open", url)
    case "linux":
        cmd = exec.Command("xdg-open", url)
    case "windows":
        cmd = exec.Command("start", url)
    }
    
    if cmd != nil {
        cmd.Run()
    }
}

func init() {
    rootCmd.AddCommand(serveCmd)
    serveCmd.Flags().String("db-path", ".water", "Path to .water directory")
    serveCmd.Flags().String("host", "127.0.0.1", "Server host")
    serveCmd.Flags().Int("port", 3141, "Server port")
    serveCmd.Flags().BoolP("open-browser", "o", true, "Auto-open browser")
}
```

---

### `internal/server/server.go`

```go
package server

import (
    "net/http"
    
    "github.com/water-viz/water/internal/config"
    "github.com/water-viz/water/internal/graph"
)

type Server struct {
    config *config.Config
    graph  *graph.Client
}

func NewServer(cfg *config.Config, g *graph.Client) *Server {
    return &Server{
        config: cfg,
        graph:  g,
    }
}

func (s *Server) Router() http.Handler {
    mux := http.NewServeMux()
    
    // API endpoints
    mux.HandleFunc("GET /api/nodes", s.handleGetNodes)
    mux.HandleFunc("GET /api/edges", s.handleGetEdges)
    mux.HandleFunc("GET /api/stats", s.handleGetStats)
    mux.HandleFunc("POST /api/events", s.handlePostEvent)
    
    // Static files (embedded React build)
    mux.HandleFunc("GET /", s.handleIndex)
    mux.HandleFunc("GET /static/", s.handleStatic)
    
    // Add middleware
    return s.withCORS(s.withLogging(mux))
}

func (s *Server) handleGetNodes(w http.ResponseWriter, r *http.Request) {
    // TODO: Fetch nodes from graph
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"nodes":[]}`))
}

func (s *Server) handleGetEdges(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"edges":[]}`))
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"stats":{}}`))
}

func (s *Server) handlePostEvent(w http.ResponseWriter, r *http.Request) {
    // TODO: Parse and store event
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
    // TODO: Serve embedded React build
    w.Header().Set("Content-Type", "text/html")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`<html><body>Water Dashboard</body></html>`))
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
    // TODO: Serve static files
    http.NotFound(w, r)
}

func (s *Server) withCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
        
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}

func (s *Server) withLogging(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // TODO: Add structured logging
        next.ServeHTTP(w, r)
    })
}
```

---

## PART 3: TESTING STRATEGY

### `test/fixtures/sample_events.jsonl`

```jsonl
{"id":"evt-001","timestamp":"2026-03-30T14:00:00Z","session_id":"sess-1","agent_id":"agent-1","event_type":"mcp_tool_call","mcp_tool_call":{"server_name":"github","tool_name":"search_repositories","input":{"query":"golang orm"},"output":{"results":[]},"input_tokens":100,"output_tokens":500,"execution_ms":1200,"success":true}}
{"id":"evt-002","timestamp":"2026-03-30T14:00:02Z","session_id":"sess-1","agent_id":"agent-1","event_type":"context_window","context_window":{"role":"user","prompt_tokens":2048,"completion_tokens":256,"cached_tokens":1024,"model":"claude-opus-4-6","temperature":0.7,"top_p":1.0}}
```

### `test/integration/graph_test.go`

```go
package integration

import (
    "context"
    "os"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/water-viz/water/internal/graph"
)

func TestGraphClient(t *testing.T) {
    // Create temporary directory
    tmpDir := t.TempDir()
    
    // Initialize client
    ctx := context.Background()
    client, err := graph.NewClient(ctx, tmpDir)
    require.NoError(t, err)
    defer client.Close()
    
    // Insert a node
    node := &graph.Node{
        NodeID:      "node-1",
        Content:     "Test content",
        SourceType:  "mcp_output",
        TokensIn:    100,
        TokensOut:   200,
        AccessCount: 1,
    }
    
    err = client.InsertNode(ctx, node)
    require.NoError(t, err)
    
    // Retrieve node
    retrieved, err := client.GetNode(ctx, "node-1")
    require.NoError(t, err)
    
    assert.Equal(t, "node-1", retrieved.NodeID)
    assert.Equal(t, "Test content", retrieved.Content)
}
```

---

## PART 4: BUILD & DEPLOYMENT

### Cross-Compilation Script

```bash
#!/bin/bash
# scripts/cross-compile.sh

set -e

VERSION=${1:-0.1.0}
TARGETS=(
    "darwin:amd64"
    "darwin:arm64"
    "linux:amd64"
    "linux:arm64"
    "windows:amd64"
)

mkdir -p dist

for target in "${TARGETS[@]}"; do
    IFS=':' read -r GOOS GOARCH <<< "$target"
    
    OUTPUT="water"
    [ "$GOOS" = "windows" ] && OUTPUT="water.exe"
    
    echo "Building $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "-X main.Version=$VERSION" \
        -o dist/${OUTPUT}-${GOOS}-${GOARCH} \
        ./cmd/water
done

echo "Build complete. Artifacts in dist/"
```

### Homebrew Formula

```ruby
# Formula/water.rb
class Water < Formula
  desc "Visual brain of MCP agents"
  homepage "https://github.com/water-viz/water"
  url "https://github.com/water-viz/water/releases/download/v0.1.0/water-0.1.0-darwin-amd64.tar.gz"
  version "0.1.0"
  sha256 "abc123..."
  
  def install
    bin.install "water"
  end
  
  def post_install
    system "#{bin}/water", "init", "--db-path=#{var}/lib/water"
  end
  
  service do
    run [opt_bin/"water", "serve", "--db-path=#{var}/lib/water"]
    keep_alive true
    log_path var/"log/water.log"
  end
end
```

---

## NEXT STEPS

1. **Week 1**: Implement `config/`, `capture/`, `graph/client`, and basic CLI stubs
2. **Week 2**: Build HTTP server + DuckDB schema integration
3. **Week 3**: Create React frontend scaffold and connect to backend
4. **Week 4+**: Add intelligence (vectors, KNN, Louvain)

Good luck! 🚀