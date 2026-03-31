# Water - Quick Reference

## Commands

### Initialize Database
```bash
./bin/water init [--db-path=.water]
```
Creates `.water/database.duckdb` and `config.json`

### Start Server
```bash
./bin/water serve [--port=3141] [--open-browser] [--db-path=.water]
```
Runs on `http://127.0.0.1:3141` by default

### Wrapper Scripts
```bash
# Generic (any command)
./scripts/water-run.sh "agent-name" <command> [args...]

# Specific agents (when CLI is installed)
./scripts/water-copilot.sh suggest "prompt"
./scripts/water-claude.sh "prompt"
./scripts/water-aichat.sh "prompt"
```

---

## Environment Variables

```bash
# Override event log location
export WATER_LOG=".water/events.jsonl"

# Set persistent session ID
export WATER_SESSION_ID="my-session-$(date +%s)"

# Disable logging
export WATER_DISABLE=1
```

---

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/nodes` | List all nodes |
| `GET` | `/api/nodes/:id` | Get single node |
| `GET` | `/api/edges` | List all edges |
| `GET` | `/api/graph` | Full graph (nodes + edges) |
| `GET` | `/api/stats` | Statistics |
| `GET` | `/api/traces` | Reasoning traces |
| `POST` | `/api/events` | Ingest event |
| `GET` | `/ws` | WebSocket stream |
| `GET` | `/healthz` | Health check |

---

## Database Schema Tables

**events_log**
- id, timestamp, session_id, agent, event_type, content, metadata

**knowledge_nodes**
- node_id, content, source_type, source_tool, tokens_in/out, importance_score

**edges**
- edge_id, from_node_id, to_node_id, edge_type, weight, salience

**reasoning_traces**
- trace_id, session_id, agent, steps, status, started_at, completed_at

**sessions**
- session_id, created_at, closed_at, agents, total_events

**daily_metrics**
- metric_id, date, agent, total_tokens, unique_nodes, edges_created

---

## Event Types

| Type | Description |
|------|-------------|
| `input` | User prompt/command |
| `output` | Agent response |
| `execution_context` | CWD, env, session |
| `file_access` | File read/write/delete |
| `command_execution` | Shell command details |
| `mcp_tool_call` | MCP tool invocation |
| `context_window` | Token usage |
| `decision` | Agent decision point |
| `error` | Error/exception |

---

## Quick Workflows

### Single Command
```bash
./bin/water init
./bin/water serve &
export WATER_LOG=".water/events.jsonl"
./scripts/water-run.sh "my-agent" npm run build
curl http://127.0.0.1:3141/api/graph | jq .
```

### Multi-Agent Session
```bash
export WATER_SESSION_ID="session-$(date +%s)"
export WATER_LOG=".water/events.jsonl"

./scripts/water-claude.sh "design architecture"
./scripts/water-copilot.sh suggest "implement feature"
./scripts/water-copilot.sh suggest "write tests"
```

### Debug Failed Build
```bash
export WATER_LOG=".water/build-debug.jsonl"
./scripts/water-run.sh "build-debug" npm run build

# Inspect event stream
jq 'select(.event_type == "output")' .water/build-debug.jsonl | jq -s .
```

---

## File Locations

```
.water/
├── config.json          ← Configuration
├── database.duckdb      ← DuckDB file (SQL queries here)
├── events.jsonl         ← Raw event log
└── database-wal         ← Write-ahead log

bin/
└── water               ← Executable (macOS)

scripts/
├── water-copilot.sh
├── water-claude.sh
├── water-aichat.sh
└── water-run.sh
```

---

## Config (`.water/config.json`)

```json
{
  "db_path": ".water",
  "host": "127.0.0.1",
  "port": 3141,
  "embedding_mode": "local",
  "log_level": "info",
  "enable_websocket": true,
  "enable_analytics": false
}
```

---

## Troubleshooting

| Problem | Solution |
|---------|----------|
| "Cannot open database" | Run `water init` first |
| "Address already in use" | Use `--port=3142` |
| "No events logged" | Check `$WATER_LOG` env var |
| "Events in log but not in DB" | Restart server to ingest |
| "Wrapper script not found" | Check `scripts/` exists |

---

**Water — Visibility into AI reasoning.** 🌊
