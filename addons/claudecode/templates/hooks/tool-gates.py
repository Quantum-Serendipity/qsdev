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

import sys

# Keep this first: Python puts the script's own directory at the front of
# sys.path, so a module planted beside this hook (json.py, re.py, a .pyc, a
# package directory) would replace the stdlib module the hook imports and
# could make it allow everything. -P, -I and PYTHONSAFEPATH leave the
# directory out already.
if __name__ == "__main__" and not (getattr(sys.flags, "safe_path", False) or sys.flags.isolated):
    del sys.path[0]

import fnmatch
import importlib.util
import json
import os

# Shared hook library (.claude/hooks/_qsdev_hooklib.py: audit log, deadline
# watchdog, interpreter floor). It is loaded by explicit path, so this
# directory never goes back on sys.path, and without bytecode, so no
# __pycache__ lands in the project. Missing or broken, it blocks the call
# (fail closed); below the minimum Python, its import exits 2.
sys.dont_write_bytecode = True
try:
    _lib_spec = importlib.util.spec_from_file_location(
        "_qsdev_hooklib", os.path.join(os.path.dirname(os.path.abspath(__file__)), "_qsdev_hooklib.py"))
    lib = importlib.util.module_from_spec(_lib_spec)
    _lib_spec.loader.exec_module(lib)
    # What this hook uses: an empty or stale library blocks here, not with an
    # AttributeError (exit 1, which Claude Code lets through) mid-evaluation.
    lib.arm_deadline, lib.audit_log  # noqa: B018
except Exception as _exc:
    print(f"tool-gates: hook library unavailable ({_exc}); blocking (fail closed)", file=sys.stderr)
    sys.exit(2)

# Internal deadline (lib.arm_deadline): the hook's registered settings.json
# timeout minus 2s, so the watchdog blocks before Claude Code's timeout lets
# the call through.
_HOOK_DEADLINE_S = 8

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


def main() -> None:
    lib.arm_deadline("tool-gates", _HOOK_DEADLINE_S)
    try:
        input_data = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError) as e:
        lib.audit_log({"event": "parse_error", "hook": "tool-gates", "error": str(e)})
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
        lib.audit_log({
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
        lib.audit_log({
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
    lib.audit_log(entry)
    sys.exit(0)


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(f"tool gates error: {e}", file=sys.stderr)
        sys.exit(2)
