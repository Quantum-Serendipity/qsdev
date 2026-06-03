#!/usr/bin/env python3
"""
Claude Code PreToolUse Hook: File Boundary Enforcement

Restricts Write/Edit/Read file operations to the current project directory
tree and configured safe paths. Prevents cross-project access and path
traversal attacks.

Exit codes:
  0 — allow or deny (with JSON on stdout for deny)
  2 — hook error (fail-closed)

Configuration via environment variables:
  FILE_BOUNDARY_SAFE_PATHS  — comma-separated paths exempt from boundary check
                              (default: /tmp)
  FILE_BOUNDARY_STRICT_MODE — set to "true" to deny ALL out-of-project access
                              including safe paths
"""

import json
import os
import sys
from datetime import datetime, timezone
from pathlib import Path

SAFE_PATHS: list[str] = [
    p.strip()
    for p in os.environ.get("FILE_BOUNDARY_SAFE_PATHS", "/tmp").split(",")
    if p.strip()
]
# Expand ~ in safe paths.
SAFE_PATHS = [os.path.expanduser(p) for p in SAFE_PATHS]

STRICT_MODE: bool = os.environ.get("FILE_BOUNDARY_STRICT_MODE", "").lower() == "true"

AUDIT_LOG: Path = Path(
    os.environ.get("CLAUDE_PROJECT_DIR", ".")
) / ".claude" / "logs" / "hook-audit.jsonl"

BLOCKED_PREFIXES: tuple[str, ...] = (
    "/proc/self/root",
    "/dev/fd/",
)


def audit_log(entry: dict) -> None:
    """Append a JSON entry to the audit log. Never raises."""
    try:
        AUDIT_LOG.parent.mkdir(parents=True, exist_ok=True)
        entry["timestamp"] = datetime.now(timezone.utc).isoformat()
        with open(AUDIT_LOG, "a") as f:
            f.write(json.dumps(entry) + "\n")
    except OSError:
        pass  # Audit logging must not interrupt hook decisions.


def deny(reason: str, target: str, cwd: str) -> None:
    """Output structured deny JSON and exit."""
    audit_log({
        "event": "deny",
        "hook": "file-boundary",
        "target": target,
        "cwd": cwd,
        "reason": reason,
    })
    result = {
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "deny",
            "permissionDecisionReason": reason,
        }
    }
    print(json.dumps(result))
    sys.exit(0)


def is_safe_path(resolved: str) -> bool:
    """Check if the resolved path falls within a configured safe path."""
    if STRICT_MODE:
        return False
    for safe in SAFE_PATHS:
        try:
            safe_resolved = os.path.realpath(safe)
            if resolved == safe_resolved or resolved.startswith(safe_resolved + "/"):
                return True
        except (OSError, ValueError):
            pass  # Skip invalid safe paths; continue checking others.
    return False


def main() -> None:
    try:
        input_data = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError) as e:
        audit_log({"event": "parse_error", "hook": "file-boundary", "error": str(e)})
        print(f"file boundary error: {e}", file=sys.stderr)
        sys.exit(2)

    tool_name = input_data.get("tool_name", "")
    tool_input = input_data.get("tool_input", {})
    file_path = tool_input.get("file_path", "")

    if tool_name not in ("Write", "Edit", "MultiEdit", "Read"):
        sys.exit(0)

    if not file_path:
        sys.exit(0)

    cwd = os.environ.get("CLAUDE_PROJECT_DIR", "")
    if not cwd:
        sys.exit(0)

    # Block known traversal bypass vectors.
    for prefix in BLOCKED_PREFIXES:
        if file_path.startswith(prefix):
            deny(
                f"File operation blocked: path uses disallowed prefix {prefix}. "
                f"Target: {file_path}",
                file_path, cwd,
            )

    # Canonicalize paths.
    try:
        cwd_resolved = os.path.realpath(cwd)
    except (OSError, ValueError):
        cwd_resolved = cwd

    # Use os.path.realpath for the target — handles symlinks and ../ sequences.
    # For non-existent paths (Write targets), realpath resolves what it can.
    try:
        target_resolved = os.path.realpath(file_path)
    except (OSError, ValueError):
        target_resolved = os.path.normpath(os.path.join(cwd_resolved, file_path))

    # Check if target is within the project directory.
    if target_resolved == cwd_resolved or target_resolved.startswith(cwd_resolved + "/"):
        audit_log({
            "event": "allow",
            "hook": "file-boundary",
            "target": file_path,
            "resolved": target_resolved,
        })
        sys.exit(0)

    # Target is outside project — check safe paths.
    if is_safe_path(target_resolved):
        audit_log({
            "event": "allow_safe_path",
            "hook": "file-boundary",
            "target": file_path,
            "resolved": target_resolved,
        })
        sys.exit(0)

    deny(
        f"File operation targets path outside the current project directory. "
        f"Target: {file_path}, Project: {cwd}",
        file_path, cwd,
    )


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(f"file boundary error: {e}", file=sys.stderr)
        sys.exit(2)
