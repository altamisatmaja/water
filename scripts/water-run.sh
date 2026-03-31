#!/bin/bash

# water-run.sh: Generic wrapper for arbitrary command execution with Water tracking
# Captures input, output, exit code, and execution metadata
# Usage: water-run.sh <agent_name> <command> [args...]
# Example: water-run.sh "bash" "npm" "run" "build"

set -e

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
    local agent="$2"
    local content="$3"
    local exit_code="${4:-0}"
    local metadata="${5:-{}}"
    
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

# Parse arguments
AGENT_NAME="${1:-unknown}"
shift || true
COMMAND="$1"
shift || true

if [ -z "$COMMAND" ]; then
    echo "Usage: water-run.sh <agent_name> <command> [args...]" >&2
    exit 1
fi

# Log execution context
log_event "execution_context" "$AGENT_NAME" "cwd=$(pwd)" "0" "{\"command\": \"$COMMAND\", \"args\": $(printf '%s\n' "$@" | jq -Rs .)}"

# Capture input
FULL_CMD="$COMMAND $*"
log_event "input" "$AGENT_NAME" "$FULL_CMD" "0" "{\"format\": \"command\"}"

# Execute command and capture output
START_TIME=$(date +%s)
if OUTPUT=$("$COMMAND" "$@" 2>&1); then
    EXIT_CODE=0
else
    EXIT_CODE=$?
fi
END_TIME=$(date +%s)
DURATION_MS=$(( (END_TIME - START_TIME) * 1000 ))

# Log output
OUTPUT_LEN=${#OUTPUT}
log_event "output" "$AGENT_NAME" "$OUTPUT" "$EXIT_CODE" "{\"length\": $OUTPUT_LEN, \"duration_ms\": $DURATION_MS}"

# Log command execution details
log_event "command_execution" "$AGENT_NAME" "$COMMAND" "$EXIT_CODE" "{\"args\": $(printf '%s\n' "$@" | jq -Rs .), \"stdout_len\": $OUTPUT_LEN, \"duration_ms\": $DURATION_MS}"

# Print output to user
echo "$OUTPUT"

exit "$EXIT_CODE"
