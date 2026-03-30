package graph

const schemaSQL = `
CREATE TABLE IF NOT EXISTS knowledge_nodes (
    node_id              TEXT PRIMARY KEY,
    content              TEXT NOT NULL,
    content_hash         TEXT,
    source_type          TEXT CHECK(source_type IN ('mcp_output','context','memory')),
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

CREATE TABLE IF NOT EXISTS edges (
    edge_id        TEXT PRIMARY KEY,
    from_node_id   TEXT NOT NULL,
    to_node_id     TEXT NOT NULL,
    edge_type      TEXT CHECK(edge_type IN ('semantic','causal','retrieval')),
    weight         FLOAT DEFAULT 1.0 CHECK(weight BETWEEN 0 AND 1),
    salience       FLOAT DEFAULT 1.0 CHECK(salience BETWEEN 0 AND 1),
    reasoning_path TEXT,
    community_id   INTEGER,
    created_at     TIMESTAMP DEFAULT NOW(),
    updated_at     TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (from_node_id) REFERENCES knowledge_nodes(node_id),
    FOREIGN KEY (to_node_id)   REFERENCES knowledge_nodes(node_id)
);

CREATE INDEX IF NOT EXISTS idx_edges_from ON edges(from_node_id);
CREATE INDEX IF NOT EXISTS idx_edges_to ON edges(to_node_id);
`
