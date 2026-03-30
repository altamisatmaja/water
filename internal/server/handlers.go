package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/water-viz/water/internal/capture"
	"github.com/water-viz/water/internal/graph"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleGetNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := s.graph.ListNodes(r.Context(), 200)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "list nodes"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"nodes": nodes})
}

func (s *Server) handleGetEdges(w http.ResponseWriter, r *http.Request) {
	edges, err := s.graph.ListEdges(r.Context(), 500)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "list edges"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"edges": edges})
}

func (s *Server) handleGetGraph(w http.ResponseWriter, r *http.Request) {
	g, err := s.graph.GetFullGraph(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "get graph"})
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	// Minimal stats for the vertical slice.
	nodes, err := s.graph.ListNodes(r.Context(), 1)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "stats"})
		return
	}
	_ = nodes
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handlePostEvent(w http.ResponseWriter, r *http.Request) {
	var evt capture.Event
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad json"})
		return
	}

	if err := s.writer.Write(&evt); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "write event"})
		return
	}

	// Minimal ingestion -> create a node for MCP tool call output (or the whole event).
	if evt.EventType == capture.EventTypeMCPToolCall && evt.MCPToolCall != nil {
		content := fmt.Sprintf("%s.%s output: %s", evt.MCPToolCall.ServerName, evt.MCPToolCall.ToolName, string(evt.MCPToolCall.Output))
		sum := sha256.Sum256([]byte(content))
		h := hex.EncodeToString(sum[:])

		sourceTool := evt.MCPToolCall.ServerName
		n := &graph.Node{
			NodeID:              "node-" + evt.ID,
			Content:             content,
			ContentHash:         h,
			SourceType:          "mcp_output",
			SourceTool:          &sourceTool,
			TokensIn:            evt.MCPToolCall.InputTokens,
			TokensOut:           evt.MCPToolCall.OutputTokens,
			CreatedAt:           time.Now().UTC(),
			AccessCount:         1,
			ImportanceScore:     0.5,
			RetentionConfidence: 1.0,
			Tags:                []string{"mcp", "tool_call"},
		}
		_ = s.graph.InsertNode(r.Context(), n)
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "id": evt.ID})
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Minimal dashboard until Svelte is wired/embedded.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html>
  <head>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1"/>
    <title>Water</title>
    <style>
      body { font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Arial; background:#0b1220; color:#e5e7eb; margin:0; }
      header { padding:16px 20px; border-bottom:1px solid #1f2937; display:flex; justify-content:space-between; align-items:center; }
      main { padding:20px; max-width:1000px; margin:0 auto; }
      code { background:#111827; padding:2px 6px; border-radius:6px; }
      .row { display:flex; gap:16px; flex-wrap:wrap; }
      .card { background:#0f172a; border:1px solid #1f2937; border-radius:12px; padding:14px; flex:1; min-width:280px; }
      .muted { color:#94a3b8; }
      ul { margin: 8px 0 0 20px; }
      a { color:#93c5fd; }
    </style>
  </head>
  <body>
    <header>
      <div><strong>Water</strong> <span class="muted">local agent brain viz</span></div>
      <div class="muted">API: <code>/api/graph</code> · WS: <code>/ws</code></div>
    </header>
    <main>
      <div class="row">
        <div class="card">
          <div><strong>Try it</strong></div>
          <div class="muted">Post an event, then refresh graph JSON.</div>
          <ul>
            <li><code>POST /api/events</code> (JSON body)</li>
            <li><code>GET /api/graph</code></li>
          </ul>
        </div>
        <div class="card">
          <div><strong>Next</strong></div>
          <div class="muted">Svelte dashboard will replace this static page.</div>
        </div>
      </div>
      <p class="muted" style="margin-top:18px">Health: <a href="/healthz">/healthz</a></p>
    </main>
  </body>
</html>`))
}
