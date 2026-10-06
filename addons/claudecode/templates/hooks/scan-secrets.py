#!/usr/bin/env python3
"""
Claude Code PreToolUse Hook: Credential Scanning

Scans Write/Edit/MultiEdit/NotebookEdit content for hardcoded credentials,
API keys, private keys, and other secrets. Blocks writes containing detected
secrets with actionable feedback.

Exit codes:
  0 — allow or deny (with JSON on stdout for deny)
  2 — hook error (fail-closed, blocks the operation)

Configuration via environment variables:
  CREDENTIAL_SCAN_EXTRA_PATTERNS — comma-separated regex patterns to add

Security invariant:
  - Fail-closed: any uncaught exception blocks the operation (exit 2).
  - Uses only stdlib (no pip dependencies).
"""

import sys

# Keep this first: Python puts the script's own directory at the front of
# sys.path, so a module planted beside this hook (json.py, re.py, a .pyc, a
# package directory) would replace the stdlib module the hook imports and
# could make it allow everything. -P, -I and PYTHONSAFEPATH leave the
# directory out already.
if __name__ == "__main__" and not (getattr(sys.flags, "safe_path", False) or sys.flags.isolated):
    del sys.path[0]

import importlib.util
import json
import os
import re

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
    print(f"credential-scan: hook library unavailable ({_exc}); blocking (fail closed)", file=sys.stderr)
    sys.exit(2)

# Internal deadline (lib.arm_deadline): the hook's registered settings.json
# timeout minus 2s, so the watchdog blocks before Claude Code's timeout lets
# the call through.
_HOOK_DEADLINE_S = 8

# Checked in every file: the credential token shapes of the qsdev canon
# (internal/secrets ValuePatterns, in canon order), then the scan-only
# assignment heuristics (addons/claudecode ScanHeuristicPatterns). Kept equal to
# the Go side by TestSecretPatterns_MatchPythonHook; every canon entry must
# compile in both Go RE2 and Python re.
DEFAULT_PATTERNS: list[str] = [
    # AWS access key IDs (long-term AKIA and temporary STS ASIA)
    r'(AKIA|ASIA)[0-9A-Z]{16}',
    # GitHub classic, OAuth, user-to-server, server and refresh tokens
    r'gh[pousr]_[A-Za-z0-9_]{36,}',
    # GitHub fine-grained personal access tokens
    r'github_pat_[A-Za-z0-9_]{22,}',
    # GitLab personal access tokens
    r'glpat-[A-Za-z0-9_-]{20,}',
    # Stripe secret keys
    r'sk_(live|test)_[A-Za-z0-9]{20,}',
    # Stripe restricted keys
    r'rk_(live|test)_[A-Za-z0-9]{20,}',
    # npm access tokens
    r'npm_[A-Za-z0-9]{36,}',
    # JWT tokens (three base64url segments; the signature may be short or empty)
    r'eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]*',
    # PEM private keys, including encrypted keys and PGP secret key blocks
    r'-----BEGIN [A-Z0-9 ]*PRIVATE KEY(?: BLOCK)?-----',
    # Azure storage account keys in connection strings
    r'AccountKey=[A-Za-z0-9+/=]{44,}',
    # Google API keys
    r'AIza[0-9A-Za-z_-]{35}',
    # Database connection strings with credentials
    r"""(mongodb(\+srv)?|postgres(ql)?|mysql|redis)://[^\s"':]+:[^\s"'@]+@[^\s"']{5,}""",
    # HashiCorp Vault service, batch and recovery tokens
    r'hv[sbr]\.[A-Za-z0-9_-]{24,}',
    # Slack API tokens
    r'xox[bprase]-[A-Za-z0-9-]{10,}',
    # SendGrid API keys
    r'SG\.[A-Za-z0-9_-]{22}\.[A-Za-z0-9_-]{43}',
    # Anthropic API and admin keys
    r'sk-ant-[A-Za-z0-9_-]{20,}',
    # OpenAI API keys (project, service-account, admin and legacy)
    r'sk-(proj|svcacct|admin)-[A-Za-z0-9_-]{20,}|sk-[A-Za-z0-9]{20}T3BlbkFJ[A-Za-z0-9]{20}',
    # PyPI API tokens
    r'pypi-[A-Za-z0-9_-]{50,}',
    # Slack incoming webhooks
    r'https://hooks\.slack\.com/services/T[A-Za-z0-9]+/B[A-Za-z0-9]+/[A-Za-z0-9]+',
    # Docker Hub personal access tokens
    r'dckr_pat_[A-Za-z0-9_-]{20,}',
    # Hugging Face access tokens
    r'hf_[A-Za-z0-9]{30,}',
    # age secret keys
    r'AGE-SECRET-KEY-1[0-9A-Z]{58}',
    # Heuristic: AWS secret/session token assignments
    r'(?i)aws[_-]?(secret[_-]?access[_-]?key|session[_-]?token)\s*[=:]\s*[A-Za-z0-9/+=]{20,}',
    # Heuristic: generic API key assignments
    r"""["']?[Aa](pi|PI)[_-]?[Kk](ey|EY)["']?\s*[=:]\s*["'][A-Za-z0-9_-]{20,}["']""",
    # Heuristic: generic secret/password assignments; the key may be quoted (JSON)
    r"""(?i)(password|passwd|secret|token|credential)["']?\s*[=:]\s*["'][^\s"']{8,}["']""",
]

# Unquoted `KEY=value` / `key: value` assignments, checked only in dotenv and
# config files (CONFIG_FILE): in source code the same shape is an ordinary
# variable assignment (`token = get_token()`). The key must end with the
# secret word, so keys that only name or point at a secret (secretName,
# tokenUrl, PASSWORD_FILE) are not flagged, and the value must sit on the same
# line and may not start with a variable reference or template
# (`${DB_PASSWORD}`, `<password>`, `%(pw)s`).
CONFIG_PATTERNS: list[str] = [
    r"""(?im)^[ \t]*(export[ \t]+)?[A-Za-z0-9_.-]*(password|passwd|secret|token|credential|api[_-]?key|access[_-]?key)["']?[ \t]*[=:][ \t]*[^\s"'#$<%{][^\s"'#]{7,}""",
]

# Dotenv and configuration files, by base name.
CONFIG_FILE = re.compile(
    r"""(?i)(^\.env(\..*)?$|\.env$|\.(ya?ml|properties|ini|toml|cfg|conf|config)$|^\.(npmrc|pypirc|netrc)$|^credentials$)"""
)

KNOWN_EXAMPLES: set[str] = {
    'AKIAIOSFODNN7EXAMPLE',
    'AKIAI44QH8DHBEXAMPLE',
    'wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY',
}

# Compared against the upper-cased match, so every indicator must be upper case.
PLACEHOLDER_INDICATORS: tuple[str, ...] = (
    'EXAMPLE', 'PLACEHOLDER', 'YOUR_', 'REPLACE', 'CHANGEME',
    'INSERT_', 'TODO', 'XXXX', 'SAMPLE', 'DUMMY', 'TEST_KEY',
)

def get_patterns(file_path: str) -> list[re.Pattern]:
    """Compile default + extra patterns from environment, plus the config
    assignment patterns when file_path is a dotenv or config file."""
    raw = list(DEFAULT_PATTERNS)
    if CONFIG_FILE.search(os.path.basename(file_path)):
        raw.extend(CONFIG_PATTERNS)
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
        except re.error as err:
            lib.audit_log({"event": "pattern_compile_error", "hook": "credential-scan", "pattern": p, "error": str(err)})
    return compiled


def is_placeholder(matched_text: str) -> bool:
    """Check if matched text is a known example or placeholder."""
    if matched_text in KNOWN_EXAMPLES:
        return True
    upper = matched_text.upper()
    return any(indicator in upper for indicator in PLACEHOLDER_INDICATORS)


# Tools whose written content this hook scans. The hook's settings.json matcher
# must list exactly these tools (hook_registry.go; kept in sync by
# TestHookMatchersCoverScriptTools).
SCANNED_TOOLS: tuple[str, ...] = ("Write", "Edit", "MultiEdit", "NotebookEdit")


def written_content(tool_name: str, tool_input: dict) -> tuple[str, str]:
    """Return (content, file_path) for the text a file-writing tool call adds:
    Write's content, Edit's new_string, every MultiEdit edits[].new_string, or
    NotebookEdit's new_source."""
    if tool_name == "Write":
        return tool_input.get("content", "") or "", tool_input.get("file_path", "")
    if tool_name == "Edit":
        return tool_input.get("new_string", "") or "", tool_input.get("file_path", "")
    if tool_name == "MultiEdit":
        edits = tool_input.get("edits") or []
        parts = [e.get("new_string", "") or "" for e in edits if isinstance(e, dict)]
        return "\n".join(parts), tool_input.get("file_path", "")
    if tool_name == "NotebookEdit":
        return tool_input.get("new_source", "") or "", tool_input.get("notebook_path", "")
    return "", ""


def main() -> None:
    lib.arm_deadline("credential-scan", _HOOK_DEADLINE_S)
    try:
        input_data = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError) as e:
        lib.audit_log({"event": "parse_error", "hook": "credential-scan", "error": str(e)})
        print(f"credential scan error: {e}", file=sys.stderr)
        sys.exit(2)

    tool_name = input_data.get("tool_name", "")
    tool_input = input_data.get("tool_input", {})

    if tool_name not in SCANNED_TOOLS:
        sys.exit(0)
    content, file_path = written_content(tool_name, tool_input)

    # Tool input is always text, so everything is scanned: neither a file name
    # (notes.png) nor a NUL character in the content can switch the scan off.
    if not content:
        sys.exit(0)

    for pattern in get_patterns(file_path):
        # Every match counts: a placeholder earlier in the file (the AWS docs'
        # example key) must not hide a real secret of the same kind below it.
        for match in pattern.finditer(content):
            matched_text = match.group()
            if is_placeholder(matched_text):
                continue

            redacted = matched_text[:8] + "..." if len(matched_text) > 8 else matched_text

            lib.audit_log({
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

    lib.audit_log({
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
