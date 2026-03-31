package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/water-viz/water/internal/capture"
	"github.com/water-viz/water/internal/claude"
	"github.com/water-viz/water/internal/config"
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
	if snapshot, ok := s.loadClaudeSnapshot(r); ok {
		writeJSON(w, http.StatusOK, snapshot.Graph)
		return
	}

	g, err := s.graph.GetFullGraph(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "get graph"})
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	if snapshot, ok := s.loadClaudeSnapshot(r); ok {
		writeJSON(w, http.StatusOK, snapshot.Stats)
		return
	}

	// Minimal stats for the vertical slice.
	nodes, err := s.graph.ListNodes(r.Context(), 1)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "stats"})
		return
	}
	_ = nodes
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	snapshot, ok := s.loadClaudeSnapshot(r)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "claude project not configured"})
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleGetProjects(w http.ResponseWriter, r *http.Request) {
	if s.claude == nil {
		writeJSON(w, http.StatusOK, map[string]any{"projects": []any{}})
		return
	}
	projects, err := s.claude.ListProjects()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	defaultProject := s.cfg.ClaudeProjectKey
	if defaultProject == "" && s.cfg.ClaudeProjectPath != "" {
		defaultProject = s.cfg.ClaudeProjectPath
	}
	if defaultProject == "" {
		if cwd, err := os.Getwd(); err == nil {
			if ref, err := s.claude.ResolveProject("", cwd); err == nil && ref != nil {
				defaultProject = ref.Key
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"projects":        projects,
		"default_project": defaultProject,
	})
}

func (s *Server) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	snapshot, ok := s.loadClaudeSnapshot(r)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "claude project not configured"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": snapshot.Sessions})
}

func (s *Server) handleGetMemo(w http.ResponseWriter, r *http.Request) {
	snapshot, ok := s.loadClaudeSnapshot(r)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "claude project not configured"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"memory": snapshot.Memory})
}

func (s *Server) loadClaudeSnapshot(r *http.Request) (*claude.ProjectSnapshot, bool) {
	if s.claude == nil {
		return nil, false
	}

	projectQuery := r.URL.Query().Get("project")
	if projectQuery == "" {
		projectQuery = fallbackConfigProject(s.cfg)
	}
	cwd, _ := os.Getwd()
	snapshot, err := s.claude.LoadProjectByQuery(projectQuery, cwd)
	if err != nil {
		return nil, false
	}
	return snapshot, true
}

func fallbackConfigProject(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	if cfg.ClaudeProjectPath != "" {
		return cfg.ClaudeProjectPath
	}
	return cfg.ClaudeProjectKey
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html>
  <head>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1"/>
    <title>Water</title>
    <link rel="icon" type="image/svg+xml" href="https://raw.githubusercontent.com/altamisatmaja/water/main/assets/favicon.svg"/>
    <style>
      :root {
        --bg: #000000;
        --bg-2: #000000;
        --card: rgba(8, 20, 27, 0.78);
        --line: rgba(150, 226, 255, 0.18);
        --text: #eef7fb;
        --muted: #8fb6c6;
        --aqua: #7ae7ff;
        --sea: #3fc2d8;
        --sand: #f4d28d;
        --rose: #ff8f78;
        --mint: #87f1c8;
        --field: #ffd36e;
      }
      * { box-sizing: border-box; }
      body {
        margin: 0;
        color: var(--text);
        font-family: "Avenir Next", "Segoe UI", sans-serif;
        background: #000000;
        min-height: 100vh;
      }
      .shell {
        width: 100%;
        min-height: 100vh;
        padding: 18px;
      }
      .layout {
        display: grid;
        grid-template-columns: minmax(280px, 0.85fr) minmax(0, 2.75fr);
        gap: 18px;
        min-height: calc(100vh - 36px);
        align-items: stretch;
      }
      .sidebar {
        display: grid;
        gap: 18px;
        align-content: start;
      }
      .sidebar-sticky {
        position: sticky;
        top: 18px;
        display: grid;
        gap: 18px;
      }
      .title {
        display: block;
        width: 220px;
        height: auto;
        margin: 0;
      }
      .title img {
        display: block;
        width: 100%;
        height: auto;
        filter: brightness(0) invert(1);
      }
      .subtitle {
        color: var(--muted);
        margin-top: 10px;
        font-size: 14px;
        line-height: 1.5;
      }
      .controls {
        display: grid;
        gap: 12px;
        align-items: start;
        background: var(--card);
        border: 1px solid var(--line);
        padding: 14px;
        border-radius: 18px;
        backdrop-filter: blur(18px);
      }
      .controls-row {
        display: flex;
        gap: 10px;
        align-items: center;
        width: 100%;
      }
      .controls-row select {
        flex: 1;
        width: 100%;
      }
      select, button {
        background: rgba(255,255,255,0.06);
        color: var(--text);
        border: 1px solid rgba(255,255,255,0.08);
        border-radius: 12px;
        padding: 10px 12px;
      }
      button { cursor: pointer; }
      .panel {
        background: var(--card);
        border: 1px solid var(--line);
        border-radius: 28px;
        padding: 20px;
        backdrop-filter: blur(18px);
        box-shadow: 0 20px 60px rgba(0,0,0,0.22);
      }
      .panel h2 { margin: 0 0 8px; font-size: 18px; }
      .graph-panel {
        padding: 22px;
        min-height: 100%;
        display: flex;
        flex-direction: column;
      }
      .panel-head {
        display: flex;
        justify-content: space-between;
        align-items: end;
        gap: 18px;
        margin-bottom: 14px;
      }
      .panel-copy p {
        margin: 0;
        max-width: 760px;
        color: var(--muted);
        line-height: 1.5;
      }
      .muted { color: var(--muted); }
      .graph-stats {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(120px, max-content));
        gap: 12px;
        width: min(820px, calc(100% - 420px));
        max-width: 100%;
      }
      .stat {
        padding: 14px 16px;
        border-radius: 16px;
        background:
          linear-gradient(180deg, rgba(255,255,255,0.04), rgba(255,255,255,0.02)),
          rgba(255,255,255,0.02);
        border: 1px solid rgba(255,255,255,0.07);
        backdrop-filter: blur(10px);
        min-width: 0;
      }
      .stat span,
      .stat strong {
        overflow-wrap: anywhere;
        word-break: break-word;
      }
      .stat strong { display: block; font-size: 22px; margin-top: 4px; }
      .canvas-wrap {
        position: relative;
        min-height: 0;
        height: 100%;
        flex: 1;
        border-radius: 30px;
        overflow: hidden;
        border: 1px solid rgba(255,255,255,0.08);
        background:
          radial-gradient(circle at 50% 42%, rgba(122, 231, 255, 0.12), transparent 22%),
          radial-gradient(circle at 16% 18%, rgba(244, 210, 141, 0.14), transparent 20%),
          radial-gradient(circle at 78% 22%, rgba(154, 176, 255, 0.10), transparent 18%),
          linear-gradient(180deg, rgba(255,255,255,0.03), rgba(2,8,12,0.34));
      }
      .canvas-wrap::before {
        content: "";
        position: absolute;
        inset: 0;
        background:
          linear-gradient(rgba(255,255,255,0.03) 1px, transparent 1px),
          linear-gradient(90deg, rgba(255,255,255,0.03) 1px, transparent 1px);
        background-size: 42px 42px;
        opacity: 0.18;
        pointer-events: none;
      }
      .canvas-wrap::after {
        content: "";
        position: absolute;
        inset: auto 0 0 0;
        height: 38%;
        background: linear-gradient(180deg, transparent, rgba(0, 0, 0, 0.34));
        pointer-events: none;
      }
      #graph {
        width: 100%;
        height: 100%;
        display: block;
      }
      .graph-hud {
        position: absolute;
        inset: 20px 20px auto auto;
        z-index: 2;
        display: grid;
        gap: 10px;
        width: min(360px, calc(100% - 40px));
        max-width: calc(100% - 40px);
      }
      .graph-top {
        position: absolute;
        left: 20px;
        top: 20px;
        z-index: 2;
        display: grid;
        gap: 12px;
        width: calc(100% - 400px);
        max-width: 840px;
      }
      .graph-actions {
        display: grid;
        grid-template-columns: repeat(2, max-content);
        gap: 8px;
        align-items: stretch;
        justify-content: start;
      }
      .graph-action {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        padding: 10px 12px;
        border-radius: 999px;
        background: rgba(4, 12, 18, 0.72);
        border: 1px solid rgba(255,255,255,0.08);
        backdrop-filter: blur(10px);
        color: var(--text);
        font-size: 12px;
        cursor: pointer;
        width: fit-content;
        max-width: 100%;
      }
      .graph-action strong {
        color: var(--muted);
        font-size: 11px;
        letter-spacing: 0.08em;
        text-transform: uppercase;
      }
      .graph-action.active {
        border-color: rgba(122,231,255,0.34);
        background: rgba(122,231,255,0.12);
      }
      .graph-overview {
        position: absolute;
        left: 20px;
        bottom: 20px;
        z-index: 2;
        max-width: min(420px, calc(100% - 40px));
        padding: 14px 16px;
        border-radius: 18px;
        background: rgba(4, 12, 18, 0.68);
        border: 1px solid rgba(255,255,255,0.08);
        backdrop-filter: blur(12px);
        overflow-wrap: anywhere;
      }
      .graph-overview strong {
        display: block;
        margin-bottom: 6px;
      }
      .detail-card {
        position: absolute;
        z-index: 3;
        width: clamp(240px, 28vw, 420px);
        max-width: calc(100% - 40px);
        min-height: 0;
        max-height: min(42vh, 420px);
        padding: 14px 16px;
        border-radius: 18px;
        border: 1px solid rgba(255,255,255,0.10);
        background: rgba(4, 12, 18, 0.82);
        backdrop-filter: blur(16px);
        box-shadow: 0 18px 48px rgba(0,0,0,0.28);
        white-space: pre-wrap;
        line-height: 1.45;
        font-size: 13px;
        transform: translate(-50%, calc(-100% - 22px));
        overflow: auto;
        overflow-wrap: anywhere;
        scrollbar-width: none;
        -ms-overflow-style: none;
        pointer-events: auto;
        opacity: 0;
        transition: opacity 140ms ease;
      }
      .detail-card::-webkit-scrollbar {
        width: 0;
        height: 0;
      }
      .detail-card.visible {
        opacity: 1;
      }
      .graph-badge, .graph-tip {
        padding: 12px 14px;
        border-radius: 16px;
        background: rgba(4, 12, 18, 0.72);
        border: 1px solid rgba(255,255,255,0.08);
        backdrop-filter: blur(10px);
        overflow-wrap: anywhere;
      }
      .graph-badge strong {
        display: block;
        font-size: 11px;
        letter-spacing: 0.16em;
        text-transform: uppercase;
        color: var(--muted);
        margin-bottom: 6px;
      }
      .legend {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
      }
      .legend span {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        font-size: 12px;
        color: var(--text);
      }
      .legend i {
        width: 9px;
        height: 9px;
        border-radius: 999px;
        display: inline-block;
      }
      .list {
        display: grid;
        gap: 10px;
        max-height: 36vh;
        overflow: auto;
        padding-right: 4px;
        scrollbar-width: none;
        -ms-overflow-style: none;
      }
      .list::-webkit-scrollbar {
        width: 0;
        height: 0;
      }
      .item {
        border-radius: 16px;
        border: 1px solid rgba(255,255,255,0.06);
        padding: 12px 14px;
        background: rgba(255,255,255,0.03);
      }
      .item strong { display: block; margin-bottom: 6px; }
      .meta { color: var(--muted); font-size: 12px; }
      .chips {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        margin: 10px 0 0;
      }
      .chip {
        display: inline-flex;
        align-items: center;
        border-radius: 999px;
        padding: 4px 9px;
        font-size: 11px;
        letter-spacing: 0.04em;
        background: rgba(255, 211, 110, 0.12);
        border: 1px solid rgba(255, 211, 110, 0.24);
        color: #ffe4a0;
      }
      .history-list {
        position: relative;
        padding-left: 22px;
      }
      .history-list::before {
        content: "";
        position: absolute;
        left: 9px;
        top: 4px;
        bottom: 4px;
        width: 2px;
        background: linear-gradient(180deg, rgba(122,231,255,0.35), rgba(255,143,120,0.18));
      }
      .history-item {
        position: relative;
        cursor: pointer;
        transition: transform 140ms ease, border-color 140ms ease, background 140ms ease;
      }
      .history-item::before {
        content: "";
        position: absolute;
        left: -18px;
        top: 18px;
        width: 10px;
        height: 10px;
        border-radius: 999px;
        background: var(--sea);
        box-shadow: 0 0 0 4px rgba(63,194,216,0.14);
      }
      .history-item:hover,
      .history-item.active {
        transform: translateX(4px);
        border-color: rgba(122,231,255,0.28);
        background: rgba(122,231,255,0.08);
      }
      .history-item.active::before {
        background: var(--sand);
        box-shadow: 0 0 0 5px rgba(244,210,141,0.16);
      }
      .eyebrow {
        display: inline-flex;
        align-items: center;
        gap: 8px;
        font-size: 11px;
        text-transform: uppercase;
        letter-spacing: 0.18em;
        color: var(--muted);
        margin-bottom: 10px;
      }
      .memo-grid {
        grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
        max-height: 300px;
      }
      .sidebar-panel .list {
        max-height: 34vh;
      }
      @media (max-width: 1100px) {
        .shell {
          padding: 12px;
        }
        .layout, .graph-stats { grid-template-columns: 1fr; }
        .layout {
          gap: 12px;
          min-height: auto;
        }
        .panel,
        .graph-panel,
        .controls {
          padding: 16px;
          border-radius: 22px;
        }
        .title {
          width: 168px;
        }
        .controls-row {
          flex-direction: column;
          align-items: stretch;
        }
        .controls-row button,
        .controls-row select {
          width: 100%;
        }
        .panel-head { flex-direction: column; align-items: start; }
        .canvas-wrap, #graph { min-height: 560px; height: 560px; }
        .graph-top {
          left: 12px;
          top: 12px;
          width: calc(100% - 24px);
          max-width: none;
        }
        .graph-stats {
          grid-template-columns: repeat(2, minmax(0, 1fr));
          gap: 8px;
        }
        .stat {
          padding: 12px;
        }
        .stat strong {
          font-size: 18px;
        }
        .graph-hud {
          inset: auto 12px 12px 12px;
          width: auto;
          max-width: none;
          display: flex;
          flex-direction: column;
        }
        .graph-overview {
          display: none;
        }
        .detail-card {
          width: min(320px, calc(100% - 24px));
          max-height: 34vh;
        }
        .sidebar-sticky {
          position: static;
        }
        .sidebar-panel .list,
        .list {
          max-height: 32vh;
        }
      }
      @media (max-width: 640px) {
        .canvas-wrap,
        #graph {
          min-height: 500px;
          height: 500px;
        }
        .graph-top {
          left: 10px;
          top: 10px;
          width: calc(100% - 20px);
          gap: 6px;
        }
        .graph-stats {
          grid-template-columns: 1fr;
          gap: 6px;
          width: 100%;
        }
        .stat {
          padding: 10px 12px;
        }
        .stat span {
          font-size: 11px;
        }
        .stat strong {
          font-size: 16px;
        }
        .graph-actions {
          grid-template-columns: 1fr;
          gap: 6px;
          width: 100%;
        }
        .graph-action {
          font-size: 11px;
          padding: 8px 10px;
          width: 100%;
          justify-content: space-between;
        }
        .graph-hud {
          left: 10px;
          right: 10px;
          bottom: 10px;
          gap: 6px;
        }
        .graph-badge,
        .graph-tip {
          padding: 10px 12px;
          border-radius: 14px;
        }
        .legend {
          gap: 6px;
        }
        .legend span {
          font-size: 11px;
        }
        .detail-card {
          width: calc(100% - 20px);
          max-width: calc(100% - 20px);
          max-height: 32vh;
          padding: 12px;
          font-size: 12px;
        }
      }
      @media (min-width: 1101px) {
        .graph-panel {
          min-height: 100vh;
        }
        .canvas-wrap,
        #graph {
          min-height: calc(100vh - 280px);
        }
      }
    </style>
  </head>
  <body>
    <div class="shell">
      <div class="layout">
        <aside class="sidebar">
          <div class="sidebar-sticky">
              <div>
                <h1 class="title"><img src="https://raw.githubusercontent.com/altamisatmaja/water/main/assets/water-icon.svg" alt="Water"></h1>
              </div>
            <section class="controls">
              <label for="project-select" class="muted">Project</label>
              <div class="controls-row">
                <select id="project-select"></select>
                <button id="reload-btn">Reload</button>
              </div>
            </section>
            <section class="panel sidebar-panel">
              <h2>History</h2>
              <div class="eyebrow">Session Timeline</div>
              <div id="history" class="list history-list"></div>
            </section>
            <section class="panel sidebar-panel">
              <h2>Knowledge Fields</h2>
              <div class="eyebrow">Classified</div>
              <div id="fields" class="list"></div>
            </section>
          </div>
        </aside>

        <section class="panel graph-panel">
        <div class="panel-head">
          <div class="panel-copy">
            <h2>3D Knowledge Network</h2>
            <p id="project-meta"></p>
          </div>
        </div>
        <div class="canvas-wrap">
          <canvas id="graph"></canvas>
          <div class="graph-top">
            <div id="stats" class="graph-stats"></div>
            <div class="graph-actions">
              <button type="button" id="pan-mode-btn" class="graph-action"><strong>Pan</strong><span>Drag empty space</span></button>
              <button type="button" id="rotate-mode-btn" class="graph-action"><strong>Rotate</strong><span>Left-drag view</span></button>
              <button type="button" id="zoom-in-btn" class="graph-action"><strong>Zoom In</strong><span>Closer</span></button>
              <button type="button" id="zoom-out-btn" class="graph-action"><strong>Zoom Out</strong><span>Farther</span></button>
            </div>
          </div>
          <div id="detail-card" class="detail-card muted">Select a node to inspect its linked session, knowledge field, tool, memory note, or file path.</div>
          <div class="graph-hud">
            <div class="graph-badge">
              <strong>Graph Layers</strong>
              <div class="legend">
                <span><i style="background:#7ae7ff"></i>Project</span>
                <span><i style="background:#3fc2d8"></i>Sessions</span>
                <span><i style="background:#f4d28d"></i>Memory</span>
                <span><i style="background:#ffd36e"></i>Fields</span>
                <span><i style="background:#87f1c8"></i>Tools</span>
                <span><i style="background:#ff8f78"></i>Paths</span>
                <span><i style="background:#9ab0ff"></i>Subagents</span>
              </div>
            </div>
            <div class="graph-tip muted">Use the controls below the summary cards to switch between pan and rotate. Scroll or use the zoom buttons to move in and out.</div>
          </div>
          <div class="graph-overview muted"><strong>Network View</strong>The node layout is spread out like a 3D network map, making relationships between sessions, tools, fields, memory, and files easier to read spatially.</div>
        </div>
        </section>
      </div>
    </div>

    <script type="importmap">
      {
        "imports": {
          "three": "https://unpkg.com/three@0.179.1/build/three.module.js"
        }
      }
    </script>
    <script type="module">
      import * as THREE from 'https://unpkg.com/three@0.179.1/build/three.module.js'
      import { OrbitControls } from 'https://unpkg.com/three@0.179.1/examples/jsm/controls/OrbitControls.js'

      const state = {
        projects: [],
        selectedProject: '',
        snapshot: null,
        selectedNodeID: '',
        selectedSessionID: '',
      }

      const kindColors = {
        project: '#7ae7ff',
        session: '#3fc2d8',
        memory: '#f4d28d',
        tool: '#87f1c8',
        field: '#ffd36e',
        file: '#ff8f78',
        dir: '#ff8f78',
        path: '#ff8f78',
        subagent: '#9ab0ff',
      }

      const three = {
        scene: null,
        camera: null,
        renderer: null,
        controls: null,
        raycaster: new THREE.Raycaster(),
        pointer: new THREE.Vector2(),
        dragPlane: new THREE.Plane(),
        dragOffset: new THREE.Vector3(),
        dragPoint: new THREE.Vector3(),
        dragStartPosition: new THREE.Vector3(),
        graphGroup: null,
        backdrop: null,
        hoveredNodeID: '',
        nodeMeshes: new Map(),
        labelByNodeID: new Map(),
        edgeObjects: [],
        labelSprites: [],
        positions: {},
        animationFrame: 0,
        renderQueued: false,
        draggedNodeID: '',
        pointerDownNodeID: '',
        dragMoved: false,
        hoverTiltX: 0,
        hoverTiltY: 0,
      }

      const detailState = {
        text: 'Select a node to inspect its linked session, knowledge field, tool, memory note, or file path.',
        visible: false,
      }

      const uiState = {
        interactionMode: '',
      }

      async function fetchJSON(url) {
        const res = await fetch(url)
        if (!res.ok) throw new Error('HTTP ' + res.status)
        return res.json()
      }

      async function loadProjects() {
        const data = await fetchJSON('/api/projects')
        state.projects = data.projects || []
        const select = document.getElementById('project-select')
        const current = state.selectedProject || data.default_project || state.projects[0]?.key || ''
        state.selectedProject = current
        select.innerHTML = state.projects.map(project => {
          const selected = project.key === current ? 'selected' : ''
          const label = projectDisplayLabel(project)
          return '<option value="' + project.key + '" ' + selected + '>' + escapeHTML(label) + '</option>'
        }).join('')
      }

      async function loadSnapshot() {
        if (!state.selectedProject) return
        const query = '?project=' + encodeURIComponent(state.selectedProject)
        state.snapshot = await fetchJSON('/api/project' + query)
        render()
      }

      function render() {
        renderStats()
        renderMeta()
        renderGraph()
        renderHistory()
        renderFields()
      }

      function renderStats() {
        const stats = state.snapshot?.stats || {}
        const cards = [
          ['Sessions', stats.sessions || 0],
          ['Agents', stats.subagents || 0],
          ['Fields', stats.knowledge_fields || 0],
          ['Tools', stats.tools || 0],
          ['Input Tokens', formatCompactNumber(stats.input_tokens || 0)],
          ['Output Tokens', formatCompactNumber(stats.output_tokens || 0)],
        ]
        document.getElementById('stats').innerHTML = cards.map(([label, value]) =>
          '<div class="stat"><span class="muted">' + label + '</span><strong>' + value + '</strong></div>'
        ).join('')
      }

      function renderMeta() {
        const project = state.snapshot?.project
        if (!project) return
        const stats = state.snapshot?.stats || {}
        const parts = [
          projectDisplayLabel(project),
          (stats.sessions || 0) + ' sessions',
          (stats.knowledge_fields || 0) + ' fields',
          (stats.paths || 0) + ' touched paths',
        ]
        document.getElementById('project-meta').textContent = parts.join('  •  ')
      }

      function renderGraph() {
        const graph = state.snapshot?.graph
        if (!graph) {
          clearScene()
          return
        }
        initThree()
        buildSceneGraph(graph)
        syncGraphSelection()
      }

      function buildPositions(nodes) {
        const groups = {
          project: nodes.filter(node => node.group === 'project'),
          memory: nodes.filter(node => node.group === 'memory'),
          session: nodes.filter(node => node.group === 'session'),
          field: nodes.filter(node => node.group === 'field'),
          tool: nodes.filter(node => node.group === 'tool'),
          path: nodes.filter(node => node.group === 'path'),
          subagent: nodes.filter(node => node.group === 'subagent'),
        }

        const pos = {}
        if (groups.project[0]) {
          pos[groups.project[0].id] = { x: 0, y: 120, z: 0 }
        }

        placeCluster(groups.field, pos, { x: 0, y: 280, z: -110 }, 240, 90)
        placeCluster(groups.memory, pos, { x: -300, y: 120, z: 160 }, 250, 80)
        placeNetworkRing(groups.session, pos, { x: 0, y: 0, z: 0 }, 360, 150)
        placeCluster(groups.tool, pos, { x: -430, y: -30, z: -170 }, 220, 68)
        placeCluster(groups.path, pos, { x: 430, y: -10, z: 150 }, 320, 58)
        placeCluster(groups.subagent, pos, { x: 0, y: -240, z: -250 }, 260, 76)
        return pos
      }

      function placeCluster(nodes, positions, center, radius, lift) {
        if (!nodes.length) return
        nodes.forEach((node, index) => {
          const angle = index * 2.399963229728653
          const spread = radius * Math.sqrt((index + 0.5) / nodes.length)
          positions[node.id] = {
            x: center.x + Math.cos(angle) * spread,
            y: center.y + Math.sin(index * 0.7) * lift,
            z: center.z + Math.sin(angle) * spread,
          }
        })
      }

      function placeNetworkRing(nodes, positions, center, radius, lift) {
        if (!nodes.length) return
        nodes.forEach((node, index) => {
          const angle = (-Math.PI / 2) + ((Math.PI * 2) * index / Math.max(nodes.length, 1))
          const wave = 0.72 + (Math.sin(index * 1.37) * 0.18)
          positions[node.id] = {
            x: center.x + Math.cos(angle) * radius * wave,
            y: center.y + Math.sin(index * 0.9) * lift,
            z: center.z + Math.sin(angle) * radius * (1.08 + Math.cos(index * 0.6) * 0.16),
          }
        })
      }

      function renderHistory() {
        const sessions = state.snapshot?.sessions || []
        document.getElementById('history').innerHTML = sessions.map(session => {
          const preview = escapeHTML(session.prompt_preview || '(no prompt preview)')
          const tools = (session.tool_names || []).join(', ')
          const fields = renderChips(session.knowledge_fields || [])
          const active = state.selectedSessionID === session.id ? ' active' : ''
          return [
            '<article class="item history-item' + active + '" data-session="' + encodeURIComponent(session.id) + '">',
            '<strong>' + session.id + '</strong>',
            '<div class="meta">' + formatDate(session.updated_at) + '  •  ' + (session.git_branch || 'no branch') + '</div>',
            '<div>' + preview + '</div>',
            fields,
            '<div class="meta" style="margin-top:8px">tools: ' + escapeHTML(tools || 'none') + '  •  subagents: ' + (session.subagents || []).length + '</div>',
            '</article>'
          ].join('')
        }).join('')
        document.querySelectorAll('.history-item').forEach(item => {
          item.addEventListener('click', () => {
            const sessionID = decodeURIComponent(item.dataset.session || '')
            selectSession(sessionID)
          })
        })
      }

      function renderFields() {
        const fields = state.snapshot?.fields || []
        document.getElementById('fields').innerHTML = fields.length
          ? fields.map(field => [
              '<article class="item">',
              '<strong>' + escapeHTML(field.label) + '</strong>',
              '<div class="meta">' + (field.message_count || 0) + ' messages  •  ' + (field.sessions || []).length + ' sessions</div>',
              '<div>' + escapeHTML(field.description || '') + '</div>',
              renderChips(field.sessions || [], 'session'),
              '</article>',
            ].join('')).join('')
          : '<div class="item"><strong>No classifications yet</strong><div class="meta">Session prompts and agent replies have not been bucketed into knowledge fields.</div></div>'
      }

      function showDetail(nodeID) {
        const graphNode = (state.snapshot?.graph?.nodes || []).find(node => node.id === nodeID)
        if (!graphNode) return
        state.selectedNodeID = nodeID
        state.selectedSessionID = graphNode.session_id || (graphNode.kind === 'session' ? graphNode.id : state.selectedSessionID)

        const lines = [
          graphNode.label,
          '',
          'kind: ' + graphNode.kind,
        ]
        if (graphNode.knowledge_field) lines.push('field: ' + graphNode.knowledge_field)
        if (graphNode.path) lines.push('path: ' + graphNode.path)
        if (graphNode.session_id) lines.push('session: ' + graphNode.session_id)
        if (graphNode.ref_count) lines.push('refs: ' + graphNode.ref_count)
        const session = graphNode.session_id
          ? (state.snapshot?.sessions || []).find(item => item.id === graphNode.session_id)
          : graphNode.kind === 'session'
            ? (state.snapshot?.sessions || []).find(item => ('session:' + item.id) === graphNode.id)
            : null
        if (session?.knowledge_fields?.length) lines.push('fields: ' + session.knowledge_fields.join(', '))
        if (graphNode.preview) lines.push('', graphNode.preview)
        detailState.text = lines.join('\n')
        detailState.visible = true
        updateDetailCardPosition()
        syncGraphSelection()
        renderHistory()
      }

      function selectSession(sessionID) {
        state.selectedSessionID = sessionID
        const sessionNode = (state.snapshot?.graph?.nodes || []).find(node => node.kind === 'session' && node.session_id === sessionID)
        if (sessionNode) {
          showDetail(sessionNode.id)
          focusNode(sessionNode.id)
          return
        }
        syncGraphSelection()
        renderHistory()
      }

      function initThree() {
        if (three.renderer) {
          resizeRenderer()
          return
        }

        const canvas = document.getElementById('graph')
        const wrap = canvas.parentElement
        three.scene = new THREE.Scene()
        three.scene.background = new THREE.Color(0x08141b)
        three.scene.fog = new THREE.FogExp2(0x08141b, 0.0009)

        three.camera = new THREE.PerspectiveCamera(48, 1, 1, 4000)
        three.camera.position.set(0, 140, 980)

        three.renderer = new THREE.WebGLRenderer({
          canvas,
          antialias: false,
          alpha: true,
          powerPreference: 'low-power',
        })
        three.renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2))

        const ambient = new THREE.AmbientLight(0xd9f7ff, 0.72)
        const key = new THREE.PointLight(0x7ae7ff, 1.3, 1800)
        key.position.set(0, 260, 280)
        const rim = new THREE.PointLight(0xffb07c, 0.9, 1800)
        rim.position.set(320, -180, 180)
        three.scene.add(ambient, key, rim)

        const stars = []
        for (let i = 0; i < 420; i++) {
          stars.push(
            (Math.random() - 0.5) * 2200,
            (Math.random() - 0.25) * 1400,
            (Math.random() - 0.5) * 2200,
          )
        }
        const starGeometry = new THREE.BufferGeometry()
        starGeometry.setAttribute('position', new THREE.Float32BufferAttribute(stars, 3))
        const starMaterial = new THREE.PointsMaterial({
          color: 0x9edfff,
          size: 3,
          sizeAttenuation: true,
          transparent: true,
          opacity: 0.22,
          depthWrite: false,
        })
        three.backdrop = new THREE.Points(starGeometry, starMaterial)
        three.scene.add(three.backdrop)

        three.graphGroup = new THREE.Group()
        three.scene.add(three.graphGroup)

        three.controls = new OrbitControls(three.camera, canvas)
        three.controls.enableDamping = false
        three.controls.enablePan = true
        three.controls.screenSpacePanning = true
        three.controls.minDistance = 260
        three.controls.maxDistance = 1800
        three.controls.target.set(0, 40, 0)
        three.controls.mouseButtons = {
          LEFT: THREE.MOUSE.PAN,
          MIDDLE: THREE.MOUSE.DOLLY,
          RIGHT: THREE.MOUSE.ROTATE,
        }
        three.controls.addEventListener('change', requestRender)
        applyInteractionMode()

        canvas.addEventListener('pointermove', onPointerMove)
        canvas.addEventListener('pointerdown', onPointerDown)
        canvas.addEventListener('pointerup', onPointerUp)
        canvas.addEventListener('pointerleave', onPointerUp)
        canvas.addEventListener('click', onCanvasClick)
        window.addEventListener('resize', resizeRenderer)
        resizeRenderer()
        requestRender()
      }

      function resizeRenderer() {
        if (!three.renderer) return
        const canvas = document.getElementById('graph')
        const wrap = canvas.parentElement
        const width = wrap.clientWidth
        const height = Math.max(520, wrap.clientHeight)
        three.renderer.setSize(width, height, false)
        three.camera.aspect = width / height
        three.camera.updateProjectionMatrix()
        requestRender()
      }

      function requestRender() {
        if (!three.renderer) return
        if (three.renderQueued) return
        three.renderQueued = true
        three.animationFrame = requestAnimationFrame(renderScene)
      }

      function applyInteractionMode() {
        if (!three.controls) return
        three.controls.enablePan = uiState.interactionMode === 'pan'
        three.controls.mouseButtons.LEFT =
          uiState.interactionMode === 'rotate'
            ? THREE.MOUSE.ROTATE
            : uiState.interactionMode === 'pan'
              ? THREE.MOUSE.PAN
              : -1
        three.controls.mouseButtons.RIGHT = uiState.interactionMode === 'rotate' ? THREE.MOUSE.ROTATE : -1
        document.getElementById('pan-mode-btn')?.classList.toggle('active', uiState.interactionMode === 'pan')
        document.getElementById('rotate-mode-btn')?.classList.toggle('active', uiState.interactionMode === 'rotate')
      }

      function setInteractionMode(mode) {
        uiState.interactionMode = uiState.interactionMode === mode ? '' : mode
        applyInteractionMode()
        requestRender()
      }

      function zoomGraph(direction) {
        if (!three.camera || !three.controls) return
        const offset = three.camera.position.clone().sub(three.controls.target)
        const factor = direction > 0 ? 0.84 : 1.18
        offset.multiplyScalar(factor)
        const distance = offset.length()
        if (distance < three.controls.minDistance) {
          offset.setLength(three.controls.minDistance)
        }
        if (distance > three.controls.maxDistance) {
          offset.setLength(three.controls.maxDistance)
        }
        three.camera.position.copy(three.controls.target.clone().add(offset))
        three.controls.update()
        requestRender()
      }

      function renderScene() {
        three.renderQueued = false
        if (!three.renderer) return
        if (three.graphGroup) {
          three.graphGroup.rotation.x = three.hoverTiltX
          three.graphGroup.rotation.y = three.hoverTiltY
        }
        updateLabels()
        updateDetailCardPosition()
        three.renderer.render(three.scene, three.camera)
      }

      function clearScene() {
        if (!three.graphGroup) return
        while (three.graphGroup.children.length) {
          const child = three.graphGroup.children[0]
          three.graphGroup.remove(child)
          disposeObject(child)
        }
        three.nodeMeshes = new Map()
        three.labelByNodeID = new Map()
        three.edgeObjects = []
        three.labelSprites = []
        three.draggedNodeID = ''
        three.pointerDownNodeID = ''
        three.dragMoved = false
        if (three.animationFrame) {
          cancelAnimationFrame(three.animationFrame)
          three.animationFrame = 0
        }
        three.renderQueued = false
        detailState.visible = false
        updateDetailCardPosition()
      }

      function disposeObject(object) {
        if (!object) return
        if (object.geometry) object.geometry.dispose()
        if (object.material) {
          if (Array.isArray(object.material)) {
            object.material.forEach(mat => mat.dispose())
          } else {
            object.material.dispose()
          }
        }
      }

      function buildSceneGraph(graph) {
        clearScene()
        const positions = buildPositions(graph.nodes || [])
        three.positions = positions

        ;(graph.edges || []).forEach(edge => {
          const from = positions[edge.source]
          const to = positions[edge.target]
          if (!from || !to) return
          const curve = new THREE.QuadraticBezierCurve3(
            new THREE.Vector3(from.x, from.y, from.z),
            new THREE.Vector3((from.x + to.x) / 2, Math.max(from.y, to.y) + 70, (from.z + to.z) / 2),
            new THREE.Vector3(to.x, to.y, to.z),
          )
          const points = curve.getPoints(24)
          const geometry = new THREE.BufferGeometry().setFromPoints(points)
          const material = new THREE.LineBasicMaterial({ color: 0x8fb6c6, transparent: true, opacity: 0.22 })
          const line = new THREE.Line(geometry, material)
          line.userData = { source: edge.source, target: edge.target }
          three.graphGroup.add(line)
          three.edgeObjects.push(line)
        })

        ;(graph.nodes || []).forEach(node => {
          const pos = positions[node.id]
          if (!pos) return
          const radius = Math.max(7, Math.min(18, 5 + (node.size || 8)))
          const geometry = new THREE.SphereGeometry(radius, 24, 24)
          const material = new THREE.MeshStandardMaterial({
            color: kindColors[node.kind] || '#ffffff',
            emissive: kindColors[node.kind] || '#ffffff',
            emissiveIntensity: 0.18,
            roughness: 0.18,
            metalness: 0.28,
          })
          const mesh = new THREE.Mesh(geometry, material)
          mesh.position.set(pos.x, pos.y, pos.z)
          mesh.userData = {
            nodeID: node.id,
            sessionID: node.session_id || (node.kind === 'session' ? node.id : ''),
            radius,
          }
          three.graphGroup.add(mesh)
          three.nodeMeshes.set(node.id, mesh)

          const sprite = makeLabelSprite((node.label || node.id).slice(0, 18), kindColors[node.kind] || '#ffffff')
          sprite.position.set(pos.x, pos.y + radius + 18, pos.z)
          sprite.userData = { nodeID: node.id }
          three.graphGroup.add(sprite)
          three.labelSprites.push(sprite)
          three.labelByNodeID.set(node.id, sprite)
        })
        requestRender()
      }

      function makeLabelSprite(text, color) {
        const canvas = document.createElement('canvas')
        canvas.width = 256
        canvas.height = 80
        const context = canvas.getContext('2d')
        context.fillStyle = 'rgba(4, 12, 18, 0.80)'
        context.strokeStyle = color
        context.lineWidth = 2
        roundRect(context, 4, 6, 248, 68, 18)
        context.fill()
        context.stroke()
        context.fillStyle = '#eef7fb'
        context.font = '600 24px Avenir Next'
        context.textAlign = 'center'
        context.textBaseline = 'middle'
        context.fillText(text, 128, 40)
        const texture = new THREE.CanvasTexture(canvas)
        texture.needsUpdate = true
        const material = new THREE.SpriteMaterial({ map: texture, transparent: true, depthWrite: false })
        const sprite = new THREE.Sprite(material)
        sprite.scale.set(90, 28, 1)
        return sprite
      }

      function roundRect(context, x, y, width, height, radius) {
        context.beginPath()
        context.moveTo(x + radius, y)
        context.lineTo(x + width - radius, y)
        context.quadraticCurveTo(x + width, y, x + width, y + radius)
        context.lineTo(x + width, y + height - radius)
        context.quadraticCurveTo(x + width, y + height, x + width - radius, y + height)
        context.lineTo(x + radius, y + height)
        context.quadraticCurveTo(x, y + height, x, y + height - radius)
        context.lineTo(x, y + radius)
        context.quadraticCurveTo(x, y, x + radius, y)
        context.closePath()
      }

      function updateLabels() {
        if (!three.labelSprites.length) return
        three.labelSprites.forEach(sprite => {
          sprite.quaternion.copy(three.camera.quaternion)
          const nodeID = sprite.userData.nodeID
          const visible = nodeID === state.selectedNodeID || nodeID === three.hoveredNodeID
          sprite.material.opacity = visible ? 0.98 : 0.18
        })
      }

      function updateDetailCardPosition() {
        const card = document.getElementById('detail-card')
        if (!card) return
        card.textContent = detailState.text
        if (!detailState.visible || !state.selectedNodeID || !three.renderer) {
          card.classList.remove('visible')
          card.style.left = '20px'
          card.style.top = '20px'
          return
        }
        const mesh = three.nodeMeshes.get(state.selectedNodeID)
        if (!mesh) {
          card.classList.remove('visible')
          return
        }
        const projected = mesh.position.clone().project(three.camera)
        const canvas = three.renderer.domElement
        const wrap = canvas.parentElement
        const x = ((projected.x + 1) / 2) * wrap.clientWidth
        const y = ((-projected.y + 1) / 2) * wrap.clientHeight
        const clampedX = Math.max(180, Math.min(wrap.clientWidth - 180, x))
        const clampedY = Math.max(90, Math.min(wrap.clientHeight - 32, y))
        card.style.left = clampedX + 'px'
        card.style.top = clampedY + 'px'
        card.classList.add('visible')
      }

      function onPointerMove(event) {
        if (!three.renderer) return
        updatePointer(event)
        if (three.draggedNodeID) {
          dragActiveNode()
          document.getElementById('graph').style.cursor = 'grabbing'
          return
        }
        const hits = intersectNodeMeshes()
        const hoveredNodeID = hits[0]?.object?.userData?.nodeID || ''
        if (hoveredNodeID !== three.hoveredNodeID) {
          three.hoveredNodeID = hoveredNodeID
          syncGraphSelection()
          requestRender()
        }
        if (uiState.interactionMode === 'rotate') {
          three.hoverTiltY = three.pointer.x * 0.64
          three.hoverTiltX = -three.pointer.y * 0.42
        }
        document.getElementById('graph').style.cursor = three.hoveredNodeID ? 'grab' : 'default'
        requestRender()
      }

      function onPointerDown(event) {
        if (!three.renderer) return
        updatePointer(event)
        const hit = intersectNodeMeshes()[0]
        three.pointerDownNodeID = hit?.object?.userData?.nodeID || ''
        if (!three.pointerDownNodeID) return
        const mesh = three.nodeMeshes.get(three.pointerDownNodeID)
        if (!mesh) return
        three.draggedNodeID = three.pointerDownNodeID
        three.hoveredNodeID = three.pointerDownNodeID
        three.dragMoved = false
        three.dragStartPosition.copy(mesh.position)
        three.controls.enabled = false
        const normal = new THREE.Vector3()
        three.camera.getWorldDirection(normal)
        three.dragPlane.setFromNormalAndCoplanarPoint(normal, mesh.position)
        if (three.raycaster.ray.intersectPlane(three.dragPlane, three.dragPoint)) {
          three.dragOffset.copy(mesh.position).sub(three.dragPoint)
        } else {
          three.dragOffset.set(0, 0, 0)
        }
        document.getElementById('graph').style.cursor = 'grabbing'
        syncGraphSelection()
        requestRender()
      }

      function onPointerUp() {
        if (!three.renderer) return
        const draggedNodeID = three.draggedNodeID
        const dragMoved = three.dragMoved
        three.draggedNodeID = ''
        three.controls.enabled = true
        if (draggedNodeID) {
          updateNodeGeometry(draggedNodeID)
        }
        if (dragMoved) {
          window.setTimeout(() => {
            three.pointerDownNodeID = ''
            three.dragMoved = false
          }, 0)
        }
        syncGraphSelection()
        requestRender()
      }

      function onCanvasClick() {
        if (three.pointerDownNodeID && three.pointerDownNodeID === three.hoveredNodeID && !three.draggedNodeID && !three.dragMoved) {
          showDetail(three.hoveredNodeID)
          focusNode(three.hoveredNodeID)
        } else if (three.hoveredNodeID && !three.pointerDownNodeID) {
          showDetail(three.hoveredNodeID)
          focusNode(three.hoveredNodeID)
        }
        three.pointerDownNodeID = ''
      }

      function focusNode(nodeID) {
        const mesh = three.nodeMeshes.get(nodeID)
        if (!mesh) return
        three.controls.target.copy(mesh.position)
        requestRender()
      }

      function syncGraphSelection() {
        if (!three.nodeMeshes.size) return
        three.nodeMeshes.forEach((mesh, nodeID) => {
          const material = mesh.material
          const active = nodeID === state.selectedNodeID
          const related = !active && state.selectedSessionID && mesh.userData.sessionID === state.selectedSessionID
          const hovered = nodeID === three.hoveredNodeID
          const dragged = nodeID === three.draggedNodeID
          mesh.scale.setScalar(dragged ? 1.78 : active ? 1.65 : related ? 1.28 : hovered ? 1.18 : 1)
          material.emissiveIntensity = active ? 0.82 : related ? 0.46 : hovered ? 0.34 : 0.14
          material.opacity = active || related || hovered || !state.selectedSessionID ? 1 : 0.35
          material.transparent = material.opacity < 1
        })

        three.edgeObjects.forEach(edge => {
          const sourceNode = edge.userData.source
          const targetNode = edge.userData.target
          const active = sourceNode === state.selectedNodeID || targetNode === state.selectedNodeID
          const related = state.selectedSessionID && isEdgeRelatedToSession(sourceNode, targetNode, state.selectedSessionID)
          edge.material.opacity = active ? 0.92 : related ? 0.48 : state.selectedSessionID ? 0.1 : 0.22
          edge.material.color.setHex(active ? 0xf4d28d : related ? 0x7ae7ff : 0x8fb6c6)
        })
        requestRender()
      }

      function isEdgeRelatedToSession(sourceNodeID, targetNodeID, sessionID) {
        const source = (state.snapshot?.graph?.nodes || []).find(node => node.id === sourceNodeID)
        const target = (state.snapshot?.graph?.nodes || []).find(node => node.id === targetNodeID)
        return [source, target].some(node => node && ((node.session_id && node.session_id === sessionID) || (node.kind === 'session' && node.id === ('session:' + sessionID))))
      }

      function renderChips(values, prefix = '') {
        if (!values || !values.length) return ''
        return '<div class="chips">' + values.map(value => {
          const label = prefix ? (prefix + ':' + value) : value
          return '<span class="chip">' + escapeHTML(label) + '</span>'
        }).join('') + '</div>'
      }

      function projectDisplayLabel(project) {
        const candidates = [
          project?.name,
          project?.cwd ? basename(project.cwd) : '',
          project?.path ? basename(project.path) : '',
          project?.key,
        ]
        for (const candidate of candidates) {
          const cleaned = cleanProjectLabel(candidate)
          if (cleaned) return cleaned
        }
        return 'Project'
      }

      function cleanProjectLabel(value) {
        const input = String(value || '').trim()
        if (!input) return ''
        if (input.startsWith('-Users-') || input.startsWith('-home-') || input.startsWith('/')) {
          const derived = encodedProjectBasename(input)
          if (derived) return prettyProjectLabel(derived)
        }
        return prettyProjectLabel(input)
      }

      function prettyProjectLabel(value) {
        const cleaned = String(value || '')
          .replaceAll('_', ' ')
          .replaceAll('-', ' ')
          .replace(/\s+/g, ' ')
          .trim()
        return cleaned || ''
      }

      function basename(value) {
        const parts = String(value || '')
          .split(/[\\/]/)
          .filter(Boolean)
        return parts.length ? parts[parts.length - 1] : ''
      }

      function encodedProjectBasename(value) {
        const input = String(value || '').trim()
        if (!input) return ''
        if (input.startsWith('/')) {
          return basename(input)
        }
        const parts = input.split('-').filter(Boolean)
        if (!parts.length) return input
        if (parts.length >= 2) {
          return parts.slice(-2).join(' ')
        }
        return parts[parts.length - 1]
      }

      function updatePointer(event) {
        const rect = three.renderer.domElement.getBoundingClientRect()
        three.pointer.x = ((event.clientX - rect.left) / rect.width) * 2 - 1
        three.pointer.y = -((event.clientY - rect.top) / rect.height) * 2 + 1
        three.raycaster.setFromCamera(three.pointer, three.camera)
      }

      function intersectNodeMeshes() {
        return three.raycaster.intersectObjects(Array.from(three.nodeMeshes.values()))
      }

      function dragActiveNode() {
        const mesh = three.nodeMeshes.get(three.draggedNodeID)
        if (!mesh) return
        if (!three.raycaster.ray.intersectPlane(three.dragPlane, three.dragPoint)) return
        mesh.position.copy(three.dragPoint).add(three.dragOffset)
        if (mesh.position.distanceToSquared(three.dragStartPosition) > 9) {
          three.dragMoved = true
        }
        const nodeID = mesh.userData.nodeID
        three.positions[nodeID] = {
          x: mesh.position.x,
          y: mesh.position.y,
          z: mesh.position.z,
        }
        updateNodeGeometry(nodeID)
        syncGraphSelection()
        requestRender()
      }

      function updateNodeGeometry(nodeID) {
        const mesh = three.nodeMeshes.get(nodeID)
        if (!mesh) return
        const sprite = three.labelByNodeID.get(nodeID)
        if (sprite) {
          sprite.position.set(mesh.position.x, mesh.position.y + mesh.userData.radius + 18, mesh.position.z)
        }
        three.edgeObjects.forEach(edge => {
          if (edge.userData.source !== nodeID && edge.userData.target !== nodeID) return
          updateEdgeGeometry(edge)
        })
      }

      function updateEdgeGeometry(edge) {
        const from = resolveNodeVector(edge.userData.source)
        const to = resolveNodeVector(edge.userData.target)
        if (!from || !to) return
        const curve = new THREE.QuadraticBezierCurve3(
          from,
          new THREE.Vector3((from.x + to.x) / 2, Math.max(from.y, to.y) + 70, (from.z + to.z) / 2),
          to,
        )
        const points = curve.getPoints(24)
        edge.geometry.dispose()
        edge.geometry = new THREE.BufferGeometry().setFromPoints(points)
      }

      function resolveNodeVector(nodeID) {
        const mesh = three.nodeMeshes.get(nodeID)
        if (mesh) {
          return mesh.position.clone()
        }
        const pos = three.positions[nodeID]
        if (!pos) return null
        return new THREE.Vector3(pos.x, pos.y, pos.z)
      }

      function formatDate(value) {
        if (!value) return 'unknown'
        try {
          return new Date(value).toLocaleString()
        } catch {
          return value
        }
      }

      function formatCompactNumber(value) {
        const number = Number(value || 0)
        try {
          return new Intl.NumberFormat(undefined).format(number)
        } catch {
          return String(number)
        }
      }

      function escapeHTML(value) {
        return String(value || '')
          .replaceAll('&', '&amp;')
          .replaceAll('<', '&lt;')
          .replaceAll('>', '&gt;')
      }

      document.getElementById('project-select').addEventListener('change', async (event) => {
        state.selectedProject = event.target.value
        await loadSnapshot()
      })
      document.getElementById('reload-btn').addEventListener('click', loadSnapshot)
      document.getElementById('pan-mode-btn').addEventListener('click', () => setInteractionMode('pan'))
      document.getElementById('rotate-mode-btn').addEventListener('click', () => setInteractionMode('rotate'))
      document.getElementById('zoom-in-btn').addEventListener('click', () => zoomGraph(1))
      document.getElementById('zoom-out-btn').addEventListener('click', () => zoomGraph(-1))

      ;(async function bootstrap() {
        try {
          await loadProjects()
          await loadSnapshot()
          setInterval(loadSnapshot, 5000)
        } catch (error) {
          document.getElementById('detail').textContent = String(error)
        }
      })()
    </script>
  </body>
</html>`))
}
