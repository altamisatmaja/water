package server

import (
	"net/http"

	"github.com/water-viz/water/internal/capture"
	"github.com/water-viz/water/internal/claude"
	"github.com/water-viz/water/internal/config"
	"github.com/water-viz/water/internal/graph"
)

type Server struct {
	cfg        *config.Config
	claude     *claude.Store
	graph      *graph.Client
	writer     *capture.Writer
	eventsPath string
}

func NewServer(cfg *config.Config, g *graph.Client, w *capture.Writer, eventsPath string) *Server {
	var claudeStore *claude.Store
	if store, err := claude.NewStore(cfg.ClaudeProjectsPath); err == nil {
		claudeStore = store
	}
	return &Server{
		cfg:        cfg,
		claude:     claudeStore,
		graph:      g,
		writer:     w,
		eventsPath: eventsPath,
	}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/nodes", s.handleGetNodes)
	mux.HandleFunc("GET /api/edges", s.handleGetEdges)
	mux.HandleFunc("GET /api/graph", s.handleGetGraph)
	mux.HandleFunc("GET /api/project", s.handleGetProject)
	mux.HandleFunc("GET /api/projects", s.handleGetProjects)
	mux.HandleFunc("GET /api/history", s.handleGetHistory)
	mux.HandleFunc("GET /api/memo", s.handleGetMemo)
	mux.HandleFunc("GET /api/stats", s.handleGetStats)
	mux.HandleFunc("POST /api/events", s.handlePostEvent)

	mux.HandleFunc("GET /ws", s.handleWS)

	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return s.withCORS(s.withLogging(mux))
}
