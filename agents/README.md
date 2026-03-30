# Water Agents

Agent definitions untuk Claude Code. Setiap file mendeskripsikan satu specialized agent beserta tools, context, dan task scope-nya.

## Hierarki

```
orchestrator
├── backend-agent       ← Go code (CLI, server, graph, metrics)
├── frontend-agent      ← Svelte UI (Cytoscape, stores, API)
├── schema-agent        ← DuckDB schema & SQL queries
└── devops-agent        ← CI/CD, Makefile, Homebrew
```

## Cara Pakai

Di Claude Code, mulai dengan orchestrator untuk task besar:
```
/agents/orchestrator.md — implement week 1 tasks
```

Atau langsung ke specialist agent untuk task spesifik:
```
/agents/backend-agent.md — implement internal/graph/nodes.go
```

Setiap agent akan otomatis membaca skill yang relevan sebelum menghasilkan kode.