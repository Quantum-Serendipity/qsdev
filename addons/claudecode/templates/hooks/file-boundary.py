#!/usr/bin/env python3
"""
Claude Code PreToolUse Hook: File Boundary Enforcement

Restricts Write/Edit/MultiEdit/NotebookEdit/Read/Grep/Glob file operations
to the current project directory tree and configured safe paths. Prevents
cross-project access and path traversal attacks.

Paths are resolved the way Claude Code's file tools resolve them: a leading ~
is the home directory and relative paths are relative to the session's cwd
(the hook input's `cwd`). Reads (Read, Grep, Glob) may also reach dependency
sources outside the project — the Go module cache, Cargo registry, rustup
toolchains, Maven/Gradle caches, /nix/store and Claude Code plugins — so the
agent can follow an LSP go-to-definition into a library, as may the
directories .qsdev.yaml lists under hooks.file_boundary.extra_read_paths. Those
locations are never writable through this hook.

Scope: this hook sees Claude Code's file tools only. Shell commands (Bash,
PowerShell, Monitor) can still read outside the project; use the sandbox to
confine them.

Exit codes:
  0 — allow or deny (with JSON on stdout for deny)
  2 — hook error (fail-closed)

Configuration via environment variables:
  FILE_BOUNDARY_SAFE_PATHS  — comma-separated paths exempt from boundary check
                              (default: /tmp)
  FILE_BOUNDARY_EXTRA_READ_PATHS — comma-separated absolute or ~/ paths that
                              Read, Grep and Glob may reach (read-only);
                              generated into settings.json "env" from
                              .qsdev.yaml hooks.file_boundary.extra_read_paths.
                              Relative paths, the filesystem root and the home
                              directory or an ancestor of it are ignored.
  FILE_BOUNDARY_STRICT_MODE — set to "true" to deny ALL out-of-project access
                              including safe paths, dependency caches and
                              extra read paths
"""

from __future__ import annotations

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

# Tools this hook inspects, and the tool_input key that names the target file.
# The hook's settings.json matcher must list exactly these tools
# (hook_registry.go; kept in sync by TestHookMatchersCoverScriptTools).
PATH_KEYS: dict[str, str] = {
    "Write": "file_path",
    "Edit": "file_path",
    "MultiEdit": "file_path",
    "NotebookEdit": "notebook_path",
    "Read": "file_path",
    "Grep": "path",
    "Glob": "path",
}

# Tools that only read; they may also reach the dependency caches and the
# configured extra read paths.
READ_ONLY_TOOLS: frozenset[str] = frozenset({"Read", "Grep", "Glob"})

_GLOB_CHARS = set("*?[{")


def _dependency_source_dirs() -> list[str]:
    """Directories holding third-party sources an LSP go-to-definition lands
    in. Read-only: Write/Edit there is still denied."""
    home = os.path.expanduser("~")
    gopath = os.environ.get("GOPATH", "").split(os.pathsep)[0] or os.path.join(home, "go")
    cargo = os.environ.get("CARGO_HOME") or os.path.join(home, ".cargo")
    dirs = [
        os.environ.get("GOMODCACHE") or os.path.join(gopath, "pkg", "mod"),
        os.environ.get("GOROOT", ""),
        os.path.join(cargo, "registry"),
        os.path.join(cargo, "git"),
        os.path.join(os.environ.get("RUSTUP_HOME") or os.path.join(home, ".rustup"), "toolchains"),
        os.path.join(home, ".m2", "repository"),
        os.path.join(home, ".gradle", "caches"),
        os.path.join(home, ".claude", "plugins"),
        "/nix/store",
    ]
    return [d for d in dirs if d]


def _extra_read_paths() -> list[str]:
    """The configured extra read-only directories. An entry that is not
    absolute after ~ expansion, or that resolves to the filesystem root, the
    home directory or an ancestor of it (which would lift the read boundary),
    grants nothing."""
    home = os.path.realpath(os.path.expanduser("~"))
    paths = []
    for raw in os.environ.get("FILE_BOUNDARY_EXTRA_READ_PATHS", "").split(","):
        p = os.path.expanduser(raw.strip())
        if not p or not os.path.isabs(p):
            continue
        try:
            real = os.path.realpath(p)
        except (OSError, ValueError):
            continue
        if os.path.dirname(real) == real or real == home or home.startswith(real + os.sep):
            continue
        paths.append(p)
    return paths


BLOCKED_PREFIXES: tuple[str, ...] = (
    "/proc/self/root",
    "/dev/fd/",
)


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


def is_safe_path(resolved: str, read_only: bool) -> bool:
    """Check if the resolved path falls within a configured safe path, or,
    for a read-only tool, within a dependency source directory or a
    configured extra read path."""
    if STRICT_MODE:
        return False
    read_dirs = _dependency_source_dirs() + _extra_read_paths() if read_only else []
    for safe in SAFE_PATHS + read_dirs:
        try:
            safe_resolved = os.path.realpath(safe)
            if resolved == safe_resolved or resolved.startswith(safe_resolved + "/"):
                return True
        except (OSError, ValueError):
            pass  # Skip invalid safe paths; continue checking others.
    return False


def _glob_base(pattern: str) -> str:
    """The directory a Glob pattern can reach: its literal leading segments,
    plus one `..` for every `..` after the first wildcard (`src/**/../..`
    climbs out of src), so a relative pattern cannot walk out unseen."""
    base: list[str] = []
    climbs = 0
    wild = False
    for part in pattern.split("/"):
        wild = wild or bool(_GLOB_CHARS & set(part))
        if not wild:
            base.append(part)
        elif part == "..":
            climbs += 1
    base.extend([".."] * climbs)
    if pattern.startswith("/"):
        return "/".join(base) or "/"
    return "/".join(base) or "."


def target_path(tool_name: str, tool_input: dict, session_cwd: str) -> str:
    """The path a tool call reads or writes. Grep and Glob search their `path`
    (default: the session cwd); Glob's pattern is resolved against that path,
    since a pattern such as ../**/* or ~/.ssh/* reaches beyond it."""
    path = tool_input.get(PATH_KEYS[tool_name]) or ""
    if tool_name not in ("Grep", "Glob"):
        return path
    path = path or session_cwd
    if tool_name == "Glob":
        base = _glob_base(tool_input.get("pattern") or "")
        path = base if base.startswith(("/", "~")) else os.path.join(path, base)
    return path


def main() -> None:
    try:
        input_data = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError) as e:
        audit_log({"event": "parse_error", "hook": "file-boundary", "error": str(e)})
        print(f"file boundary error: {e}", file=sys.stderr)
        sys.exit(2)

    tool_name = input_data.get("tool_name", "")
    tool_input = input_data.get("tool_input") or {}
    if tool_name not in PATH_KEYS:
        sys.exit(0)

    cwd = os.environ.get("CLAUDE_PROJECT_DIR", "")
    if not cwd:
        sys.exit(0)
    # The directory Claude Code resolves relative tool paths against.
    session_cwd = input_data.get("cwd") or cwd

    file_path = target_path(tool_name, tool_input, session_cwd)
    if not file_path:
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

    # Expand ~ as Claude Code does. A path still led by ~ (an unknown user)
    # or a shell variable cannot be resolved the way the tool will, so deny.
    expanded = os.path.expanduser(file_path)
    if expanded.startswith(("~", "$")):
        deny(
            f"File operation blocked: cannot resolve {file_path} the way the tool will.",
            file_path, cwd,
        )
    # Relative paths are relative to the session's cwd, not this process's.
    if not os.path.isabs(expanded):
        expanded = os.path.join(session_cwd, expanded)

    # Use os.path.realpath for the target — handles symlinks and ../ sequences.
    # For non-existent paths (Write targets), realpath resolves what it can.
    try:
        target_resolved = os.path.realpath(expanded)
    except (OSError, ValueError):
        target_resolved = os.path.normpath(expanded)

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
    if is_safe_path(target_resolved, tool_name in READ_ONLY_TOOLS):
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
