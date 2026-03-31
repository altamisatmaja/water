package graph

const schemaSQL = `
-- Events log: raw append-only event stream from wrappers
CREATE TABLE IF NOT EXISTS events_log (
    id TEXT PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    session_id TEXT NOT NULL,
    agent TEXT NOT NULL,
    event_type TEXT NOT NULL,
    content TEXT,
    metadata JSON,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Knowledge nodes: chunks of information agents learned or created
CREATE TABLE IF NOT EXISTS knowledge_nodes (
    node_id              TEXT PRIMARY KEY,
    session_id           TEXT,
    agent                TEXT,
    content              TEXT NOT NULL,
    content_hash         TEXT,
    source_type          TEXT,
    source_tool          TEXT,
    tokens_in            BIGINT DEFAULT 0,
    tokens_out           BIGINT DEFAULT 0,
    created_at           TIMESTAMP DEFAULT NOW(),
    first_accessed_at    TIMESTAMP,
    last_accessed_at     TIMESTAMP,
    access_count         BIGINT DEFAULT 0,
    importance_score     FLOAT DEFAULT 0.5 CHECK(importance_score BETWEEN 0 AND 1),
    retention_confidence FLOAT DEFAULT 1.0 CHECK(retention_confidence BETWEEN 0 AND 1),
    tags                 TEXT DEFAULT '[]'
);

-- Edges: relationships between knowledge nodes
CREATE TABLE IF NOT EXISTS edges (
    edge_id        TEXT PRIMARY KEY,
    from_node_id   TEXT NOT NULL,
    to_node_id     TEXT NOT NULL,
    edge_type      TEXT,
    weight         FLOAT DEFAULT 1.0 CHECK(weight BETWEEN 0 AND 1),
    salience       FLOAT DEFAULT 1.0 CHECK(salience BETWEEN 0 AND 1),
    reasoning_path TEXT,
    community_id   INTEGER,
    created_at     TIMESTAMP DEFAULT NOW(),
    updated_at     TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (from_node_id) REFERENCES knowledge_nodes(node_id),
    FOREIGN KEY (to_node_id)   REFERENCES knowledge_nodes(node_id)
);

-- Reasoning traces: ordered paths of decisions across operations
CREATE TABLE IF NOT EXISTS reasoning_traces (
    trace_id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    agent TEXT,
    step_count BIGINT DEFAULT 0,
    steps JSON,
    execution_context JSON,
    total_duration_ms BIGINT DEFAULT 0,
    token_usage JSON,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    status TEXT DEFAULT 'running',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Sessions: grouping of related agent operations
CREATE TABLE IF NOT EXISTS sessions (
    session_id TEXT PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    closed_at TIMESTAMP,
    agents TEXT,
    total_events BIGINT DEFAULT 0,
    total_nodes BIGINT DEFAULT 0,
    total_traces BIGINT DEFAULT 0,
    metadata JSON
);

-- Daily metrics: aggregated statistics
CREATE TABLE IF NOT EXISTS daily_metrics (
    metric_id TEXT PRIMARY KEY,
    date DATE NOT NULL,
    session_id TEXT,
    agent TEXT,
    total_tokens BIGINT,
    unique_nodes BIGINT,
    edges_created BIGINT,
    traces_completed BIGINT,
    avg_node_importance FLOAT,
    avg_retention_confidence FLOAT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for fast queries
CREATE INDEX IF NOT EXISTS idx_events_session_timestamp ON events_log(session_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_events_agent_type ON events_log(agent, event_type);
CREATE INDEX IF NOT EXISTS idx_nodes_session ON knowledge_nodes(session_id);
CREATE INDEX IF NOT EXISTS idx_nodes_source ON knowledge_nodes(source_type, source_tool);
CREATE INDEX IF NOT EXISTS idx_nodes_importance ON knowledge_nodes(importance_score DESC);
CREATE INDEX IF NOT EXISTS idx_edges_from ON edges(from_node_id);
CREATE INDEX IF NOT EXISTS idx_edges_to ON edges(to_node_id);
CREATE INDEX IF NOT EXISTS idx_traces_session_status ON reasoning_traces(session_id, status);
CREATE INDEX IF NOT EXISTS idx_sessions_created ON sessions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_date_agent ON daily_metrics(date DESC, agent);
`
