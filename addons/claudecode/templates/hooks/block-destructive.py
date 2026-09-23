#!/usr/bin/env python3
"""
Claude Code PreToolUse Hook: Destructive Operation Prevention

Blocks dangerous Bash commands across six categories: filesystem destruction,
git force operations, database destruction, remote code execution, consulting
cross-environment protection, and infrastructure destruction.

Complements static deny rules in settings.json with dynamic, context-aware
pattern matching that catches variable-expanded, shell-wrapped, and
consulting-specific patterns that glob rules cannot express.

Exit codes:
  0 — allow or deny (with JSON on stdout for deny)
  2 — hook error (fail-closed)

Configuration via environment variables:
  DESTRUCTIVE_PREVENTION_PROTECTED_BRANCHES — comma-separated (default: main,master,production,release)
  DESTRUCTIVE_PREVENTION_PRODUCTION_HOSTS   — comma-separated substrings (default: prod,production,live,staging)
"""

import json
import os
import posixpath
import re
import shlex
import sys
from datetime import datetime, timezone
from pathlib import Path

PROTECTED_BRANCHES: list[str] = [
    b.strip()
    for b in os.environ.get(
        "DESTRUCTIVE_PREVENTION_PROTECTED_BRANCHES",
        "main,master,production,release",
    ).split(",")
    if b.strip()
]

PRODUCTION_HOST_PATTERNS: list[str] = [
    p.strip()
    for p in os.environ.get(
        "DESTRUCTIVE_PREVENTION_PRODUCTION_HOSTS",
        "prod,production,live,staging",
    ).split(",")
    if p.strip()
]

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


def deny(category: str, reason: str, remediation: str, command: str) -> None:
    """Output structured deny JSON and exit."""
    audit_log({
        "event": "deny",
        "hook": "destructive-prevention",
        "category": category,
        "command": command,
        "reason": reason,
    })
    result = {
        "hookSpecificOutput": {
            "hookEventName": "PreToolUse",
            "permissionDecision": "deny",
            "permissionDecisionReason": (
                f"[{category}] {reason} Remediation: {remediation}"
            ),
        }
    }
    print(json.dumps(result))
    sys.exit(0)


# ---------------------------------------------------------------------------
# Category 1: Filesystem destruction
# ---------------------------------------------------------------------------

# rm is handled by check_rm_danger, which parses its argv rather than pattern
# matching the raw string.
FILESYSTEM_PATTERNS: list[tuple[re.Pattern, str, str]] = [
    (
        re.compile(r':\(\)\{\s*:\|:&\s*\};:'),
        "Fork bomb detected.",
        "Do not run fork bombs.",
    ),
    (
        re.compile(r'\bdd\b.*\bof=/dev/[a-z]'),
        "Direct device write detected.",
        "Do not write directly to block devices.",
    ),
    (
        re.compile(r'\bmkfs\b'),
        "Filesystem creation on device detected.",
        "Do not format devices from within a project.",
    ),
    (
        re.compile(r'>\s*/dev/sd[a-z]'),
        "Direct device write via redirection detected.",
        "Do not write directly to block devices.",
    ),
]


# Whole rm targets that mean "the filesystem root" or "the home directory",
# after quote removal and normalisation by _normalize_rm_target.
_DANGEROUS_RM_TARGETS: set[str] = {"/", "~", "$HOME", "${HOME}"}


def _command_tokens(command: str) -> list[list[str]]:
    """Tokenize `command` into per-command token lists, splitting on unquoted
    shell operators (; & | ( ), backticks and newlines). Quotes are removed, so `"$HOME"`
    and `'/'` become `$HOME` and `/`. When the command cannot be tokenized
    (unbalanced quotes), fall back to whitespace splitting with quote characters
    stripped so a dangerous target is still recognised."""
    try:
        lex = shlex.shlex(command, posix=True, punctuation_chars=";&|()`\n")
        lex.whitespace = " \t\r"
        lex.whitespace_split = True
        tokens = list(lex)
    except ValueError:
        tokens = [t.strip("'\"") for t in re.split(r"[\s;&|()`]+", command)]
        tokens = [t for t in tokens if t]
        return [tokens]
    commands: list[list[str]] = [[]]
    for tok in tokens:
        if tok and all(c in ";&|()`\n" for c in tok):
            commands.append([])
        else:
            commands[-1].append(tok)
    return [c for c in commands if c]


_HOME_PREFIXES: tuple[str, ...] = ("~", "$HOME", "${HOME}")


def _normalize_rm_target(target: str) -> str:
    """Reduce an rm operand to its base: trailing `/*` globs are dropped and
    the path is normalised (`.`/`..` segments, repeated and trailing slashes),
    so `/*`, `~/`, `$HOME/*`, `//` and `/usr/..` compare equal to `/`, `~`,
    `$HOME`, `/` and `/`. A home-relative target that climbs out of the home
    directory (`~/..`) normalises to the root marker `/`, since it takes the
    home directory (and its siblings) with it."""
    t = target
    while t.endswith("/*"):
        t = t[:-2] or "/"
    if not t:
        return t
    norm = posixpath.normpath(t)
    if norm.startswith("//"):
        norm = "/" + norm.lstrip("/")  # normpath keeps a POSIX leading "//"
    home = next((h for h in _HOME_PREFIXES if t == h or t.startswith(h + "/")), None)
    if home is not None and (norm == "." or norm == ".." or norm.startswith("../")):
        return "/"
    return norm


# How deep check_rm_danger re-parses quoted arguments as nested scripts.
_MAX_RM_NESTING = 4


def _rm_is_dangerous(args: list[str]) -> bool:
    """Whether rm's arguments request a recursive delete of root or home."""
    recursive = False
    end_of_opts = False
    targets: list[str] = []
    for arg in args:
        if not end_of_opts and arg == "--":
            end_of_opts = True
        elif not end_of_opts and arg.startswith("--"):
            recursive = recursive or arg == "--recursive"
        elif not end_of_opts and arg.startswith("-") and len(arg) > 1:
            recursive = recursive or "r" in arg[1:] or "R" in arg[1:]
        else:
            targets.append(arg)
    return recursive and any(
        _normalize_rm_target(t) in _DANGEROUS_RM_TARGETS for t in targets
    )


def check_rm_danger(command: str, _depth: int = 0) -> tuple[str, str] | None:
    """Check for a recursive rm whose target is the filesystem root or the home
    directory, handling combined/long flag forms and quoted targets. Only whole
    targets count: `rm -rf /tmp/build` and `rm ~/notes.txt` are ordinary.

    A quoted argument that could itself be a script (`bash -c "rm -rf /"`,
    `echo "$(rm -rf ~)"`) is re-parsed and checked the same way."""
    for argv in _command_tokens(command):
        for k, tok in enumerate(argv):
            if (
                _depth < _MAX_RM_NESTING
                and tok != command
                and (any(c in tok for c in " \t\n`") or "$(" in tok)
                and (nested := check_rm_danger(tok, _depth + 1))
            ):
                return nested
            if os.path.basename(tok) == "rm" and _rm_is_dangerous(argv[k + 1:]):
                return (
                    "Recursive deletion of root/home directory detected.",
                    "Use targeted rm on specific files or directories within the project.",
                )
    return None


# ---------------------------------------------------------------------------
# Category 2: Git force operations
# ---------------------------------------------------------------------------

def check_git_force(command: str) -> tuple[str, str] | None:
    """Check for destructive git operations targeting protected branches."""
    branch_pattern = "|".join(re.escape(b) for b in PROTECTED_BRANCHES)

    # git push --force / -f to protected branches
    force_push = re.compile(
        rf'\bgit\s+push\s+.*(-f\b|--force\b|--force-with-lease\b).*\b({branch_pattern})\b'
        rf'|\bgit\s+push\s+.*\b({branch_pattern})\b.*(-f\b|--force\b|--force-with-lease\b)'
    )
    # A leading `+` on a refspec forces that ref (`git push origin +main`,
    # `git push origin +HEAD:refs/heads/main`) exactly like --force.
    plus_refspec = re.compile(
        rf'\bgit\s+push\b.*\s\+(?:[^\s:]*:)?(?:refs/heads/)?({branch_pattern})(?![\w./-])'
    )
    if force_push.search(command) or plus_refspec.search(command):
        return (
            "Force push to protected branch detected.",
            "Use a feature branch and PR workflow instead of force-pushing.",
        )

    # git reset --hard
    if re.search(r'\bgit\s+reset\s+--hard\b', command):
        return (
            "Hard reset detected — this discards uncommitted changes.",
            "Use git stash or create a backup branch before resetting.",
        )

    # git clean -f (with any flag combination)
    if re.search(r'\bgit\s+clean\s+(-[a-zA-Z]*f|--force)', command):
        return (
            "Git clean with force detected — removes untracked files permanently.",
            "Review untracked files with git clean -n (dry run) first.",
        )

    # git branch -D (force delete)
    if re.search(r'\bgit\s+branch\s+-D\b', command):
        return (
            "Force branch deletion detected.",
            "Use git branch -d (lowercase) for safe deletion that checks merge status.",
        )

    return None


# ---------------------------------------------------------------------------
# Category 3: Database destruction
# ---------------------------------------------------------------------------

DB_PATTERNS: list[tuple[re.Pattern, str, str]] = [
    (
        re.compile(r'\bDROP\s+(TABLE|DATABASE|SCHEMA)\b', re.IGNORECASE),
        "SQL DROP statement detected.",
        "Use migrations with rollback support instead of raw DROP statements.",
    ),
    (
        re.compile(r'\bTRUNCATE\s+TABLE\b', re.IGNORECASE),
        "SQL TRUNCATE statement detected.",
        "Use targeted DELETE with WHERE clause or migrations instead.",
    ),
]


def check_delete_without_where(command: str) -> tuple[str, str] | None:
    """Detect bare DELETE FROM without WHERE clause."""
    for stmt in command.split(';'):
        match = re.search(r'\bDELETE\s+FROM\s+\S+', stmt, re.IGNORECASE)
        if match and not re.search(r'\bWHERE\b', stmt, re.IGNORECASE):
            return (
                "DELETE FROM without WHERE clause — deletes all rows.",
                "Add a WHERE clause to target specific rows, or use TRUNCATE if intended.",
            )
    return None


# ---------------------------------------------------------------------------
# Category 4: Remote code execution
# ---------------------------------------------------------------------------

RCE_PATTERNS: list[tuple[re.Pattern, str, str]] = [
    (
        re.compile(r'\b(curl|wget)\b.*\|\s*(ba)?sh\b'),
        "Pipe-to-shell pattern detected (curl/wget piped to bash/sh).",
        "Download the script first, review it, then execute.",
    ),
    (
        re.compile(r'\b(curl|wget)\b.*\|\s*sudo\b'),
        "Pipe-to-sudo pattern detected.",
        "Download the script first, review it, then execute with appropriate permissions.",
    ),
]


# ---------------------------------------------------------------------------
# Category 5: Consulting cross-environment
# ---------------------------------------------------------------------------

def check_cross_environment(command: str, cwd: str) -> tuple[str, str] | None:
    """Check for operations targeting production or outside project scope."""
    # SSH/SCP to production hosts
    for pattern in PRODUCTION_HOST_PATTERNS:
        if re.search(rf'\b(ssh|scp)\b.*\b{re.escape(pattern)}\b', command, re.IGNORECASE):
            return (
                f"SSH/SCP to production-like host detected (matched: {pattern}).",
                "Use deployment pipelines instead of direct production access.",
            )

    # Deployment commands targeting production
    deploy_patterns = [
        (r'\b(kubectl|helm|docker)\s+(apply|deploy|push)\b.*\b(prod|production|live)\b',
         "Deployment command targeting production detected."),
        (r'\b(ansible-playbook|terraform\s+apply)\b.*\b(prod|production|live)\b',
         "Infrastructure command targeting production detected."),
    ]
    for pat, reason in deploy_patterns:
        if re.search(pat, command, re.IGNORECASE):
            return (
                reason,
                "Use CI/CD pipelines for production deployments.",
            )

    # File operations outside project tree
    if cwd:
        # Detect cp/mv/rsync targeting directories outside CWD
        file_ops = re.findall(
            r'\b(?:cp|mv|rsync)\b\s+.*?(?:\s+|=)(\/[^\s;|&]+)',
            command,
        )
        for target in file_ops:
            try:
                resolved = os.path.realpath(target)
                cwd_resolved = os.path.realpath(cwd)
                if not resolved.startswith(cwd_resolved + "/") and resolved != cwd_resolved:
                    # Allow /tmp and common safe paths
                    if not resolved.startswith("/tmp"):
                        return (
                            f"File operation targets path outside project directory: {target}",
                            "Restrict file operations to the current project tree.",
                        )
            except (OSError, ValueError):
                pass  # Skip unresolvable paths; continue scanning.

    return None


# ---------------------------------------------------------------------------
# Category 6: Infrastructure destruction
# ---------------------------------------------------------------------------

INFRA_PATTERNS: list[tuple[re.Pattern, str, str]] = [
    (
        re.compile(r'\b(terraform|tofu)\s+destroy\s+(-auto-approve|--auto-approve)\b'),
        "Terraform/OpenTofu auto-approved destroy detected.",
        "Review the destroy plan manually: terraform plan -destroy.",
    ),
    (
        re.compile(r'\bdocker\s+system\s+prune\s+(-a|--all)\b'),
        "Docker system prune --all detected — removes all unused data.",
        "Use targeted docker prune commands (image, container, volume).",
    ),
    (
        re.compile(r'\bdocker\s+volume\s+prune\s+(-f|--force)\b'),
        "Docker volume force prune detected — removes all unused volumes.",
        "List volumes first with docker volume ls and remove specific ones.",
    ),
    (
        re.compile(r'\bkubectl\s+delete\s+namespace\b'),
        "Kubernetes namespace deletion detected.",
        "Verify the namespace and use kubectl delete with --dry-run=client first.",
    ),
]


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    try:
        input_data = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError) as e:
        audit_log({
            "event": "parse_error",
            "hook": "destructive-prevention",
            "error": str(e),
        })
        print(f"destructive prevention error: {e}", file=sys.stderr)
        sys.exit(2)

    tool_name = input_data.get("tool_name", "")
    tool_input = input_data.get("tool_input", {})
    command = tool_input.get("command", "")

    if tool_name != "Bash" or not command:
        sys.exit(0)

    cwd = os.environ.get("CLAUDE_PROJECT_DIR", "")

    # Category 1: Filesystem
    for pattern, reason, remediation in FILESYSTEM_PATTERNS:
        if pattern.search(command):
            deny("Filesystem Destruction", reason, remediation, command)

    result = check_rm_danger(command)
    if result:
        deny("Filesystem Destruction", result[0], result[1], command)

    # Category 2: Git
    result = check_git_force(command)
    if result:
        deny("Git Force Operation", result[0], result[1], command)

    # Category 3: Database
    for pattern, reason, remediation in DB_PATTERNS:
        if pattern.search(command):
            deny("Database Destruction", reason, remediation, command)
    result = check_delete_without_where(command)
    if result:
        deny("Database Destruction", result[0], result[1], command)

    # Category 4: Remote code execution
    for pattern, reason, remediation in RCE_PATTERNS:
        if pattern.search(command):
            deny("Remote Code Execution", reason, remediation, command)

    # Category 5: Cross-environment
    result = check_cross_environment(command, cwd)
    if result:
        deny("Cross-Environment", result[0], result[1], command)

    # Category 6: Infrastructure
    for pattern, reason, remediation in INFRA_PATTERNS:
        if pattern.search(command):
            deny("Infrastructure Destruction", reason, remediation, command)

    # All checks passed — allow.
    audit_log({
        "event": "allow",
        "hook": "destructive-prevention",
        "command": command,
    })
    sys.exit(0)


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(f"destructive prevention error: {e}", file=sys.stderr)
        sys.exit(2)
