#!/usr/bin/env bash
set -euo pipefail

# PostToolUse audit logging hook for Claude Code.
# Appends a JSON-lines entry for every tool invocation to a local audit log.

LOG_DIR="${CLAUDE_PROJECT_DIR:-.}/.claude/logs"
LOG_FILE="${LOG_DIR}/audit-$(date +%Y-%m-%d).jsonl"
mkdir -p "$LOG_DIR"

TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Read the hook input from stdin.
INPUT=$(cat)

TOOL_NAME=$(echo "$INPUT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('tool_name','unknown'))" 2>/dev/null || echo "unknown")
TOOL_INPUT=$(echo "$INPUT" | python3 -c "import sys,json; print(json.dumps(json.load(sys.stdin).get('tool_input',{})))" 2>/dev/null || echo "{}")

printf '{"timestamp":"%s","tool":"%s","input":%s}\n' \
  "$TIMESTAMP" "$TOOL_NAME" "$TOOL_INPUT" >> "$LOG_FILE"

# Always allow — this hook only observes.
echo '{"decision":"approve"}'
