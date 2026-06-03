#!/usr/bin/env python3
"""
Claude Code PreToolUse Hook: Credential Scanning

Scans Write/Edit/MultiEdit content for hardcoded credentials, API keys,
private keys, and other secrets. Blocks writes containing detected secrets
with actionable feedback.

Exit codes:
  0 — allow or deny (with JSON on stdout for deny)
  2 — hook error (fail-closed, blocks the operation)

Configuration via environment variables:
  CREDENTIAL_SCAN_EXTRA_PATTERNS — comma-separated regex patterns to add

Security invariant:
  - Fail-closed: any uncaught exception blocks the operation (exit 2).
  - Uses only stdlib (no pip dependencies).
"""

import json
import os
import re
import sys
from datetime import datetime, timezone
from pathlib import Path

DEFAULT_PATTERNS: list[str] = [
    # AWS access key IDs
    r'AKIA[0-9A-Z]{16}',
    # AWS secret/session token assignments
    r'(?i)aws[_-]?(secret[_-]?access[_-]?key|session[_-]?token)\s*[=:]\s*[A-Za-z0-9/+=]{20,}',
    # GitHub personal access tokens and secrets
    r'gh[ps]_[A-Za-z0-9_]{36,}',
    # GitLab personal access tokens
    r'glpat-[A-Za-z0-9_-]{20,}',
    # Generic API key assignments
    r"""["']?[Aa](pi|PI)[_-]?[Kk](ey|EY)["']?\s*[=:]\s*["'][A-Za-z0-9_-]{20,}["']""",
    # PEM private keys
    r'-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----',
    # JWT tokens (three base64url segments)
    r'eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}',
    # Database connection strings with credentials
    r"""(mongodb(\+srv)?|postgres(ql)?|mysql|redis)://[^\s"':]+:[^\s"'@]+@[^\s"']{5,}""",
    # Slack API tokens
    r'xox[bprase]-[A-Za-z0-9-]{10,}',
    # Stripe secret keys
    r'sk_(live|test)_[A-Za-z0-9]{20,}',
    # SendGrid API keys
    r'SG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}',
    # Generic secret/password assignments
    r"""(?i)(password|passwd|secret|token|credential)\s*[=:]\s*["'][^\s"']{8,}["']""",
]

KNOWN_EXAMPLES: set[str] = {
    'AKIAIOSFODNN7EXAMPLE',
    'AKIAI44QH8DHBEXAMPLE',
    'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY',
}

PLACEHOLDER_INDICATORS: tuple[str, ...] = (
    'EXAMPLE', 'PLACEHOLDER', 'YOUR_', 'REPLACE', 'CHANGEME',
    'INSERT_', 'TODO', 'XXXX', 'sample', 'dummy', 'test_key',
)

BINARY_EXTENSIONS: set[str] = {
    '.png', '.jpg', '.jpeg', '.gif', '.ico', '.bmp', '.tiff', '.webp',
    '.woff', '.woff2', '.ttf', '.eot', '.otf',
    '.pdf', '.zip', '.tar', '.gz', '.bz2', '.xz', '.7z', '.rar',
    '.bin', '.exe', '.dll', '.so', '.dylib', '.o', '.a',
    '.pyc', '.class', '.jar', '.war',
    '.mp3', '.mp4', '.wav', '.avi', '.mov', '.mkv',
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
        pass


def get_patterns() -> list[re.Pattern]:
    """Compile default + extra patterns from environment."""
    raw = list(DEFAULT_PATTERNS)
    extra = os.environ.get("CREDENTIAL_SCAN_EXTRA_PATTERNS", "")
    if extra:
        for p in extra.split(","):
            p = p.strip()
            if p:
                raw.append(p)
    compiled = []
    for p in raw:
        try:
            compiled.append(re.compile(p))
        except re.error:
            pass
    return compiled


def is_placeholder(matched_text: str) -> bool:
    """Check if matched text is a known example or placeholder."""
    if matched_text in KNOWN_EXAMPLES:
        return True
    upper = matched_text.upper()
    return any(indicator in upper for indicator in PLACEHOLDER_INDICATORS)


def main() -> None:
    try:
        input_data = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError) as e:
        audit_log({"event": "parse_error", "hook": "credential-scan", "error": str(e)})
        print(f"credential scan error: {e}", file=sys.stderr)
        sys.exit(2)

    tool_name = input_data.get("tool_name", "")
    tool_input = input_data.get("tool_input", {})

    if tool_name == "Write":
        content = tool_input.get("content", "")
        file_path = tool_input.get("file_path", "")
    elif tool_name in ("Edit", "MultiEdit"):
        content = tool_input.get("new_string", "")
        file_path = tool_input.get("file_path", "")
    else:
        sys.exit(0)

    if file_path:
        ext = os.path.splitext(file_path)[1].lower()
        if ext in BINARY_EXTENSIONS:
            sys.exit(0)

    if not content:
        sys.exit(0)

    patterns = get_patterns()
    for pattern in patterns:
        match = pattern.search(content)
        if match:
            matched_text = match.group()
            if is_placeholder(matched_text):
                continue

            redacted = matched_text[:8] + "..." if len(matched_text) > 8 else matched_text

            audit_log({
                "event": "deny",
                "hook": "credential-scan",
                "tool": tool_name,
                "file": file_path,
                "pattern": pattern.pattern,
                "redacted_match": redacted,
            })

            result = {
                "hookSpecificOutput": {
                    "hookEventName": "PreToolUse",
                    "permissionDecision": "deny",
                    "permissionDecisionReason": (
                        f"Credential/secret detected in {tool_name} content for "
                        f"{file_path}. Matched pattern: {pattern.pattern} "
                        f"(redacted: {redacted}). Use environment variables "
                        f"or a secrets manager instead of hardcoding credentials."
                    ),
                }
            }
            print(json.dumps(result))
            sys.exit(0)

    audit_log({
        "event": "allow",
        "hook": "credential-scan",
        "tool": tool_name,
        "file": file_path,
    })
    sys.exit(0)


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(f"credential scan error: {e}", file=sys.stderr)
        sys.exit(2)
