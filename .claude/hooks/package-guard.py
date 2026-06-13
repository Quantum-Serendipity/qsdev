#!/usr/bin/env python3
"""
Claude Code PreToolUse Hook: Package Install Guardrail

Intercepts package install commands, validates packages against OSV.dev
vulnerability database and registry publication age, then allows, denies,
or rewrites the command with safety flags.

Exit codes:
  0 — allow (with optional JSON on stdout for updatedInput or deny)
  2 — deny (stderr message fed back to Claude)

Design decisions:
  - FAILS CLOSED: if any API call fails or times out, the install is denied.
  - Uses only stdlib + urllib (no pip dependencies).
  - OSV.dev is the primary vulnerability source (free, no auth, no rate limits).
  - Publication age checked via npm registry / PyPI JSON API.
  - Configurable allowlist for known-safe packages (lockfile deps, stdlib, etc.).
  - All decisions logged to an audit file for traceability.
  - Timeout budget: 10s per API call, 25s total (fits within 30s hook timeout).

Configuration via environment variables:
  PACKAGE_GUARD_MIN_AGE_DAYS — int  (default: 3, minimum: 1)
  PACKAGE_GUARD_ALLOWLIST    — comma-separated package names to always allow (max 200)
  PACKAGE_GUARD_DENYLIST     — comma-separated package names to always deny

Security invariants (not configurable):
  - Fail-closed mode is always enabled.
  - Minimum age cannot be set below 1 day.
  - Allowlist is capped at 200 entries.
"""

import json
import os
import re
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path
from typing import Optional

# ---------------------------------------------------------------------------
# Configuration (environment variable overrides)
# ---------------------------------------------------------------------------

# Always fail closed on API errors. This is a security invariant that cannot
# be weakened via environment variables.
FAIL_CLOSED = True

# Minimum publication age in days. Packages newer than this are blocked.
# 92% of PyPI malware is caught within 24 hours; 3 days is a strong default.
# The env var can increase strictness but cannot decrease below 1 day.
MIN_AGE_DAYS = max(int(os.environ.get("PACKAGE_GUARD_MIN_AGE_DAYS", "3")), 1)

# Packages that are always allowed without checks. Add project dependencies
# that are already in your lockfile, or well-known stdlib-adjacent packages.
# Merge hardcoded set with environment variable. Capped at 200 entries to
# prevent abuse — an oversized allowlist defeats the purpose of the guard.
_env_allowlist = os.environ.get("PACKAGE_GUARD_ALLOWLIST", "")
_MAX_ALLOWLIST_SIZE = 200
_parsed_allowlist = {p.strip() for p in _env_allowlist.split(",") if p.strip()}
if len(_parsed_allowlist) > _MAX_ALLOWLIST_SIZE:
    _parsed_allowlist = set()
ALLOWLIST: set[str] = _parsed_allowlist | {
    *(),  # empty spread — keeps this a set literal, not a dict
}

# Packages that are always denied, regardless of vulnerability status.
# Use for known-malicious names, typosquats you've encountered, etc.
# Merge hardcoded set with environment variable.
_env_denylist = os.environ.get("PACKAGE_GUARD_DENYLIST", "")
DENYLIST: set[str] = {p.strip() for p in _env_denylist.split(",") if p.strip()} | {
    # "event-stream",  # famous supply chain attack
    # "colors",        # protestware incident
    *(),  # empty spread — keeps this a set literal, not a dict
}

# Team-level allowlist: packages pre-approved by the consulting firm.
# Packages in this list bypass vulnerability and age checks entirely.
_env_team_allowlist = os.environ.get("PACKAGE_GUARD_TEAM_ALLOWLIST", "")
TEAM_ALLOWLIST: set[str] = {p.strip() for p in _env_team_allowlist.split(",") if p.strip()}

# Team-level denylist: packages blocked by consulting firm policy.
# Checked before all other checks (including per-project allowlist).
_env_team_denylist = os.environ.get("PACKAGE_GUARD_TEAM_DENYLIST", "")
TEAM_DENYLIST: set[str] = {p.strip() for p in _env_team_denylist.split(",") if p.strip()}

# New-dependency gate: controls behavior for packages not in the project lockfile.
# "deny" = block new deps, "ask" = flag for review, "allow" = permit (default).
NEW_DEP_GATE: str = os.environ.get("PACKAGE_GUARD_NEW_DEP_GATE", "allow")

# SOC 2 audit trail: when enabled, log dependency decisions to a separate file.
SOC2_AUDIT_ENABLED: bool = os.environ.get("PACKAGE_GUARD_SOC2_AUDIT", "").lower() == "true"

# Timeout per individual API call in seconds.
API_TIMEOUT: int = 10

# Audit log file path. Uses CLAUDE_PROJECT_DIR if available, else /tmp.
AUDIT_LOG: Path = Path(
    os.environ.get("CLAUDE_PROJECT_DIR", "/tmp")
) / ".claude" / "hook-audit.log"

# SOC 2 dependency audit trail.
SOC2_AUDIT_DIR: Path = Path(
    os.environ.get("CLAUDE_AUDIT_DIR", os.path.expanduser("~/.claude/audit"))
)
SOC2_AUDIT_FILE: Path = SOC2_AUDIT_DIR / f"dependency-changes-{datetime.now(timezone.utc).strftime('%Y-%m')}.jsonl"

# ---------------------------------------------------------------------------
# Package manager detection patterns
# ---------------------------------------------------------------------------

# Each tuple: (compiled regex matching an install command, ecosystem label,
#               command verb position for extraction, safety flags to append)
INSTALL_PATTERNS: list[tuple[re.Pattern, str, str]] = [
    # npm / npx
    (re.compile(r'\bnpm\s+(install|i|add)\b'), "npm", "npm"),
    (re.compile(r'\bnpx\s+'), "npm", "npx"),
    # yarn
    (re.compile(r'\byarn\s+(add|install)\b'), "npm", "yarn"),
    # pnpm
    (re.compile(r'\bpnpm\s+(add|install|i)\b'), "npm", "pnpm"),
    # bun
    (re.compile(r'\bbun\s+(add|install|i)\b'), "npm", "bun"),
    # pip / pip3
    (re.compile(r'\bpip3?\s+install\b'), "PyPI", "pip"),
    # uv
    (re.compile(r'\buv\s+pip\s+install\b'), "PyPI", "uv-pip"),
    (re.compile(r'\buv\s+add\b'), "PyPI", "uv-add"),
    # cargo
    (re.compile(r'\bcargo\s+(add|install)\b'), "crates.io", "cargo"),
    # go
    (re.compile(r'\bgo\s+(get|install)\b'), "Go", "go"),
    # gem
    (re.compile(r'\bgem\s+install\b'), "RubyGems", "gem"),
    # composer
    (re.compile(r'\bcomposer\s+require\b'), "Packagist", "composer"),
    # nix (imperative installs — these should generally be blocked entirely)
    (re.compile(r'\bnix-env\s+-i\b'), "nix", "nix-env"),
    (re.compile(r'\bnix\s+profile\s+install\b'), "nix", "nix-profile"),
]

# Safety flags to append via updatedInput, keyed by manager label.
SAFETY_FLAGS: dict[str, str] = {
    "npm":     " --ignore-scripts",
    "yarn":    "",  # yarn 2+ has different flag semantics; leave to env config
    "pnpm":    "",  # pnpm v10+ blocks scripts by default
    "bun":     "",  # bun blocks scripts by default
    "pip":     " --only-binary :all:",
    "uv-pip":  " --only-binary :all:",
    "uv-add":  "",  # uv add does not support --only-binary
    "cargo":   " --locked",
    "go":      "",  # go modules have sumdb verification built in
    "gem":     "",
    "composer": "",
    "nix-env":     "",  # nix-env should be denied outright
    "nix-profile": "",
    "npx":     "",
}

# Flags that consume the next argument (so we skip them during extraction).
FLAGS_WITH_ARGS: set[str] = {
    "--registry", "--save-prefix", "--save-exact", "--tag",
    "--cache", "--prefer-offline", "--target", "--platform",
    "--index-url", "--extra-index-url", "--find-links",
    "--constraint", "--requirement", "-r", "-c", "-f",
    "--git", "--path", "--branch", "--rev", "--features",
    "-p", "--package",
}

# Ecosystem-specific version separator patterns.
# npm: package@version, pip: package==version or package>=version,
# cargo: package@version, go: package@version, gem: package -v version
VERSION_STRIP_RE = re.compile(r"[@=><~^!]+.*$")

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

def audit_log(entry: dict) -> None:
    """Append a JSON entry to the audit log file."""
    try:
        AUDIT_LOG.parent.mkdir(parents=True, exist_ok=True)
        with open(AUDIT_LOG, "a") as f:
            entry["timestamp"] = datetime.now(timezone.utc).isoformat()
            f.write(json.dumps(entry) + "\n")
    except OSError:
        # Logging failure must not block the hook decision.
        pass


def soc2_audit_log(entry: dict) -> None:
    """Append a dependency decision to the SOC 2 audit trail when enabled."""
    if not SOC2_AUDIT_ENABLED:
        return
    try:
        SOC2_AUDIT_DIR.mkdir(parents=True, exist_ok=True)
        entry["timestamp"] = datetime.now(timezone.utc).isoformat()
        with open(SOC2_AUDIT_FILE, "a") as f:
            f.write(json.dumps(entry) + "\n")
    except OSError:
        pass  # SOC2 audit logging is best-effort; must not block hook decisions.


def is_in_lockfile(package_name: str, ecosystem: str) -> Optional[bool]:
    """
    Check if a package appears in a project lockfile.
    Returns True if found, False if lockfile exists but package is missing,
    None if no lockfile is detected (skip check).
    """
    project_dir = os.environ.get("CLAUDE_PROJECT_DIR", ".")
    lower_name = package_name.lower()
    lockfiles: list[tuple[str, str]] = []

    if ecosystem == "npm":
        lockfiles = [
            ("package-lock.json", lower_name),
            ("yarn.lock", package_name),
            ("pnpm-lock.yaml", package_name),
        ]
    elif ecosystem == "PyPI":
        lockfiles = [
            ("requirements.txt", lower_name),
            ("Pipfile.lock", f'"{package_name}"'),
            ("uv.lock", lower_name),
        ]
    elif ecosystem == "crates.io":
        lockfiles = [
            ("Cargo.lock", f'name = "{package_name}"'),
        ]

    for filename, search_term in lockfiles:
        lockpath = Path(project_dir) / filename
        try:
            if lockpath.exists():
                content = lockpath.read_text(errors="replace")
                return search_term.lower() in content.lower()
        except OSError:
            pass  # Unreadable lockfiles are treated as not determinable.

    return None


# ---------------------------------------------------------------------------
# API callers
# ---------------------------------------------------------------------------

def query_osv(package_name: str, ecosystem: str, version: Optional[str] = None) -> dict:
    """
    Query OSV.dev for known vulnerabilities.
    Returns the raw response dict, or raises on failure.
    """
    payload: dict = {
        "package": {
            "name": package_name,
            "ecosystem": ecosystem,
        }
    }
    if version:
        payload["version"] = version

    req = urllib.request.Request(
        "https://api.osv.dev/v1/query",
        data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=API_TIMEOUT) as resp:
        return json.loads(resp.read().decode("utf-8"))


def check_npm_age(package_name: str) -> Optional[float]:
    """
    Query npm registry for the package's latest version publication date.
    Returns age in days, or None on failure.

    Uses dist-tags.latest to find the current version, then looks up that
    version's publish date in the time map. This avoids using time.modified
    which reflects any metadata change (e.g. deprecation notices) and does
    not represent an actual version publication event.
    """
    url = f"https://registry.npmjs.org/{urllib.request.quote(package_name, safe='@/')}"
    req = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=API_TIMEOUT) as resp:
        data = json.loads(resp.read().decode("utf-8"))

    dist_tags = data.get("dist-tags", {})
    latest_version = dist_tags.get("latest")
    if not latest_version:
        return None

    time_map = data.get("time", {})
    version_published = time_map.get(latest_version)
    if not version_published:
        return None

    pub_date = datetime.fromisoformat(version_published.replace("Z", "+00:00"))
    age = (datetime.now(timezone.utc) - pub_date).total_seconds() / 86400
    return age


def check_pypi_age(package_name: str) -> Optional[float]:
    """
    Query PyPI JSON API for the package's latest upload date.
    Returns age in days, or None on failure.
    """
    url = f"https://pypi.org/pypi/{urllib.request.quote(package_name)}/json"
    req = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=API_TIMEOUT) as resp:
        data = json.loads(resp.read().decode("utf-8"))

    # urls[] contains upload_time_iso_8601 for each file in the latest release.
    urls = data.get("urls", [])
    if not urls:
        return None

    # Use the most recent upload timestamp across all files.
    timestamps = [
        datetime.fromisoformat(u["upload_time_iso_8601"].replace("Z", "+00:00"))
        for u in urls
        if u.get("upload_time_iso_8601")
    ]
    if not timestamps:
        return None
    latest = max(timestamps)
    age = (datetime.now(timezone.utc) - latest).total_seconds() / 86400
    return age


def check_crates_age(package_name: str) -> Optional[float]:
    """
    Query crates.io API for the crate's latest version publication date.
    Returns age in days, or None on failure.
    """
    url = f"https://crates.io/api/v1/crates/{urllib.request.quote(package_name)}"
    req = urllib.request.Request(
        url,
        headers={
            "Accept": "application/json",
            # crates.io requires a User-Agent header.
            "User-Agent": "claude-code-package-guardrail/1.0",
        },
    )
    with urllib.request.urlopen(req, timeout=API_TIMEOUT) as resp:
        data = json.loads(resp.read().decode("utf-8"))

    newest = data.get("crate", {}).get("newest_version")
    versions = data.get("versions", [])
    for v in versions:
        if v.get("num") == newest:
            created = v.get("created_at")
            if created:
                pub_date = datetime.fromisoformat(created.replace("Z", "+00:00"))
                return (datetime.now(timezone.utc) - pub_date).total_seconds() / 86400
    return None


# ---------------------------------------------------------------------------
# Package name extraction
# ---------------------------------------------------------------------------

def extract_packages(command: str, manager: str) -> list[str]:
    """
    Extract package name(s) from a package install command string.
    Strips flags, handles version specifiers, quoted names, etc.

    Returns a list of raw package specifiers (may include version info).
    """
    # Split respecting basic quoting (handles "pkg with space" or 'pkg').
    # For robust shell parsing we'd need shlex, but package names don't have
    # spaces so a simple split suffices for the common case.
    parts = command.split()
    packages: list[str] = []
    skip_next = False

    # Words that are part of the command prefix, not package names.
    command_verbs = {
        "npm", "npx", "yarn", "pnpm", "bun",
        "pip", "pip3", "uv",
        "cargo", "go",
        "gem", "composer",
        "nix-env", "nix", "profile",
        "install", "add", "i", "get", "require",
    }

    for i, part in enumerate(parts):
        if skip_next:
            skip_next = False
            continue

        # Skip flags.
        if part.startswith("-"):
            if part in FLAGS_WITH_ARGS or part.rstrip("=") in FLAGS_WITH_ARGS:
                skip_next = True
            continue

        # Skip command verbs.
        if part.lower() in command_verbs:
            continue

        # Stop at shell operators — anything after && is a separate command.
        if part in ("&&", "||", ";", "|"):
            break

        # Skip common non-package arguments.
        if part in (".", "./", ".."):
            continue

        packages.append(part)

    return packages


def strip_version(specifier: str) -> str:
    """
    Strip version info from a package specifier.
    Examples:
      axios@1.6.0       -> axios
      requests==2.31.0   -> requests
      serde@^1.0         -> serde
      lodash             -> lodash
      @scope/pkg@1.0.0   -> @scope/pkg
    """
    # Handle scoped npm packages: @scope/name@version
    if specifier.startswith("@") and "/" in specifier:
        # Find the second @ which is the version separator.
        slash_pos = specifier.index("/")
        rest = specifier[slash_pos + 1:]
        at_pos = rest.find("@")
        if at_pos >= 0:
            return specifier[:slash_pos + 1 + at_pos]
        # No version — check for == or >= etc.
        cleaned = VERSION_STRIP_RE.sub("", specifier)
        return cleaned if cleaned else specifier

    # For Go module paths: github.com/user/repo@version
    if specifier.startswith("github.com/") or specifier.startswith("golang.org/"):
        at_pos = specifier.rfind("@")
        if at_pos > 0:
            return specifier[:at_pos]
        return specifier

    # General case: strip at first version separator.
    cleaned = VERSION_STRIP_RE.sub("", specifier)
    # Also handle the @ separator for non-scoped packages.
    at_pos = cleaned.find("@")
    if at_pos > 0:
        cleaned = cleaned[:at_pos]

    return cleaned if cleaned else specifier


# ---------------------------------------------------------------------------
# Command matching
# ---------------------------------------------------------------------------

def detect_install_commands(command: str) -> list[tuple[str, str, str]]:
    """
    Find ALL package install invocations in a (possibly compound) command.
    Returns a list of (ecosystem, manager_label, segment) tuples — one per
    segment that contains an install command. Validates every segment so that
    compound commands like ``pip install safe && npm install evil`` cannot
    sneak an unchecked install past the guard.
    """
    segments = re.split(r'\s*(?:&&|\|\||;)\s*', command)
    results: list[tuple[str, str, str]] = []

    for segment in segments:
        for pattern, ecosystem, manager in INSTALL_PATTERNS:
            if pattern.search(segment):
                results.append((ecosystem, manager, segment))
                break  # one match per segment is sufficient

    return results


# ---------------------------------------------------------------------------
# Safety flag injection
# ---------------------------------------------------------------------------

def apply_safety_flags(command: str, manager: str) -> Optional[str]:
    """
    Return a modified command with safety flags appended, or None if
    no modification is needed.
    """
    flags = SAFETY_FLAGS.get(manager, "")
    if not flags:
        return None

    # Check if the flag is already present.
    flag_name = flags.strip().split()[0]  # e.g., "--ignore-scripts"
    if flag_name in command:
        return None

    # For compound commands, we need to inject the flag into the right segment.
    # Find the segment that contains the install command and append there.
    segments = re.split(r'(\s*(?:&&|\|\||;)\s*)', command)
    for i, segment in enumerate(segments):
        for pattern, _, mgr in INSTALL_PATTERNS:
            if mgr == manager and pattern.search(segment):
                segments[i] = segment.rstrip() + flags
                return "".join(segments)

    # Fallback: append to end.
    return command.rstrip() + flags


# ---------------------------------------------------------------------------
# Core validation logic
# ---------------------------------------------------------------------------

def validate_package(
    package_name: str,
    ecosystem: str,
    manager: str,
    version: Optional[str] = None,
) -> tuple[str, str]:
    """
    Validate a single package. Returns (decision, reason).
    decision is one of: "allow", "deny", "ask"
    """
    # 0a. Team denylist (checked before everything).
    if package_name.lower() in {d.lower() for d in TEAM_DENYLIST}:
        return "deny", f"Package '{package_name}' is on the team denylist per consulting policy."

    # 0b. Team allowlist (pre-approved, bypass vuln/age checks).
    if TEAM_ALLOWLIST:
        if package_name.lower() in {a.lower() for a in TEAM_ALLOWLIST}:
            return "allow", f"Package '{package_name}' is team-approved (pre-approved bypass)."
        # If team allowlist is non-empty and package not in it, deny.
        return "deny", f"Package '{package_name}' is not on the team-approved list."

    # 1. Check denylist.
    if package_name.lower() in {d.lower() for d in DENYLIST}:
        return "deny", f"Package '{package_name}' is on the explicit denylist."

    # 2. Check allowlist.
    if package_name.lower() in {a.lower() for a in ALLOWLIST}:
        return "allow", f"Package '{package_name}' is on the allowlist."

    # 3. Check OSV.dev for known vulnerabilities.
    try:
        osv_result = query_osv(package_name, ecosystem, version)
        vulns = osv_result.get("vulns", [])
        if vulns:
            # Categorize by severity.
            vuln_ids = [v.get("id", "unknown") for v in vulns[:5]]
            severities = []
            for v in vulns:
                for s in v.get("severity", []):
                    if s.get("type") == "CVSS_V3":
                        score_str = s.get("score", "")
                        # CVSS vector string — extract base score.
                        # Format: CVSS:3.1/AV:N/AC:L/... but OSV also
                        # stores numeric scores in the "score" field of
                        # database_specific or elsewhere. The severity[].score
                        # in OSV schema is the CVSS vector string.
                        severities.append(score_str)

            summary = (
                f"Package '{package_name}' has {len(vulns)} known vulnerabilities: "
                f"{', '.join(vuln_ids[:5])}. Choose a patched version or an alternative package."
            )
            if len(vulns) > 5:
                summary += f" (and {len(vulns) - 5} more)"

            return "deny", summary
    except (urllib.error.URLError, urllib.error.HTTPError, OSError, json.JSONDecodeError, ValueError) as e:
        if FAIL_CLOSED:
            return "deny", f"OSV.dev API check failed for '{package_name}' ({type(e).__name__}: {e}). Failing closed."
        # If fail-open (not recommended), fall through to age check.

    # 4. Check publication age (ecosystem-specific).
    try:
        age_days: Optional[float] = None

        if ecosystem == "npm":
            age_days = check_npm_age(package_name)
        elif ecosystem == "PyPI":
            age_days = check_pypi_age(package_name)
        elif ecosystem == "crates.io":
            age_days = check_crates_age(package_name)
        # Go, RubyGems, Packagist: age check not implemented yet.
        # They fall through to allow.

        if age_days is not None and age_days < MIN_AGE_DAYS:
            return "deny", (
                f"Package '{package_name}' was published/updated {age_days:.1f} days ago "
                f"(minimum: {MIN_AGE_DAYS} days). New packages are quarantined to "
                f"block supply chain attacks. Wait for the quarantine period to pass, "
                f"or use an older, established version."
            )
    except (urllib.error.URLError, urllib.error.HTTPError, OSError, json.JSONDecodeError, ValueError) as e:
        if FAIL_CLOSED:
            return "deny", f"Registry age check failed for '{package_name}' ({type(e).__name__}: {e}). Failing closed."

    return "allow", f"Package '{package_name}' passed all checks."


# ---------------------------------------------------------------------------
# Main hook logic
# ---------------------------------------------------------------------------

def main() -> None:
    start_time = time.monotonic()

    # 1. Read JSON from stdin.
    try:
        input_data = json.load(sys.stdin)
    except (json.JSONDecodeError, ValueError) as e:
        audit_log({"event": "parse_error", "error": str(e)})
        # Cannot parse input — fail closed.
        print(f"Hook error: failed to parse stdin JSON: {e}", file=sys.stderr)
        sys.exit(2)

    tool_name = input_data.get("tool_name", "")
    command = input_data.get("tool_input", {}).get("command", "")

    # 2. Only process Bash tool calls.
    if tool_name != "Bash" or not command:
        sys.exit(0)

    # 3. Detect ALL package install commands (handles compound commands).
    detections = detect_install_commands(command)
    if not detections:
        # Not a package install command — allow silently.
        sys.exit(0)

    # 4. Validate every detected install command. Most restrictive wins.
    deny_reasons: list[str] = []
    ask_reasons: list[str] = []
    checked_packages: list[str] = []
    needs_safety_flags: list[str] = []  # managers that need flag injection

    for ecosystem, manager, segment in detections:
        # Nix imperative installs: deny outright.
        if manager in ("nix-env", "nix-profile"):
            reason = (
                f"Imperative Nix installs ({manager}) bypass flake pinning and are not "
                f"permitted. Use `qsdev devenv add-package <name>` for system packages "
                f"or `qsdev enable <tool>` for ecosystem tools instead."
            )
            deny_reasons.append(reason)
            audit_log({
                "event": "deny_nix",
                "command": command,
                "segment": segment,
                "manager": manager,
                "reason": reason,
            })
            continue

        # Extract package names from this segment (not the full compound command).
        packages = extract_packages(segment, manager)

        if not packages:
            # Bare install from lockfile/manifest — track for safety flags.
            needs_safety_flags.append(manager)
            continue

        # Validate each package in this segment.
        for specifier in packages:
            pkg_name = strip_version(specifier)
            version: Optional[str] = None
            if specifier != pkg_name:
                version = specifier[len(pkg_name):].lstrip("@=><~^!")
                if not version:
                    version = None

            checked_packages.append(pkg_name)
            decision, reason = validate_package(pkg_name, ecosystem, manager, version)

            is_new = None
            if decision == "allow" and NEW_DEP_GATE != "allow":
                in_lock = is_in_lockfile(pkg_name, ecosystem)
                if in_lock is False:
                    is_new = True
                    gate_msg = (
                        f"New dependency '{pkg_name}' not found in project lockfile. "
                        f"Add to team allowlist or lockfile first."
                    )
                    if NEW_DEP_GATE == "deny":
                        decision, reason = "deny", gate_msg
                    elif NEW_DEP_GATE == "ask":
                        decision, reason = "ask", gate_msg
                elif in_lock is True:
                    is_new = False

            if decision == "deny":
                deny_reasons.append(reason)
            elif decision == "ask":
                ask_reasons.append(reason)

            soc2_audit_log({
                "package_name": pkg_name,
                "ecosystem": ecosystem,
                "version": version or "",
                "decision": decision,
                "reason": reason,
                "is_new_dependency": is_new,
            })

        needs_safety_flags.append(manager)

    elapsed = time.monotonic() - start_time

    # 5. Make final decision. Most restrictive wins across ALL segments.
    if deny_reasons:
        combined_reason = " | ".join(deny_reasons)
        audit_log({
            "event": "deny",
            "command": command,
            "packages": checked_packages,
            "reasons": deny_reasons,
            "elapsed_seconds": round(elapsed, 2),
        })
        result = {
            "hookSpecificOutput": {
                "hookEventName": "PreToolUse",
                "permissionDecision": "deny",
                "permissionDecisionReason": combined_reason,
            }
        }
        print(json.dumps(result))
        sys.exit(0)

    if ask_reasons:
        combined_reason = " | ".join(ask_reasons)
        audit_log({
            "event": "ask",
            "command": command,
            "packages": checked_packages,
            "reasons": ask_reasons,
            "elapsed_seconds": round(elapsed, 2),
        })
        result = {
            "hookSpecificOutput": {
                "hookEventName": "PreToolUse",
                "permissionDecision": "ask",
                "permissionDecisionReason": combined_reason,
            }
        }
        print(json.dumps(result))
        sys.exit(0)

    # 6. All packages passed — apply safety flags for each manager detected.
    rewritten = command
    for mgr in needs_safety_flags:
        candidate = apply_safety_flags(rewritten, mgr)
        if candidate and candidate != rewritten:
            rewritten = candidate

    audit_log({
        "event": "allow",
        "command": command,
        "rewritten": rewritten if rewritten != command else None,
        "packages": checked_packages,
        "elapsed_seconds": round(elapsed, 2),
    })

    if rewritten != command:
        result = {
            "hookSpecificOutput": {
                "hookEventName": "PreToolUse",
                "permissionDecision": "allow",
                "updatedInput": {"command": rewritten},
                "additionalContext": (
                    f"Packages validated and safety flags appended. "
                    f"Checked: {', '.join(checked_packages)}. "
                    f"Original: `{command}` -> Rewritten: `{rewritten}`"
                ),
            }
        }
        print(json.dumps(result))

    # Exit 0 = allow.
    sys.exit(0)


if __name__ == "__main__":
    main()
