# Water Integration Guide

Water captures observable cognition traces from multiple AI agents by wrapping their CLI commands. This guide explains how to integrate different agents.

## Architecture

```
Agent CLI (Copilot, Claude, aichat)
    ↓ (via wrapper script)
events.jsonl (append-only JSONL log)
    ↓ (read by water server)
DuckDB (knowledge_nodes, edges, reasoning_traces, sessions)
    ↓ (REST API + WebSocket)
Web Dashboard + CLI watch
```

Each wrapper script:
1. Logs the **input** (user prompt, command arguments)
2. Logs **execution context** (CWD, session ID, environment)
3. Runs the original agent command
4. Logs the **output** (response, exit code, duration)

---

## Global Configuration

Set these environment variables to customize Water's behavior:

```bash
# Override the events log location (default: .water/events.jsonl)
export WATER_LOG="/path/to/events.jsonl"

# Set a persistent session ID across commands (default: auto-generated per command)
export WATER_SESSION_ID="my-session-$(date +%s)"

# Disable Water logging entirely
export WATER_DISABLE=1
```

---

## Copilot CLI Integration

### Setup

```bash
# Add to your ~/.zshrc or ~/.bashrc:
alias gh="$REPO/scripts/water-copilot.sh gh"

# Or use directly:
/path/to/water/scripts/water-copilot.sh suggest "build express API with JWT"
```

### What Gets Logged

```json
{
  "id": "evt-abc123",
  "timestamp": "2024-03-30T14:00:00.000Z",
  "session_id": "session-12345",
  "agent": "copilot",
  "event_type": "input",
  "content": "build express API with JWT",
  "metadata": {}
}
{
  "id": "evt-def456",
  "timestamp": "2024-03-30T14:00:01.500Z",
  "session_id": "session-12345",
  "agent": "copilot",
  "event_type": "output",
  "content": "Here's an Express API with JWT...",
  "metadata": {
    "length": 2048,
    "duration_ms": 1500,
    "exit_code": 0
  }
}
```

### Example: Typical Workflow

```bash
# One-time: add alias
echo 'alias gh="$REPO/scripts/water-copilot.sh gh"' >> ~/.zshrc
source ~/.zshrc

# Now use normally
gh copilot suggest "list all users with their posts"

# Verify logging
tail -f .water/events.jsonl
```

---

## Claude Code Integration

### Setup

```bash
# Add to your shell profile:
alias claude="$REPO/scripts/water-claude.sh"

# Or use directly:
/path/to/water/scripts/water-claude.sh "analyze the architecture of this repo"
```

### Example

```bash
# Run a Claude command
claude "What's the best way to structure this codebase?"

# Logs both:
# - input: "What's the best way to structure this codebase?"
# - output: Claude's response
# - execution context: CWD, session ID
```

---

## aichat Integration

### Setup

```bash
# Add to shell profile:
alias aichat="$REPO/scripts/water-aichat.sh"

# Or use directly:
/path/to/water/scripts/water-aichat.sh "explain OAuth 2.0"
```

---

## Generic Command Wrapper

For any arbitrary command, use `water-run.sh`:

```bash
# Syntax: water-run.sh <agent_name> <command> [args...]

# Example: wrap npm commands
water-run.sh "npm" npm run build

# Example: wrap git commands with a custom agent name
water-run.sh "git-assistant" git log --oneline -10

# Example: wrap Python scripts
water-run.sh "python-agent" python analyze.py
```

---

## Viewing Events in Real-Time

### Using `water watch` (Recommended)

Once Water server is running:

```bash
# Terminal 1: Start water
water serve

# Terminal 2: Watch events in real-time
water watch

# Output:
# [14:00:01] [copilot] INPUT: build express API
# [14:00:03] [copilot] OUTPUT: Here's an Express server... (2048 chars)
# [14:00:05] [claude] INPUT: analyze repo
# [14:00:08] [claude] OUTPUT: Architecture summary... (1024 chars)
```

### Using `tail`

```bash
tail -f .water/events.jsonl | jq '.'
```

### Parse with jq

```bash
# Show all copilot outputs
jq 'select(.agent == "copilot" and .event_type == "output")' .water/events.jsonl

# Show top 10 slowest operations
jq 'select(.metadata.duration_ms) | {agent, duration_ms: .metadata.duration_ms}' .water/events.jsonl | \
  sort -t: -k3 -rn | head -10
```

---

## Creating a Persistent Session

Track related commands in a single "session" by setting `WATER_SESSION_ID`:

```bash
# Start a session
SESSION_ID="debug-$(date +%s)"
export WATER_SESSION_ID="$SESSION_ID"

# All commands in this shell will share the same session
gh copilot suggest "build API"
claude "review this design"
aichat "summarize the conversation"

# Later, query this session:
water watch --session="$SESSION_ID"
```

---

## Troubleshooting

### Wrapper script not found

```bash
# Ensure wrapper scripts are executable
chmod +x /path/to/water/scripts/water-*.sh

# And in your PATH (if not using full path)
export PATH="/path/to/water/scripts:$PATH"
```

### Events not being logged

```bash
# Check if WATER_LOG directory exists
ls -la .water/

# Check permissions
ls -la .water/events.jsonl

# Try writing manually
echo '{"test": true}' >> .water/events.jsonl
```

### Command not found after alias

```bash
# Reload shell
source ~/.zshrc  # or ~/.bashrc

# Verify alias is set
alias | grep "water"

# Try absolute path instead
/path/to/water/scripts/water-copilot.sh suggest "test"
```

### Events appearing but not in web dashboard

1. Is `water serve` running?
   ```bash
   curl http://localhost:3141/api/graph
   ```

2. Check server logs for parsing errors
3. Verify events are valid JSON:
   ```bash
   jq . .water/events.jsonl > /dev/null
   ```

---

## Advanced: Custom Agents

To wrap your own agent, follow this pattern:

```bash
#!/bin/bash

WATER_LOG="${WATER_LOG:-.water/events.jsonl}"
SESSION_ID="${WATER_SESSION_ID:-$(uuidgen 2>/dev/null || echo "session-$$")}"

mkdir -p "$(dirname "$WATER_LOG")"

log_event() {
    local event_type="$1"
    local agent="$2"
    local content="$3"
    
    printf '{
        "id": "evt-%s",
        "timestamp": "%s",
        "session_id": "%s",
        "agent": "%s",
        "event_type": "%s",
        "content": %s
    }\n' \
        "$(openssl rand -hex 6 2>/dev/null || echo $$)" \
        "$(date -u +%Y-%m-%dT%H:%M:%S.000Z)" \
        "$SESSION_ID" \
        "$agent" \
        "$event_type" \
        "$(printf '%s' "$content" | jq -Rs .)" >> "$WATER_LOG"
}

# Your logic here
AGENT="my_agent"
PROMPT="$*"

log_event "input" "$AGENT" "$PROMPT"
OUTPUT=$(/path/to/your/agent "$PROMPT")
log_event "output" "$AGENT" "$OUTPUT"

echo "$OUTPUT"
```

---

## Best Practices

1. **Use persistent session IDs** for related operations:
   ```bash
   export WATER_SESSION_ID="task-$(date +%Y%m%d-%H%M%S)"
   ```

2. **Keep prompts concise** - Water logs the full content, so keep it readable.

3. **Monitor storage** - events.jsonl can grow large. Archive old sessions:
   ```bash
   water export --session=old-session-id > old-session.json
   rm .water/events.jsonl  # start fresh
   ```

4. **Use metadata** for custom tracking:
   ```bash
   # Wrapper scripts support metadata in the log_event function
   log_event "input" "copilot" "$PROMPT" "0" "{\"project\": \"myapp\", \"task\": \"feature-x\"}"
   ```

---

## Next Steps

- See [README.md](../README.md) for web dashboard setup
- Run `water serve` to start the HTTP server
- Run `water watch` for real-time event viewing
- Check the [Architecture](./architecture.md) for deep dives
