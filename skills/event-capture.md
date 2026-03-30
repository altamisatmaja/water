# Skill: Event Capture & JSONL Streaming

Panduan implementasi event capture — schema, write, read, dan streaming — di proyek Water.

---

## Event Schema (Go Structs)

```go
// internal/capture/event.go
package capture

import (
    "encoding/json"
    "time"
)

// Event types
const (
    EventTypeMCPToolCall   = "mcp_tool_call"
    EventTypeContextWindow = "context_window"
    EventTypeMemoryAccess  = "memory_access"
    EventTypeDecision      = "decision"
    EventTypeError         = "error"
)

// Event adalah canonical event dari agent
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
    PromptTokens     int     `json:"prompt_tokens"`
    CompletionTokens int     `json:"completion_tokens"`
    CachedTokens     int     `json:"cached_tokens"`
    Model            string  `json:"model"`
    Temperature      float64 `json:"temperature"`
}

type MemoryAccessEvent struct {
    ChunkID             string  `json:"chunk_id"`
    ContentPreview      string  `json:"content_preview"`
    AccessType          string  `json:"access_type"` // retrieve|update|create|delete
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
```

---

## Event Writer (JSONL)

```go
// internal/capture/writer.go
package capture

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
    "sync"
    "time"
    
    "github.com/google/uuid"
)

// Writer adalah thread-safe append-only JSONL writer
type Writer struct {
    mu   sync.Mutex
    file *os.File
    buf  *bufio.Writer
    path string
}

func NewWriter(eventsPath string) (*Writer, error) {
    f, err := os.OpenFile(eventsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
    if err != nil {
        return nil, fmt.Errorf("open events file: %w", err)
    }
    return &Writer{
        file: f,
        buf:  bufio.NewWriter(f),
        path: eventsPath,
    }, nil
}

func (w *Writer) Write(evt *Event) error {
    if evt.ID == "" {
        evt.ID = "evt-" + uuid.New().String()[:12]
    }
    if evt.Timestamp.IsZero() {
        evt.Timestamp = time.Now().UTC()
    }
    
    data, err := json.Marshal(evt)
    if err != nil {
        return fmt.Errorf("marshal event: %w", err)
    }
    
    w.mu.Lock()
    defer w.mu.Unlock()
    
    // JSONL: one JSON object per line
    if _, err := w.buf.Write(data); err != nil {
        return err
    }
    if err := w.buf.WriteByte('\n'); err != nil {
        return err
    }
    
    // Flush immediately for durability
    return w.buf.Flush()
}

func (w *Writer) Close() error {
    w.mu.Lock()
    defer w.mu.Unlock()
    w.buf.Flush()
    return w.file.Close()
}
```

---

## Event Reader (Batch + Tail)

```go
// internal/capture/reader.go
package capture

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "time"
)

// ReadAll membaca semua events dari file JSONL
func ReadAll(path string) ([]*Event, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, fmt.Errorf("open %s: %w", path, err)
    }
    defer f.Close()
    
    var events []*Event
    scanner := bufio.NewScanner(f)
    scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024) // 10MB buffer
    
    for scanner.Scan() {
        line := scanner.Bytes()
        if len(line) == 0 {
            continue
        }
        var evt Event
        if err := json.Unmarshal(line, &evt); err != nil {
            // Skip malformed lines, log warning
            continue
        }
        events = append(events, &evt)
    }
    return events, scanner.Err()
}

// Tail membaca events baru secara real-time (seperti `tail -f`)
func Tail(ctx context.Context, path string, out chan<- *Event) error {
    f, err := os.Open(path)
    if err != nil {
        return fmt.Errorf("open %s: %w", path, err)
    }
    defer f.Close()
    
    // Seek to end — only watch new events
    if _, err := f.Seek(0, io.SeekEnd); err != nil {
        return err
    }
    
    reader := bufio.NewReader(f)
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        line, err := reader.ReadBytes('\n')
        if err == io.EOF {
            time.Sleep(200 * time.Millisecond) // poll interval
            continue
        }
        if err != nil {
            return fmt.Errorf("read: %w", err)
        }
        
        line = line[:len(line)-1] // trim \n
        if len(line) == 0 {
            continue
        }
        
        var evt Event
        if err := json.Unmarshal(line, &evt); err != nil {
            continue
        }
        
        select {
        case out <- &evt:
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

---

## Contoh Penggunaan

### Write event dari HTTP handler
```go
// internal/server/handlers.go
func (s *Server) handlePostEvent(w http.ResponseWriter, r *http.Request) {
    var evt capture.Event
    if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }
    
    // Write to JSONL
    if err := s.writer.Write(&evt); err != nil {
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }
    
    // TODO: also ingest into DuckDB async
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok", "id": evt.ID})
}
```

### Tail events dan push ke WebSocket
```go
func (s *Server) streamEventsToWS(ctx context.Context, conn *websocket.Conn) {
    ch := make(chan *capture.Event, 50)
    
    go func() {
        capture.Tail(ctx, s.eventsPath, ch)
    }()
    
    for {
        select {
        case evt := <-ch:
            data, _ := json.Marshal(evt)
            conn.WriteMessage(websocket.TextMessage, data)
        case <-ctx.Done():
            return
        }
    }
}
```

---

## Sample events.jsonl (untuk testing)

```jsonl
{"id":"evt-001","timestamp":"2026-03-30T14:00:00Z","session_id":"sess-1","agent_id":"agent-1","event_type":"mcp_tool_call","mcp_tool_call":{"server_name":"github","tool_name":"search_repositories","input":{"query":"golang orm"},"output":{"results":[]},"input_tokens":100,"output_tokens":500,"execution_ms":1200,"success":true}}
{"id":"evt-002","timestamp":"2026-03-30T14:00:02Z","session_id":"sess-1","agent_id":"agent-1","event_type":"context_window","context_window":{"role":"assistant","prompt_tokens":2048,"completion_tokens":256,"cached_tokens":1024,"model":"claude-sonnet-4-6","temperature":0.7}}
{"id":"evt-003","timestamp":"2026-03-30T14:00:05Z","session_id":"sess-1","agent_id":"agent-1","event_type":"decision","decision":{"node_id":"dec-001","description":"Choose: github.search vs local cache","options":["github","cache"],"chosen":"github","reasoning":"API fresher results","confidence":0.92}}
```

---

## Tips

- JSONL (JSON Lines) = satu JSON object per baris, tidak ada trailing comma, tidak ada wrapping array
- Gunakan `bufio.Scanner` dengan buffer besar (10MB) untuk event payload yang besar
- `Tail()` menggunakan polling 200ms — cukup untuk real-time feel tanpa overhead inotify
- Event ID format: `evt-{12-char-uuid}` untuk mudah dibaca di log