#!/bin/sh
# Semble search analytics — tracks MCP tool usage patterns.
# Fail-open: logging errors never block tool execution.

set -e

TOOL_NAME="${CLAUDE_TOOL_NAME:-}"

case "$TOOL_NAME" in
    mcp__semble__*) ;;
    *) printf '{"decision":"approve"}\n'; exit 0 ;;
esac

LOG_DIR="${CLAUDE_PROJECT_DIR:-.}/.qsdev/analytics"
LOG_FILE="$LOG_DIR/semble-searches.jsonl"

mkdir -p "$LOG_DIR" 2>/dev/null || true

TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
SESSION_ID="${CLAUDE_SESSION_ID:-unknown}"
PROJECT_ROOT="${CLAUDE_PROJECT_DIR:-.}"

INPUT=$(cat)
QUERY=$(printf '%s' "$INPUT" | grep -o '"query"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"query"[[:space:]]*:[[:space:]]*"//;s/"$//' 2>/dev/null || echo "")

printf '{"timestamp":"%s","tool":"%s","query":"%s","sessionId":"%s","projectRoot":"%s"}\n' \
    "$TIMESTAMP" "$TOOL_NAME" "$QUERY" "$SESSION_ID" "$PROJECT_ROOT" \
    >> "$LOG_FILE" 2>/dev/null || true

printf '{"decision":"approve"}\n'
