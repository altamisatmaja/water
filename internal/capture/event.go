package capture

import (
	"encoding/json"
	"time"
)

const (
	EventTypeMCPToolCall   = "mcp_tool_call"
	EventTypeContextWindow = "context_window"
	EventTypeMemoryAccess  = "memory_access"
	EventTypeDecision      = "decision"
	EventTypeError         = "error"
)

type Event struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	SessionID string    `json:"session_id"`
	AgentID   string    `json:"agent_id"`
	EventType string    `json:"event_type"`

	MCPToolCall   *MCPToolCallEvent   `json:"mcp_tool_call,omitempty"`
	ContextWindow *ContextWindowEvent `json:"context_window,omitempty"`
	MemoryAccess  *MemoryAccessEvent  `json:"memory_access,omitempty"`
	Decision      *DecisionEvent      `json:"decision,omitempty"`
	Error         *ErrorEvent         `json:"error,omitempty"`

	Metadata map[string]any `json:"metadata,omitempty"`
}

type MCPToolCallEvent struct {
	ServerName   string          `json:"server_name"`
	ToolName     string          `json:"tool_name"`
	Input        json.RawMessage `json:"input"`
	Output       json.RawMessage `json:"output"`
	InputTokens  int64           `json:"input_tokens"`
	OutputTokens int64           `json:"output_tokens"`
	ExecutionMs  int64           `json:"execution_ms"`
	Success      bool            `json:"success"`
	ErrorMessage *string         `json:"error_message,omitempty"`
}

type ContextWindowEvent struct {
	Role             string  `json:"role"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	CachedTokens     int64   `json:"cached_tokens"`
	Model            string  `json:"model"`
	Temperature      float64 `json:"temperature"`
	TopP             float64 `json:"top_p"`
}

type MemoryAccessEvent struct {
	ChunkID             string  `json:"chunk_id"`
	ContentPreview      string  `json:"content_preview"`
	AccessType          string  `json:"access_type"`
	ImportanceScore     float64 `json:"importance_score"`
	RetentionConfidence float64 `json:"retention_confidence"`
	AgeSeconds          int64   `json:"age_seconds"`
}

type DecisionEvent struct {
	NodeID      string   `json:"node_id"`
	Description string   `json:"description"`
	Options     []string `json:"options"`
	Chosen      string   `json:"chosen"`
	Reasoning   string   `json:"reasoning"`
	Confidence  float64  `json:"confidence"`
}

type ErrorEvent struct {
	ErrorType  string  `json:"error_type"`
	Message    string  `json:"message"`
	StackTrace *string `json:"stack_trace,omitempty"`
}
