package graph

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (c *Client) InsertEdge(ctx context.Context, e *Edge) error {
	if e == nil {
		return fmt.Errorf("insert edge: nil")
	}
	if e.EdgeID == "" {
		e.EdgeID = "edge-" + uuid.New().String()[:12]
	}
	if e.EdgeType == "" {
		e.EdgeType = "semantic"
	}
	if e.Weight == 0 {
		e.Weight = 1.0
	}
	if e.Salience == 0 {
		e.Salience = 1.0
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.db.ExecContext(ctx, `
		INSERT INTO edges (edge_id, from_node_id, to_node_id, edge_type, weight, salience)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (edge_id) DO UPDATE SET
			weight = excluded.weight,
			salience = excluded.salience,
			updated_at = NOW()
	`, e.EdgeID, e.FromNodeID, e.ToNodeID, e.EdgeType, e.Weight, e.Salience)
	if err != nil {
		return fmt.Errorf("insert edge %s: %w", e.EdgeID, err)
	}
	return nil
}

func (c *Client) ListEdges(ctx context.Context, limit int) ([]*Edge, error) {
	if limit <= 0 {
		limit = 500
	}
	rows, err := c.db.QueryContext(ctx, `
		SELECT edge_id, from_node_id, to_node_id, edge_type, weight, salience
		FROM edges
		ORDER BY salience DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list edges: %w", err)
	}
	defer rows.Close()

	var out []*Edge
	for rows.Next() {
		e := &Edge{}
		if err := rows.Scan(&e.EdgeID, &e.FromNodeID, &e.ToNodeID, &e.EdgeType, &e.Weight, &e.Salience); err != nil {
			return nil, fmt.Errorf("scan edge: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}

func (c *Client) GetFullGraph(ctx context.Context) (*GraphData, error) {
	nodes, err := c.ListNodes(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	edges, err := c.ListEdges(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("list edges: %w", err)
	}
	return &GraphData{Nodes: nodes, Edges: edges}, nil
}
