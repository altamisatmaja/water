package graph

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

func (c *Client) InsertNode(ctx context.Context, n *Node) error {
	if n == nil {
		return fmt.Errorf("insert node: nil")
	}
	if n.NodeID == "" {
		return fmt.Errorf("insert node: missing node_id")
	}
	if n.Content == "" {
		return fmt.Errorf("insert node %s: missing content", n.NodeID)
	}
	if n.SourceType == "" {
		n.SourceType = "mcp_output"
	}
	if n.ContentHash == "" {
		sum := sha256.Sum256([]byte(n.Content))
		n.ContentHash = hex.EncodeToString(sum[:])
	}
	if n.ImportanceScore == 0 {
		n.ImportanceScore = 0.5
	}
	if n.RetentionConfidence == 0 {
		n.RetentionConfidence = 1.0
	}

	tagsJSON, err := json.Marshal(n.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	_, err = c.db.ExecContext(ctx, `
		INSERT INTO knowledge_nodes
			(node_id, content, content_hash, source_type, source_tool,
			 tokens_in, tokens_out, created_at, last_accessed_at, access_count,
			 importance_score, retention_confidence, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (node_id) DO UPDATE SET
			content = excluded.content,
			content_hash = excluded.content_hash,
			source_type = excluded.source_type,
			source_tool = excluded.source_tool,
			tokens_in = excluded.tokens_in,
			tokens_out = excluded.tokens_out,
			last_accessed_at = excluded.last_accessed_at,
			access_count = knowledge_nodes.access_count + 1,
			importance_score = excluded.importance_score,
			retention_confidence = excluded.retention_confidence,
			tags = excluded.tags
	`, n.NodeID, n.Content, n.ContentHash, n.SourceType, n.SourceTool,
		n.TokensIn, n.TokensOut, time.Now().UTC(), time.Now().UTC(), n.AccessCount,
		n.ImportanceScore, n.RetentionConfidence, string(tagsJSON),
	)
	if err != nil {
		return fmt.Errorf("insert node %s: %w", n.NodeID, err)
	}
	return nil
}

func (c *Client) GetNode(ctx context.Context, id string) (*Node, error) {
	row := c.db.QueryRowContext(ctx, `
		SELECT node_id, content, content_hash, source_type, source_tool,
		       tokens_in, tokens_out, created_at, last_accessed_at,
		       access_count, importance_score, retention_confidence, tags
		FROM knowledge_nodes
		WHERE node_id = ?
	`, id)

	var (
		n            Node
		sourceTool   sql.NullString
		lastAccessed sql.NullTime
		tagsStr      string
	)

	if err := row.Scan(
		&n.NodeID, &n.Content, &n.ContentHash, &n.SourceType, &sourceTool,
		&n.TokensIn, &n.TokensOut, &n.CreatedAt, &lastAccessed,
		&n.AccessCount, &n.ImportanceScore, &n.RetentionConfidence, &tagsStr,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get node %s: %w", id, err)
	}

	if sourceTool.Valid {
		n.SourceTool = &sourceTool.String
	}
	if lastAccessed.Valid {
		t := lastAccessed.Time
		n.LastAccessedAt = &t
	}
	_ = json.Unmarshal([]byte(tagsStr), &n.Tags)

	return &n, nil
}

func (c *Client) ListNodes(ctx context.Context, limit int) ([]*Node, error) {
	if limit <= 0 {
		limit = 200
	}

	rows, err := c.db.QueryContext(ctx, `
		SELECT node_id, content, content_hash, source_type, source_tool,
		       tokens_in, tokens_out, created_at, last_accessed_at,
		       access_count, importance_score, retention_confidence, tags
		FROM knowledge_nodes
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	defer rows.Close()

	var out []*Node
	for rows.Next() {
		var (
			n            Node
			sourceTool   sql.NullString
			lastAccessed sql.NullTime
			tagsStr      string
		)
		if err := rows.Scan(
			&n.NodeID, &n.Content, &n.ContentHash, &n.SourceType, &sourceTool,
			&n.TokensIn, &n.TokensOut, &n.CreatedAt, &lastAccessed,
			&n.AccessCount, &n.ImportanceScore, &n.RetentionConfidence, &tagsStr,
		); err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}

		if sourceTool.Valid {
			n.SourceTool = &sourceTool.String
		}
		if lastAccessed.Valid {
			t := lastAccessed.Time
			n.LastAccessedAt = &t
		}
		_ = json.Unmarshal([]byte(tagsStr), &n.Tags)
		out = append(out, &n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}
