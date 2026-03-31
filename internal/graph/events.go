package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/water-viz/water/internal/capture"
)

// IngestEvent processes a captured event and stores it in the database.
func (c *Client) IngestEvent(ctx context.Context, evt *capture.Event) error {
	if evt == nil {
		return fmt.Errorf("ingest event: nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Store raw event in events_log
	_, err := c.db.ExecContext(ctx, `
		INSERT INTO events_log (id, timestamp, session_id, agent, event_type, content)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (id) DO NOTHING
	`, evt.ID, evt.Timestamp, evt.SessionID, evt.Agent, evt.EventType, evt.Content)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	// Create knowledge node from output events
	if evt.EventType == capture.EventTypeOutput && evt.Output != nil {
		nodeID := "node-" + evt.ID[:8]
		node := &Node{
			NodeID:               nodeID,
			Content:              evt.Output.Response,
			SourceType:           "agent_output",
			SourceTool:           &evt.Agent,
			TokensOut:            int64(evt.Output.Length),
			ImportanceScore:      0.7, // outputs are important by default
			RetentionConfidence:  0.8,
			CreatedAt:            time.Now().UTC(),
		}
		// Ignore errors - node might already exist
		_ = c.insertNodeUnsafe(ctx, node)
	}

	// Update or create session
	if err := c.updateSessionUnsafe(ctx, evt.SessionID, evt.Agent); err != nil {
		return fmt.Errorf("update session: %w", err)
	}

	return nil
}

// GetEvents retrieves events for a session, optionally filtered by type and agent.
func (c *Client) GetEvents(ctx context.Context, sessionID string, eventType, agent string, limit int) ([]*capture.Event, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT id, timestamp, session_id, agent, event_type, content 
		FROM events_log WHERE session_id = ?`
	args := []any{sessionID}

	if eventType != "" {
		query += ` AND event_type = ?`
		args = append(args, eventType)
	}
	if agent != "" {
		query += ` AND agent = ?`
		args = append(args, agent)
	}

	query += ` ORDER BY timestamp DESC LIMIT ?`
	args = append(args, limit)

	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	var events []*capture.Event
	for rows.Next() {
		var evt capture.Event
		if err := rows.Scan(&evt.ID, &evt.Timestamp, &evt.SessionID, &evt.Agent, &evt.EventType, &evt.Content); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, &evt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}

	// Reverse to chronological order
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}

	return events, nil
}

// insertNodeUnsafe inserts a node without taking the mutex (assumes it's held).
func (c *Client) insertNodeUnsafe(ctx context.Context, n *Node) error {
	if n == nil || n.NodeID == "" || n.Content == "" {
		return nil
	}

	tagsJSON := "[]"
	if len(n.Tags) > 0 {
		b, _ := json.Marshal(n.Tags)
		tagsJSON = string(b)
	}

	_, err := c.db.ExecContext(ctx, `
		INSERT INTO knowledge_nodes
			(node_id, content, source_type, source_tool,
			 tokens_in, tokens_out, created_at, last_accessed_at, access_count,
			 importance_score, retention_confidence, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (node_id) DO NOTHING
	`, n.NodeID, n.Content, n.SourceType, n.SourceTool,
		n.TokensIn, n.TokensOut, n.CreatedAt, nil, 0,
		n.ImportanceScore, n.RetentionConfidence, tagsJSON)

	return err
}

// updateSessionUnsafe updates a session without taking the mutex.
func (c *Client) updateSessionUnsafe(ctx context.Context, sessionID, agent string) error {
	// Try to insert first
	_, err := c.db.ExecContext(ctx, `
		INSERT INTO sessions (session_id, created_at, agents, total_events)
		VALUES (?, ?, ?, 1)
		ON CONFLICT (session_id) DO UPDATE SET
			total_events = total_events + 1
	`, sessionID, time.Now().UTC(), agent)

	return err
}
