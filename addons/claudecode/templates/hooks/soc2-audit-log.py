#!/usr/bin/env python3
"""
Claude Code Hook: SOC 2 Audit Logging

Metadata-only session audit trail across four Claude Code hook events:
  session_start   — logged at session/resume start
  tool_use        — logged after each tool invocation (metadata only)
  session_checkpoint — logged on stop/pause
  session_end     — logged at session termination with cost summary

SOC 2 Trust Services Criteria coverage:
  CC6.1 (Access Controls), CC6.2 (Access Restriction),
  CC7.2 (Monitoring), CC7.3 (Change Detection), CC8.1 (Change Management)

Exit codes:
  0 — always (fail-open: logging failures never block developer work)

Configuration via environment variables:
  SOC2_CLIENT_DIR_PATTERN — regex for client name extraction (default: .*/clients/([^/]*)/.*  )
  CLAUDE_AUDIT_DIR        — output directory (default: ~/.claude/audit)
"""

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
    """Append a JSON entry to the monthly audit file. Never raises."""
    try:
        AUDIT_DIR.mkdir(parents=True, exist_ok=True)
        with open(AUDIT_FILE, "a") as f:
            f.write(json.dumps(entry) + "\n")
    except OSError:
        pass


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
    }
    session_data = input_data.get("session", {})
    if session_data:
        entry["estimated_cost_usd"] = session_data.get("costUSD", 0)
        entry["input_tokens"] = session_data.get("inputTokens", 0)
        entry["output_tokens"] = session_data.get("outputTokens", 0)
    write_entry(entry)


HANDLERS = {
    "session_start": handle_session_start,
    "tool_use": handle_tool_use,
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
