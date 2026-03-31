package capture

import (
	"encoding/json"
	"time"
)

// Event type constants
const (
	// Agent I/O tracing
	EventTypeInput             = "input"
	EventTypeOutput            = "output"
	EventTypeExecutionContext  = "execution_context"
	EventTypeFileAccess        = "file_access"
	EventTypeCommandExecution  = "command_execution"
	
	// Internal reasoning
	EventTypeMCPToolCall   = "mcp_tool_call"
	EventTypeContextWindow = "context_window"
	EventTypeMemoryAccess  = "memory_access"
	EventTypeDecision      = "decision"
	EventTypeError         = "error"
)

// Event is the canonical event structure for all agent observations.
type Event struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	SessionID string    `json:"session_id"`
	Agent     string    `json:"agent"`        // "copilot", "claude", "aichat", etc.
	EventType string    `json:"event_type"`   // input, output, execution_context, etc.
	
	// Content payload (varies by event_type)
	Content string `json:"content,omitempty"`
	
	// Typed event payloads (backward compatible)
	Input             *InputEvent            `json:"input,omitempty"`
	Output            *OutputEvent           `json:"output,omitempty"`
	ExecutionContext  *ExecutionContextEvent `json:"execution_context,omitempty"`
	FileAccess        *FileAccessEvent       `json:"file_access,omitempty"`
	CommandExecution  *CommandExecutionEvent `json:"command_execution,omitempty"`
	MCPToolCall       *MCPToolCallEvent      `json:"mcp_tool_call,omitempty"`
	ContextWindow     *ContextWindowEvent    `json:"context_window,omitempty"`
	MemoryAccess      *MemoryAccessEvent     `json:"memory_access,omitempty"`
	Decision          *DecisionEvent         `json:"decision,omitempty"`
	ErrorEvent        *ErrorEvent            `json:"error_event,omitempty"`
	
	Metadata map[string]any `json:"metadata,omitempty"`
}

// InputEvent captures agent input (prompt, command, etc.)
type InputEvent struct {
	Prompt    string `json:"prompt"`
	Length    int    `json:"length"`
	Format    string `json:"format,omitempty"` // "text", "code", "query", etc.
}

// OutputEvent captures agent output (response, generated code, etc.)
type OutputEvent struct {
	Response  string `json:"response"`
	Length    int    `json:"length"`
	ExitCode  int    `json:"exit_code,omitempty"`
}

// ExecutionContextEvent captures environment state
type ExecutionContextEvent struct {
	CWD        string            `json:"cwd,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	SessionID  string            `json:"session_id,omitempty"`
	ExecutionID string           `json:"execution_id,omitempty"`
}

// FileAccessEvent tracks file reads/writes
type FileAccessEvent struct {
	Path       string `json:"path"`
	AccessType string `json:"access_type"` // "read", "write", "delete"
	Size       int64  `json:"size,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// CommandExecutionEvent tracks shell command execution
type CommandExecutionEvent struct {
	Command    string `json:"command"`
	Args       []string `json:"args,omitempty"`
	ExitCode   int    `json:"exit_code"`
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
	DurationMs int64  `json:"duration_ms"`
}

// Legacy event types (preserved for backward compatibility)

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
