#!/usr/bin/env python3
"""
Claude Code PreToolUse Hook: Tool Approval Gates

Enforces consulting-controlled tool usage policies via allowlist/denylist.
When an allowlist is set, only listed tools are permitted. Denied tools are
always blocked regardless of allowlist. Entries are tool names in which "*"
matches any run of characters (e.g. "mcp__github__*"); matching is
case-sensitive, as Claude Code tool names are.

Exit codes:
  0 — allow or deny (with JSON on stdout for deny)
  2 — hook error (fail-closed)

Configuration via environment variables, which qsdev generates into the
"env" block of .claude/settings.json from .qsdev.yaml hooks.tool_gates:
  TOOL_GATES_ALLOWED — comma-separated tool name patterns (empty = all allowed)
  TOOL_GATES_DENIED  — comma-separated tool name patterns to block
With both empty the hook has no policy: it allows and logs every tool.
"""

import fnmatch
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


def matches(tool_name: str, patterns: set[str]) -> bool:
    """Report whether tool_name matches any of the patterns ("*" wildcards)."""
    return any(fnmatch.fnmatchcase(tool_name, p) for p in patterns)


AUDIT_LOG: Path = Path(
    os.environ.get("CLAUDE_PROJECT_DIR", ".")
) / ".claude" / "logs" / "hook-audit.jsonl"


AUDIT_LOG_MAX_BYTES = 10 * 1024 * 1024


def audit_log(entry: dict) -> None:
    """Append a JSON entry to the audit log. Never raises. The file is created
    0600 and rotated to <name>.1 once it exceeds AUDIT_LOG_MAX_BYTES."""
    try:
        AUDIT_LOG.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        try:
            if AUDIT_LOG.stat().st_size > AUDIT_LOG_MAX_BYTES:
                os.replace(AUDIT_LOG, AUDIT_LOG.with_name(AUDIT_LOG.name + ".1"))
        except FileNotFoundError:
            pass
        entry["timestamp"] = datetime.now(timezone.utc).isoformat()
        fd = os.open(AUDIT_LOG, os.O_WRONLY | os.O_APPEND | os.O_CREAT, 0o600)
        with os.fdopen(fd, "a") as f:
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
    if not isinstance(tool_name, str) or not tool_name:
        if ALLOWED_TOOLS or DENIED_TOOLS:
            # A policy cannot be applied to a call it cannot name: fail closed.
            print("tool gates error: payload has no tool_name", file=sys.stderr)
            sys.exit(2)
        sys.exit(0)

    # Check denylist first.
    if matches(tool_name, DENIED_TOOLS):
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
    if ALLOWED_TOOLS and not matches(tool_name, ALLOWED_TOOLS):
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

    entry = {"event": "allow", "hook": "tool-gates", "tool": tool_name}
    if not ALLOWED_TOOLS and not DENIED_TOOLS:
        entry["reason"] = "no_policy"
    audit_log(entry)
    sys.exit(0)


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(f"tool gates error: {e}", file=sys.stderr)
        sys.exit(2)
