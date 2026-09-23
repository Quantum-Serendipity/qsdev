#!/bin/sh
# Semble search analytics — PostToolUse hook on the semble MCP tools
# (settings.json matcher mcp__semble__*). Appends one JSON line per search to
# .qsdev/analytics/semble-searches.jsonl.
#
# Claude Code passes the tool call only on stdin (tool_name, tool_input,
# session_id); it sets no CLAUDE_TOOL_NAME/CLAUDE_SESSION_ID variables. jq
# builds the entry, so a query containing quotes or backslashes stays one
# well-formed JSON string.
#
# Fail-open: logging errors (including a missing jq) never affect the tool
# call, and the hook prints nothing (PostToolUse defines no "approve").

LOG_DIR="${CLAUDE_PROJECT_DIR:-.}/.qsdev/analytics"
LOG_FILE="$LOG_DIR/semble-searches.jsonl"

umask 077
mkdir -p "$LOG_DIR" 2>/dev/null || exit 0

jq -c --arg root "${CLAUDE_PROJECT_DIR:-.}" '
	select((.tool_name // "") | startswith("mcp__semble__"))
	| {
		timestamp: (now | todate),
		tool: .tool_name,
		query: (.tool_input.query // ""),
		sessionId: (.session_id // ""),
		projectRoot: $root
	}
' >>"$LOG_FILE" 2>/dev/null || true
exit 0
