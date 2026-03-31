package claude

import "time"

type ProjectRef struct {
	Key          string    `json:"key"`
	Name         string    `json:"name"`
	CWD          string    `json:"cwd,omitempty"`
	Path         string    `json:"path"`
	SessionCount int       `json:"session_count"`
	LastUpdated  time.Time `json:"last_updated"`
}

type MemoryFile struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	RelativePath string    `json:"relative_path"`
	Preview      string    `json:"preview"`
	Content      string    `json:"content,omitempty"`
	Size         int64     `json:"size"`
	ModifiedAt   time.Time `json:"modified_at"`
}

type SubagentSummary struct {
	ID              string   `json:"id"`
	Path            string   `json:"path"`
	AgentType       string   `json:"agent_type,omitempty"`
	Description     string   `json:"description,omitempty"`
	MessageCount    int      `json:"message_count"`
	ToolNames       []string `json:"tool_names,omitempty"`
	PathRefs        []string `json:"path_refs,omitempty"`
	KnowledgeFields []string `json:"knowledge_fields,omitempty"`
}

type SessionSummary struct {
	ID                    string             `json:"id"`
	Path                  string             `json:"path"`
	CWD                   string             `json:"cwd,omitempty"`
	GitBranch             string             `json:"git_branch,omitempty"`
	Version               string             `json:"version,omitempty"`
	Entrypoint            string             `json:"entrypoint,omitempty"`
	StartedAt             time.Time          `json:"started_at"`
	UpdatedAt             time.Time          `json:"updated_at"`
	PromptPreview         string             `json:"prompt_preview,omitempty"`
	MessageCount          int                `json:"message_count"`
	UserMessageCount      int                `json:"user_message_count"`
	AssistantMessageCount int                `json:"assistant_message_count"`
	ToolUseCount          int                `json:"tool_use_count"`
	InputTokens           int64              `json:"input_tokens"`
	OutputTokens          int64              `json:"output_tokens"`
	ToolNames             []string           `json:"tool_names,omitempty"`
	PathRefs              []string           `json:"path_refs,omitempty"`
	KnowledgeFields       []string           `json:"knowledge_fields,omitempty"`
	Subagents             []*SubagentSummary `json:"subagents,omitempty"`
}

type PathSummary struct {
	Path           string   `json:"path"`
	Label          string   `json:"label"`
	Kind           string   `json:"kind"`
	ReferenceCount int      `json:"reference_count"`
	Sessions       []string `json:"sessions,omitempty"`
}

type ToolSummary struct {
	Name     string   `json:"name"`
	Count    int      `json:"count"`
	Sessions []string `json:"sessions,omitempty"`
}

type KnowledgeFieldSummary struct {
	Label        string   `json:"label"`
	Description  string   `json:"description,omitempty"`
	MessageCount int      `json:"message_count"`
	Sessions     []string `json:"sessions,omitempty"`
}

type ProjectStats struct {
	Sessions        int    `json:"sessions"`
	MemoryFiles     int    `json:"memory_files"`
	Subagents       int    `json:"subagents"`
	Paths           int    `json:"paths"`
	Tools           int    `json:"tools"`
	KnowledgeFields int    `json:"knowledge_fields"`
	Messages        int    `json:"messages"`
	InputTokens     int64  `json:"input_tokens"`
	OutputTokens    int64  `json:"output_tokens"`
	LatestActivity  string `json:"latest_activity,omitempty"`
}

type GraphNode struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Kind           string `json:"kind"`
	Group          string `json:"group"`
	RefCount       int    `json:"ref_count,omitempty"`
	Size           int    `json:"size,omitempty"`
	Preview        string `json:"preview,omitempty"`
	Path           string `json:"path,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
	KnowledgeField string `json:"knowledge_field,omitempty"`
}

type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"`
	Weight int    `json:"weight,omitempty"`
}

type Graph struct {
	Nodes []*GraphNode `json:"nodes"`
	Edges []*GraphEdge `json:"edges"`
}

type ProjectSnapshot struct {
	Project  *ProjectRef              `json:"project"`
	Stats    ProjectStats             `json:"stats"`
	Sessions []*SessionSummary        `json:"sessions"`
	Memory   []*MemoryFile            `json:"memory"`
	Paths    []*PathSummary           `json:"paths"`
	Tools    []*ToolSummary           `json:"tools"`
	Fields   []*KnowledgeFieldSummary `json:"fields"`
	Graph    *Graph                   `json:"graph"`
}
