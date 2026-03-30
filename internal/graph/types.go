package graph

import "time"

type Node struct {
	NodeID              string     `json:"node_id"`
	Content             string     `json:"content"`
	ContentHash         string     `json:"content_hash,omitempty"`
	SourceType          string     `json:"source_type"`
	SourceTool          *string    `json:"source_tool"`
	TokensIn            int64      `json:"tokens_in"`
	TokensOut           int64      `json:"tokens_out"`
	CreatedAt           time.Time  `json:"created_at"`
	LastAccessedAt      *time.Time `json:"last_accessed_at,omitempty"`
	AccessCount         int64      `json:"access_count"`
	ImportanceScore     float64    `json:"importance_score"`
	RetentionConfidence float64    `json:"retention_confidence"`
	Tags                []string   `json:"tags"`
}

type Edge struct {
	EdgeID     string  `json:"edge_id"`
	FromNodeID string  `json:"from_node_id"`
	ToNodeID   string  `json:"to_node_id"`
	EdgeType   string  `json:"edge_type"`
	Weight     float64 `json:"weight"`
	Salience   float64 `json:"salience"`
}

type GraphData struct {
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`
}
