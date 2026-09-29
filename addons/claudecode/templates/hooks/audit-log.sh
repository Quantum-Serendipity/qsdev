#!/usr/bin/env bash
set -euo pipefail

# PostToolUse audit logging hook for Claude Code.
# Appends one JSON-lines entry per tool invocation to a per-day local audit log.
#
# Entries hold metadata only: the tool name, the paths it touched, flags, and
# the length and SHA-256 of every other input (file content, edit strings,
# shell commands, URLs). The inputs themselves are never written, because they
# carry secrets: the .env being written, a bearer token in a curl command.
# Logs are created 0600 under .claude/logs/, which `qsdev init` gitignores.
#
# This hook only observes: it prints nothing and never fails the tool call.

umask 077
LOG_DIR="${CLAUDE_PROJECT_DIR:-.}/.claude/logs"
LOG_FILE="${LOG_DIR}/audit-$(date +%Y-%m-%d).jsonl"
mkdir -p "$LOG_DIR" 2>/dev/null || exit 0

python3 -c '
import hashlib, json, sys
from datetime import datetime, timezone

# tool_input keys recorded verbatim: where a tool read or wrote, not what.
PATH_KEYS = {"file_path", "notebook_path", "path"}

def fingerprint(value):
    text = value if isinstance(value, str) else json.dumps(value, sort_keys=True)
    return {"sha256": hashlib.sha256(text.encode("utf-8", "surrogateescape")).hexdigest(), "len": len(text)}

try:
    data = json.load(sys.stdin)
except ValueError:
    data = {}
if not isinstance(data, dict):
    data = {}
tool_input = data.get("tool_input")
if not isinstance(tool_input, dict):
    tool_input = {}
meta = {}
for key, value in tool_input.items():
    if key in PATH_KEYS and isinstance(value, str) or isinstance(value, (bool, int, float)) or value is None:
        meta[key] = value
    else:
        meta[key] = fingerprint(value)
print(json.dumps({
    "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
    "tool": str(data.get("tool_name") or "unknown"),
    "session_id": str(data.get("session_id") or ""),
    "input": meta,
}))
' >>"$LOG_FILE" 2>/dev/null || true
