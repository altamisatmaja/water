# Water Quick Start

## 🚀 Getting Started (30 seconds)

### 1. Initialize Database
```bash
cd /path/to/water
./bin/water init --db-path=.water
```

Expected output:
```
✓ Water initialized at .water
  Next: water serve --db-path .water
```

### 2. Start the Server
```bash
./bin/water serve --db-path=.water
# Opens http://localhost:3141 automatically
```

### 3. Use a Wrapper Script
In another terminal:
```bash
export WATER_LOG=".water/events.jsonl"

# Test with any command
./scripts/water-run.sh "my-agent" npm run build

# Or with Copilot (if installed)
./scripts/water-copilot.sh suggest "build REST API"
```

### 4. Check Results
```bash
# View graph
curl http://localhost:3141/api/graph | jq .

# List nodes
curl http://localhost:3141/api/nodes | jq .
```

---

## 📝 Important Notes

### ✅ Do This First
```bash
# Initialize BEFORE running serve
./bin/water init --db-path=.water
```

### ❌ Don't Forget
```bash
# When running serve, use the SAME db-path
./bin/water serve --db-path=.water

# NOT ./bin/water serve  ← Missing db-path will use default
```

### 🔧 Environment Variable
Set once to avoid repeating:
```bash
export WATER_LOG=".water/events.jsonl"
# Now wrappers will use this location
```

---

## 📊 What Gets Logged

Each command creates ~30-40 events:

```json
{
  "id": "evt-abc123",
  "timestamp": "2024-03-31T02:04:00.000Z",
  "session_id": "session-12345",
  "agent": "test-agent",
  "event_type": "input",
  "content": "echo Hello"
}
```

Event types:
- `input` — Command/prompt sent to agent
- `output` — Response from agent
- `execution_context` — Working directory, environment
- `command_execution` — Detailed command metadata
- Plus legacy types: mcp_tool_call, context_window, decision, error

---

## 🎯 Next Steps

### Try the Dashboard
```bash
# Once server is running, open browser:
open http://localhost:3141
```

### Create a Persistent Session
```bash
export WATER_SESSION_ID="my-session-$(date +%s)"

# All commands in this shell share same session
./scripts/water-run.sh "agent-1" echo "first"
./scripts/water-run.sh "agent-2" echo "second"
./scripts/water-run.sh "agent-3" echo "third"

# Query session results
curl "http://localhost:3141/api/events?session=$WATER_SESSION_ID"
```

### Watch Events in Real-Time
```bash
# Coming soon: water watch command
# For now, tail the log:
tail -f .water/events.jsonl | jq .
```

---

## 🔧 Troubleshooting

### "Cannot open file database.duckdb"
→ Did you run `water init` first?
```bash
./bin/water init --db-path=.water
```

### "Address already in use :3141"
→ Use a different port:
```bash
./bin/water serve --port=3142
```

### No events logged
→ Check WATER_LOG env var:
```bash
echo $WATER_LOG
# Should print: .water/events.jsonl
```

→ Or specify it when running wrapper:
```bash
WATER_LOG=".water/events.jsonl" ./scripts/water-run.sh "agent" echo "test"
```

---

## 📚 Full Documentation

- See [INTEGRATION.md](./INTEGRATION.md) for agent-specific setup
- See [CLAUDE.md](./CLAUDE.md) for architecture overview
