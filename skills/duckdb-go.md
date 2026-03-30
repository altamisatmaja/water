# Skill: DuckDB Go Driver

Panduan lengkap penggunaan `github.com/marcboeker/go-duckdb` di proyek Water.

---

## Instalasi

```bash
go get github.com/marcboeker/go-duckdb
```

> ⚠️ DuckDB Go driver memerlukan CGo. Pastikan C compiler tersedia (`gcc` / `clang`).

---

## Koneksi Dasar

```go
// internal/graph/client.go
package graph

import (
    "context"
    "database/sql"
    "fmt"
    "sync"
    
    _ "github.com/marcboeker/go-duckdb"
)

type Client struct {
    db  *sql.DB
    mu  sync.Mutex  // DuckDB: single-writer, lock sebelum write
    path string
}

func NewClient(ctx context.Context, dbPath string) (*Client, error) {
    dbFile := filepath.Join(dbPath, "database.duckdb")
    
    db, err := sql.Open("duckdb", dbFile)
    if err != nil {
        return nil, fmt.Errorf("open duckdb: %w", err)
    }
    
    // DuckDB: gunakan 1 connection untuk write, banyak untuk read
    db.SetMaxOpenConns(1)
    
    if err := db.PingContext(ctx); err != nil {
        return nil, fmt.Errorf("ping duckdb: %w", err)
    }
    
    c := &Client{db: db, path: dbPath}
    
    if err := c.initSchema(ctx); err != nil {
        return nil, fmt.Errorf("init schema: %w", err)
    }
    
    return c, nil
}

func (c *Client) Close() error {
    return c.db.Close()
}
```

---

## Schema Initialization

```go
// internal/graph/schema.go
package graph

import (
    "context"
    "fmt"
)

const createSchema = `
CREATE TABLE IF NOT EXISTS knowledge_nodes (
    node_id             TEXT PRIMARY KEY,
    content             TEXT NOT NULL,
    content_hash        TEXT,
    source_type         TEXT,
    source_tool         TEXT,
    tokens_in           BIGINT DEFAULT 0,
    tokens_out          BIGINT DEFAULT 0,
    created_at          TIMESTAMP DEFAULT NOW(),
    first_accessed_at   TIMESTAMP,
    last_accessed_at    TIMESTAMP,
    access_count        BIGINT DEFAULT 0,
    importance_score    FLOAT DEFAULT 0.5,
    retention_confidence FLOAT DEFAULT 1.0,
    tags                TEXT[]
);

CREATE TABLE IF NOT EXISTS edges (
    edge_id         TEXT PRIMARY KEY,
    from_node_id    TEXT REFERENCES knowledge_nodes(node_id),
    to_node_id      TEXT REFERENCES knowledge_nodes(node_id),
    edge_type       TEXT,
    weight          FLOAT DEFAULT 1.0,
    salience        FLOAT DEFAULT 1.0,
    reasoning_path  TEXT,
    community_id    INT,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reasoning_traces (
    trace_id        TEXT PRIMARY KEY,
    session_id      TEXT,
    timestamp       TIMESTAMP DEFAULT NOW(),
    nodes_path      TEXT[],
    edge_path       TEXT[],
    depth           INT DEFAULT 0,
    total_tokens    INT DEFAULT 0,
    latency_ms      BIGINT DEFAULT 0,
    tool_calls      TEXT[],
    outcome         TEXT DEFAULT 'unknown'
);

CREATE TABLE IF NOT EXISTS daily_metrics (
    date            DATE,
    node_id         TEXT,
    access_count    BIGINT DEFAULT 0,
    avg_latency_ms  FLOAT DEFAULT 0,
    token_cost      INT DEFAULT 0,
    retention_rate  FLOAT DEFAULT 1.0,
    importance_trend FLOAT DEFAULT 0,
    community_id    INT,
    PRIMARY KEY (date, node_id)
);
`

func (c *Client) initSchema(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    _, err := c.db.ExecContext(ctx, createSchema)
    if err != nil {
        return fmt.Errorf("create schema: %w", err)
    }
    return nil
}
```

---

## CRUD: Knowledge Nodes

```go
// internal/graph/nodes.go
package graph

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    
    "github.com/google/uuid"
)

type Node struct {
    NodeID              string
    Content             string
    ContentHash         string
    SourceType          string
    SourceTool          string
    TokensIn            int64
    TokensOut           int64
    CreatedAt           time.Time
    LastAccessedAt      time.Time
    AccessCount         int64
    ImportanceScore     float64
    RetentionConfidence float64
    Tags                []string
}

func (c *Client) InsertNode(ctx context.Context, n *Node) error {
    if n.NodeID == "" {
        n.NodeID = "node-" + uuid.New().String()[:8]
    }
    
    c.mu.Lock()
    defer c.mu.Unlock()
    
    _, err := c.db.ExecContext(ctx, `
        INSERT INTO knowledge_nodes 
            (node_id, content, content_hash, source_type, source_tool,
             tokens_in, tokens_out, importance_score, retention_confidence)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT (node_id) DO NOTHING
    `,
        n.NodeID, n.Content, n.ContentHash, n.SourceType, n.SourceTool,
        n.TokensIn, n.TokensOut, n.ImportanceScore, n.RetentionConfidence,
    )
    if err != nil {
        return fmt.Errorf("insert node %s: %w", n.NodeID, err)
    }
    return nil
}

func (c *Client) GetNode(ctx context.Context, nodeID string) (*Node, error) {
    row := c.db.QueryRowContext(ctx, `
        SELECT node_id, content, content_hash, source_type, source_tool,
               tokens_in, tokens_out, access_count, importance_score,
               retention_confidence, created_at, last_accessed_at
        FROM knowledge_nodes WHERE node_id = ?
    `, nodeID)
    
    n := &Node{}
    err := row.Scan(
        &n.NodeID, &n.Content, &n.ContentHash, &n.SourceType, &n.SourceTool,
        &n.TokensIn, &n.TokensOut, &n.AccessCount, &n.ImportanceScore,
        &n.RetentionConfidence, &n.CreatedAt, &n.LastAccessedAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("get node %s: %w", nodeID, err)
    }
    return n, nil
}

func (c *Client) ListNodes(ctx context.Context, limit int) ([]*Node, error) {
    if limit <= 0 {
        limit = 100
    }
    
    rows, err := c.db.QueryContext(ctx, `
        SELECT node_id, content, source_type, tokens_in, tokens_out,
               access_count, importance_score, retention_confidence
        FROM knowledge_nodes
        ORDER BY last_accessed_at DESC NULLS LAST
        LIMIT ?
    `, limit)
    if err != nil {
        return nil, fmt.Errorf("list nodes: %w", err)
    }
    defer rows.Close()
    
    var nodes []*Node
    for rows.Next() {
        n := &Node{}
        if err := rows.Scan(
            &n.NodeID, &n.Content, &n.SourceType, &n.TokensIn, &n.TokensOut,
            &n.AccessCount, &n.ImportanceScore, &n.RetentionConfidence,
        ); err != nil {
            return nil, fmt.Errorf("scan node: %w", err)
        }
        nodes = append(nodes, n)
    }
    return nodes, rows.Err()
}
```

---

## CRUD: Edges

```go
// internal/graph/edges.go
package graph

import (
    "context"
    "fmt"
    "time"
    
    "github.com/google/uuid"
)

type Edge struct {
    EdgeID        string
    FromNodeID    string
    ToNodeID      string
    EdgeType      string  // "semantic" | "causal" | "retrieval"
    Weight        float64
    Salience      float64
    ReasoningPath string
    CommunityID   int
    CreatedAt     time.Time
}

func (c *Client) InsertEdge(ctx context.Context, e *Edge) error {
    if e.EdgeID == "" {
        e.EdgeID = "edge-" + uuid.New().String()[:8]
    }
    
    c.mu.Lock()
    defer c.mu.Unlock()
    
    _, err := c.db.ExecContext(ctx, `
        INSERT INTO edges (edge_id, from_node_id, to_node_id, edge_type, weight, salience, reasoning_path)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT (edge_id) DO UPDATE SET
            weight = excluded.weight,
            salience = excluded.salience,
            updated_at = NOW()
    `,
        e.EdgeID, e.FromNodeID, e.ToNodeID, e.EdgeType,
        e.Weight, e.Salience, e.ReasoningPath,
    )
    return err
}

func (c *Client) GetEdgesByNode(ctx context.Context, nodeID string) ([]*Edge, error) {
    rows, err := c.db.QueryContext(ctx, `
        SELECT edge_id, from_node_id, to_node_id, edge_type, weight, salience
        FROM edges
        WHERE from_node_id = ? OR to_node_id = ?
        ORDER BY salience DESC
    `, nodeID, nodeID)
    if err != nil {
        return nil, fmt.Errorf("get edges for %s: %w", nodeID, err)
    }
    defer rows.Close()
    
    var edges []*Edge
    for rows.Next() {
        e := &Edge{}
        if err := rows.Scan(&e.EdgeID, &e.FromNodeID, &e.ToNodeID, &e.EdgeType, &e.Weight, &e.Salience); err != nil {
            return nil, err
        }
        edges = append(edges, e)
    }
    return edges, rows.Err()
}
```

---

## Tips Penting

### Concurrency
DuckDB tidak mendukung concurrent writes. Selalu gunakan mutex sebelum `INSERT/UPDATE/DELETE`:
```go
c.mu.Lock()
defer c.mu.Unlock()
_, err := c.db.ExecContext(ctx, ...)
```

### Null Handling
Gunakan `sql.NullString`, `sql.NullFloat64`, `sql.NullTime` untuk kolom nullable:
```go
var lastAccessed sql.NullTime
row.Scan(&lastAccessed)
if lastAccessed.Valid {
    node.LastAccessedAt = lastAccessed.Time
}
```

### Batch Insert (performa)
```go
tx, _ := c.db.BeginTx(ctx, nil)
stmt, _ := tx.PrepareContext(ctx, `INSERT INTO knowledge_nodes (...) VALUES (?, ?, ?)`)
for _, node := range nodes {
    stmt.ExecContext(ctx, node.NodeID, node.Content, node.SourceType)
}
stmt.Close()
tx.Commit()
```

### DuckDB Arrays
DuckDB `TEXT[]` tidak di-support langsung oleh `database/sql`. Gunakan JSON string sebagai workaround:
```go
// Insert
tagsJSON, _ := json.Marshal(node.Tags)
_, err = db.ExecContext(ctx, `INSERT INTO ... VALUES (...)`, string(tagsJSON))

// Select — parse manual setelah scan
var tagsStr string
row.Scan(&tagsStr)
json.Unmarshal([]byte(tagsStr), &node.Tags)
```# Skill: DuckDB Go Driver

Panduan lengkap penggunaan `github.com/marcboeker/go-duckdb` di proyek Water.

---

## Instalasi

```bash
go get github.com/marcboeker/go-duckdb
```

> ⚠️ DuckDB Go driver memerlukan CGo. Pastikan C compiler tersedia (`gcc` / `clang`).

---

## Koneksi Dasar

```go
// internal/graph/client.go
package graph

import (
    "context"
    "database/sql"
    "fmt"
    "sync"
    
    _ "github.com/marcboeker/go-duckdb"
)

type Client struct {
    db  *sql.DB
    mu  sync.Mutex  // DuckDB: single-writer, lock sebelum write
    path string
}

func NewClient(ctx context.Context, dbPath string) (*Client, error) {
    dbFile := filepath.Join(dbPath, "database.duckdb")
    
    db, err := sql.Open("duckdb", dbFile)
    if err != nil {
        return nil, fmt.Errorf("open duckdb: %w", err)
    }
    
    // DuckDB: gunakan 1 connection untuk write, banyak untuk read
    db.SetMaxOpenConns(1)
    
    if err := db.PingContext(ctx); err != nil {
        return nil, fmt.Errorf("ping duckdb: %w", err)
    }
    
    c := &Client{db: db, path: dbPath}
    
    if err := c.initSchema(ctx); err != nil {
        return nil, fmt.Errorf("init schema: %w", err)
    }
    
    return c, nil
}

func (c *Client) Close() error {
    return c.db.Close()
}
```

---

## Schema Initialization

```go
// internal/graph/schema.go
package graph

import (
    "context"
    "fmt"
)

const createSchema = `
CREATE TABLE IF NOT EXISTS knowledge_nodes (
    node_id             TEXT PRIMARY KEY,
    content             TEXT NOT NULL,
    content_hash        TEXT,
    source_type         TEXT,
    source_tool         TEXT,
    tokens_in           BIGINT DEFAULT 0,
    tokens_out          BIGINT DEFAULT 0,
    created_at          TIMESTAMP DEFAULT NOW(),
    first_accessed_at   TIMESTAMP,
    last_accessed_at    TIMESTAMP,
    access_count        BIGINT DEFAULT 0,
    importance_score    FLOAT DEFAULT 0.5,
    retention_confidence FLOAT DEFAULT 1.0,
    tags                TEXT[]
);

CREATE TABLE IF NOT EXISTS edges (
    edge_id         TEXT PRIMARY KEY,
    from_node_id    TEXT REFERENCES knowledge_nodes(node_id),
    to_node_id      TEXT REFERENCES knowledge_nodes(node_id),
    edge_type       TEXT,
    weight          FLOAT DEFAULT 1.0,
    salience        FLOAT DEFAULT 1.0,
    reasoning_path  TEXT,
    community_id    INT,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reasoning_traces (
    trace_id        TEXT PRIMARY KEY,
    session_id      TEXT,
    timestamp       TIMESTAMP DEFAULT NOW(),
    nodes_path      TEXT[],
    edge_path       TEXT[],
    depth           INT DEFAULT 0,
    total_tokens    INT DEFAULT 0,
    latency_ms      BIGINT DEFAULT 0,
    tool_calls      TEXT[],
    outcome         TEXT DEFAULT 'unknown'
);

CREATE TABLE IF NOT EXISTS daily_metrics (
    date            DATE,
    node_id         TEXT,
    access_count    BIGINT DEFAULT 0,
    avg_latency_ms  FLOAT DEFAULT 0,
    token_cost      INT DEFAULT 0,
    retention_rate  FLOAT DEFAULT 1.0,
    importance_trend FLOAT DEFAULT 0,
    community_id    INT,
    PRIMARY KEY (date, node_id)
);
`

func (c *Client) initSchema(ctx context.Context) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    _, err := c.db.ExecContext(ctx, createSchema)
    if err != nil {
        return fmt.Errorf("create schema: %w", err)
    }
    return nil
}
```

---

## CRUD: Knowledge Nodes

```go
// internal/graph/nodes.go
package graph

import (
    "context"
    "database/sql"
    "fmt"
    "time"
    
    "github.com/google/uuid"
)

type Node struct {
    NodeID              string
    Content             string
    ContentHash         string
    SourceType          string
    SourceTool          string
    TokensIn            int64
    TokensOut           int64
    CreatedAt           time.Time
    LastAccessedAt      time.Time
    AccessCount         int64
    ImportanceScore     float64
    RetentionConfidence float64
    Tags                []string
}

func (c *Client) InsertNode(ctx context.Context, n *Node) error {
    if n.NodeID == "" {
        n.NodeID = "node-" + uuid.New().String()[:8]
    }
    
    c.mu.Lock()
    defer c.mu.Unlock()
    
    _, err := c.db.ExecContext(ctx, `
        INSERT INTO knowledge_nodes 
            (node_id, content, content_hash, source_type, source_tool,
             tokens_in, tokens_out, importance_score, retention_confidence)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT (node_id) DO NOTHING
    `,
        n.NodeID, n.Content, n.ContentHash, n.SourceType, n.SourceTool,
        n.TokensIn, n.TokensOut, n.ImportanceScore, n.RetentionConfidence,
    )
    if err != nil {
        return fmt.Errorf("insert node %s: %w", n.NodeID, err)
    }
    return nil
}

func (c *Client) GetNode(ctx context.Context, nodeID string) (*Node, error) {
    row := c.db.QueryRowContext(ctx, `
        SELECT node_id, content, content_hash, source_type, source_tool,
               tokens_in, tokens_out, access_count, importance_score,
               retention_confidence, created_at, last_accessed_at
        FROM knowledge_nodes WHERE node_id = ?
    `, nodeID)
    
    n := &Node{}
    err := row.Scan(
        &n.NodeID, &n.Content, &n.ContentHash, &n.SourceType, &n.SourceTool,
        &n.TokensIn, &n.TokensOut, &n.AccessCount, &n.ImportanceScore,
        &n.RetentionConfidence, &n.CreatedAt, &n.LastAccessedAt,
    )
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("get node %s: %w", nodeID, err)
    }
    return n, nil
}

func (c *Client) ListNodes(ctx context.Context, limit int) ([]*Node, error) {
    if limit <= 0 {
        limit = 100
    }
    
    rows, err := c.db.QueryContext(ctx, `
        SELECT node_id, content, source_type, tokens_in, tokens_out,
               access_count, importance_score, retention_confidence
        FROM knowledge_nodes
        ORDER BY last_accessed_at DESC NULLS LAST
        LIMIT ?
    `, limit)
    if err != nil {
        return nil, fmt.Errorf("list nodes: %w", err)
    }
    defer rows.Close()
    
    var nodes []*Node
    for rows.Next() {
        n := &Node{}
        if err := rows.Scan(
            &n.NodeID, &n.Content, &n.SourceType, &n.TokensIn, &n.TokensOut,
            &n.AccessCount, &n.ImportanceScore, &n.RetentionConfidence,
        ); err != nil {
            return nil, fmt.Errorf("scan node: %w", err)
        }
        nodes = append(nodes, n)
    }
    return nodes, rows.Err()
}
```

---

## CRUD: Edges

```go
// internal/graph/edges.go
package graph

import (
    "context"
    "fmt"
    "time"
    
    "github.com/google/uuid"
)

type Edge struct {
    EdgeID        string
    FromNodeID    string
    ToNodeID      string
    EdgeType      string  // "semantic" | "causal" | "retrieval"
    Weight        float64
    Salience      float64
    ReasoningPath string
    CommunityID   int
    CreatedAt     time.Time
}

func (c *Client) InsertEdge(ctx context.Context, e *Edge) error {
    if e.EdgeID == "" {
        e.EdgeID = "edge-" + uuid.New().String()[:8]
    }
    
    c.mu.Lock()
    defer c.mu.Unlock()
    
    _, err := c.db.ExecContext(ctx, `
        INSERT INTO edges (edge_id, from_node_id, to_node_id, edge_type, weight, salience, reasoning_path)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT (edge_id) DO UPDATE SET
            weight = excluded.weight,
            salience = excluded.salience,
            updated_at = NOW()
    `,
        e.EdgeID, e.FromNodeID, e.ToNodeID, e.EdgeType,
        e.Weight, e.Salience, e.ReasoningPath,
    )
    return err
}

func (c *Client) GetEdgesByNode(ctx context.Context, nodeID string) ([]*Edge, error) {
    rows, err := c.db.QueryContext(ctx, `
        SELECT edge_id, from_node_id, to_node_id, edge_type, weight, salience
        FROM edges
        WHERE from_node_id = ? OR to_node_id = ?
        ORDER BY salience DESC
    `, nodeID, nodeID)
    if err != nil {
        return nil, fmt.Errorf("get edges for %s: %w", nodeID, err)
    }
    defer rows.Close()
    
    var edges []*Edge
    for rows.Next() {
        e := &Edge{}
        if err := rows.Scan(&e.EdgeID, &e.FromNodeID, &e.ToNodeID, &e.EdgeType, &e.Weight, &e.Salience); err != nil {
            return nil, err
        }
        edges = append(edges, e)
    }
    return edges, rows.Err()
}
```

---

## Tips Penting

### Concurrency
DuckDB tidak mendukung concurrent writes. Selalu gunakan mutex sebelum `INSERT/UPDATE/DELETE`:
```go
c.mu.Lock()
defer c.mu.Unlock()
_, err := c.db.ExecContext(ctx, ...)
```

### Null Handling
Gunakan `sql.NullString`, `sql.NullFloat64`, `sql.NullTime` untuk kolom nullable:
```go
var lastAccessed sql.NullTime
row.Scan(&lastAccessed)
if lastAccessed.Valid {
    node.LastAccessedAt = lastAccessed.Time
}
```

### Batch Insert (performa)
```go
tx, _ := c.db.BeginTx(ctx, nil)
stmt, _ := tx.PrepareContext(ctx, `INSERT INTO knowledge_nodes (...) VALUES (?, ?, ?)`)
for _, node := range nodes {
    stmt.ExecContext(ctx, node.NodeID, node.Content, node.SourceType)
}
stmt.Close()
tx.Commit()
```

### DuckDB Arrays
DuckDB `TEXT[]` tidak di-support langsung oleh `database/sql`. Gunakan JSON string sebagai workaround:
```go
// Insert
tagsJSON, _ := json.Marshal(node.Tags)
_, err = db.ExecContext(ctx, `INSERT INTO ... VALUES (...)`, string(tagsJSON))

// Select — parse manual setelah scan
var tagsStr string
row.Scan(&tagsStr)
json.Unmarshal([]byte(tagsStr), &node.Tags)
```