#!/usr/bin/env python3
"""
Claude Code PreToolUse Hook: Tool Approval Gates

Enforces consulting-controlled tool usage policies via allowlist/denylist.
When an allowlist is set, only listed tools are permitted. Denied tools are
always blocked regardless of allowlist.

Exit codes:
  0 — allow or deny (with JSON on stdout for deny)
  2 — hook error (fail-closed)

Configuration via environment variables:
  TOOL_GATES_ALLOWED — comma-separated tool names (empty = all allowed)
  TOOL_GATES_DENIED  — comma-separated tool names to block
"""

import json
import os
import sys
from datetime import datetime, timezone
from pathlib import Path

ALLOWED_TOOLS: set[str] = {
    t.strip()
    for t in os.environ.get("TOOL_GATES_ALLOWED", "").split(",")
    if t.strip()
}

DENIED_TOOLS: set[str] = {
    t.strip()
    for t in os.environ.get("TOOL_GATES_DENIED", "").split(",")
    if t.strip()
}

AUDIT_LOG: Path = Path(
    os.environ.get("CLAUDE_PROJECT_DIR", ".")
) / ".claude" / "logs" / "hook-audit.jsonl"


def audit_log(entry: dict) -> None:
    """Append a JSON entry to the audit log. Never raises."""
    try:
        AUDIT_LOG.parent.mkdir(parents=True, exist_ok=True)
        entry["timestamp"] = datetime.now(timezone.utc).isoformat()
        with open(AUDIT_LOG, "a") as f:
            f.write(json.dumps(entry) + "\n")
    except OSError:
        pass  # Audit logging must not interrupt hook decisions.


def main() -> None:
    try:
        input_data = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError) as e:
        audit_log({"event": "parse_error", "hook": "tool-gates", "error": str(e)})
        print(f"tool gates error: {e}", file=sys.stderr)
        sys.exit(2)

    tool_name = input_data.get("tool_name", "")
    if not tool_name:
        sys.exit(0)

    # Check denylist first.
    if tool_name in DENIED_TOOLS:
        audit_log({
            "event": "deny",
            "hook": "tool-gates",
            "tool": tool_name,
            "reason": "denylist",
        })
        result = {
            "hookSpecificOutput": {
                "hookEventName": "PreToolUse",
                "permissionDecision": "deny",
                "permissionDecisionReason": (
                    f"Tool {tool_name} is not permitted by consulting policy "
                    f"for this context."
                ),
            }
        }
        print(json.dumps(result))
        sys.exit(0)

    # Check allowlist (empty = all allowed).
    if ALLOWED_TOOLS and tool_name not in ALLOWED_TOOLS:
        audit_log({
            "event": "deny",
            "hook": "tool-gates",
            "tool": tool_name,
            "reason": "not_in_allowlist",
        })
        result = {
            "hookSpecificOutput": {
                "hookEventName": "PreToolUse",
                "permissionDecision": "deny",
                "permissionDecisionReason": (
                    f"Tool {tool_name} is not permitted by consulting policy "
                    f"for this context. Allowed tools: {', '.join(sorted(ALLOWED_TOOLS))}"
                ),
            }
        }
        print(json.dumps(result))
        sys.exit(0)

    audit_log({
        "event": "allow",
        "hook": "tool-gates",
        "tool": tool_name,
    })
    sys.exit(0)


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(f"tool gates error: {e}", file=sys.stderr)
        sys.exit(2)
