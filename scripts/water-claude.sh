#!/bin/bash

# water-claude.sh: Wrapper for Claude Code agent integration with Water
# Captures stdin (prompt), stdout (response), and file operations
# Usage: water-claude.sh <prompt>

WATER_LOG="${WATER_LOG:-.water/events.jsonl}"
SESSION_ID="${WATER_SESSION_ID:-$(uuidgen 2>/dev/null || echo "session-$$")}"

# Ensure log directory exists
mkdir -p "$(dirname "$WATER_LOG")"

# Timestamp in ISO 8601 format
timestamp() {
    date -u +"%Y-%m-%dT%H:%M:%S.000Z"
}

# Log an event to JSONL
log_event() {
    local event_type="$1"
    local agent="claude"
    local content="$2"
    local exit_code="${3:-0}"
    local metadata="${4:-{}}"
    
    local event="{
        \"id\": \"evt-$(openssl rand -hex 6 2>/dev/null || echo $$)\",
        \"timestamp\": \"$(timestamp)\",
        \"session_id\": \"$SESSION_ID\",
        \"agent\": \"$agent\",
        \"event_type\": \"$event_type\",
        \"content\": $(printf '%s' "$content" | jq -Rs .),
        \"metadata\": $metadata
    }"
    
    echo "$event" >> "$WATER_LOG"
}

# Log execution context
log_event "execution_context" "cwd=$(pwd)" "0" "{\"agent\": \"claude_code\"}"

# Capture input (the prompt/request)
PROMPT="$*"
log_event "input" "$PROMPT" "0" "{\"format\": \"text\", \"length\": ${#PROMPT}}"

# Execute claude command and capture output
# Note: This assumes 'claude' CLI is available
START_TIME=$(date +%s)
if OUTPUT=$(claude "$@" 2>&1); then
    EXIT_CODE=0
else
    EXIT_CODE=$?
fi
END_TIME=$(date +%s)
DURATION_MS=$(( (END_TIME - START_TIME) * 1000 ))

# Log output
OUTPUT_LEN=${#OUTPUT}
log_event "output" "$OUTPUT" "$EXIT_CODE" "{\"length\": $OUTPUT_LEN, \"duration_ms\": $DURATION_MS}"

# Print output to user
echo "$OUTPUT"

exit "$EXIT_CODE"
