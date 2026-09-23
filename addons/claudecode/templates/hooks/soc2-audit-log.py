#!/usr/bin/env python3
"""
Claude Code Hook: SOC 2 Audit Logging

Metadata-only session audit trail across Claude Code hook events:
  session_start      — SessionStart, every source (startup, resume, clear,
                       compact, fork), so each session_id has an
                       attributable start record
  tool_use           — PostToolUse: a tool call that ran
  tool_failure       — PostToolUseFailure: a tool call that failed
  permission_denied  — PermissionDenied: a tool call that was refused
  session_checkpoint — Stop
  session_end        — SessionEnd, with the reason the session ended

Claude Code's hook input carries no cost or token counts, so none are
recorded here.

SOC 2 Trust Services Criteria coverage:
  CC6.1 (Access Controls), CC6.2 (Access Restriction),
  CC7.2 (Monitoring), CC7.3 (Change Detection), CC8.1 (Change Management)

Exit codes:
  0 — always (fail-open: logging failures never block developer work)

Configuration via environment variables:
  SOC2_CLIENT_DIR_PATTERN — regex for client name extraction (default: .*/clients/([^/]*)/.*  )
  CLAUDE_AUDIT_DIR        — output directory (default: ~/.claude/audit)
"""

from __future__ import annotations

import getpass
import json
import os
import platform
import re
import sys
from datetime import datetime, timezone
from pathlib import Path

CLIENT_DIR_PATTERN = re.compile(
    os.environ.get("SOC2_CLIENT_DIR_PATTERN", r".*/clients/([^/]*)/.*")
)

AUDIT_DIR = Path(
    os.environ.get("CLAUDE_AUDIT_DIR", os.path.expanduser("~/.claude/audit"))
)
AUDIT_FILE = AUDIT_DIR / f"claude-sessions-{datetime.now().strftime('%Y-%m')}.jsonl"


def detect_client(cwd: str) -> str:
    """Extract client engagement name from the working directory path."""
    match = CLIENT_DIR_PATTERN.match(cwd)
    return match.group(1) if match else ""


def write_entry(entry: dict) -> None:
    """Append a JSON entry to the monthly audit file, created 0600 in a 0700
    directory. Never raises."""
    try:
        AUDIT_DIR.mkdir(mode=0o700, parents=True, exist_ok=True)
        fd = os.open(AUDIT_FILE, os.O_WRONLY | os.O_APPEND | os.O_CREAT, 0o600)
        with os.fdopen(fd, "a") as f:
            f.write(json.dumps(entry) + "\n")
    except OSError:
        print("warning: failed to write SOC2 audit entry", file=sys.stderr)


def handle_session_start(input_data: dict) -> None:
    cwd = os.environ.get("CLAUDE_PROJECT_DIR", "")
    write_entry({
        "event": "session_start",
        "session_id": input_data.get("session_id", ""),
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "user": getpass.getuser(),
        "hostname": platform.node(),
        "project_dir": cwd,
        "client_engagement": detect_client(cwd),
        "start_source": input_data.get("source", ""),
        "model": input_data.get("model", ""),
    })


def handle_tool_use(input_data: dict) -> None:
    cwd = os.environ.get("CLAUDE_PROJECT_DIR", "")
    write_entry({
        "event": "tool_use",
        "session_id": input_data.get("session_id", ""),
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "tool_name": input_data.get("tool_name", ""),
        "client_engagement": detect_client(cwd),
    })


def handle_tool_failure(input_data: dict) -> None:
    cwd = os.environ.get("CLAUDE_PROJECT_DIR", "")
    write_entry({
        "event": "tool_failure",
        "session_id": input_data.get("session_id", ""),
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "tool_name": input_data.get("tool_name", ""),
        "is_interrupt": bool(input_data.get("is_interrupt", False)),
        "client_engagement": detect_client(cwd),
    })


def handle_permission_denied(input_data: dict) -> None:
    cwd = os.environ.get("CLAUDE_PROJECT_DIR", "")
    write_entry({
        "event": "permission_denied",
        "session_id": input_data.get("session_id", ""),
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "tool_name": input_data.get("tool_name", ""),
        "client_engagement": detect_client(cwd),
    })


def handle_session_checkpoint(input_data: dict) -> None:
    cwd = os.environ.get("CLAUDE_PROJECT_DIR", "")
    write_entry({
        "event": "session_checkpoint",
        "session_id": input_data.get("session_id", ""),
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "client_engagement": detect_client(cwd),
    })


def handle_session_end(input_data: dict) -> None:
    cwd = os.environ.get("CLAUDE_PROJECT_DIR", "")
    entry = {
        "event": "session_end",
        "session_id": input_data.get("session_id", ""),
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "user": getpass.getuser(),
        "hostname": platform.node(),
        "project_dir": cwd,
        "client_engagement": detect_client(cwd),
        "end_reason": input_data.get("reason", ""),
    }
    write_entry(entry)


HANDLERS = {
    "session_start": handle_session_start,
    "tool_use": handle_tool_use,
    "tool_failure": handle_tool_failure,
    "permission_denied": handle_permission_denied,
    "session_checkpoint": handle_session_checkpoint,
    "session_end": handle_session_end,
}


def main() -> None:
    if len(sys.argv) < 2:
        sys.exit(0)

    event_type = sys.argv[1]
    handler = HANDLERS.get(event_type)
    if handler is None:
        sys.exit(0)

    try:
        input_data = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError):
        input_data = {}

    handler(input_data)
    sys.exit(0)


if __name__ == "__main__":
    try:
        main()
    except Exception:
        sys.exit(0)
