#!/usr/bin/env python3
"""
Claude Code PreToolUse Hook: Package Install Guardrail

Intercepts package install commands, resolves the version each install would
actually pick, validates it against the OSV.dev vulnerability database and
registry publication age, then denies, asks, or rewrites the command with
safety flags.

Exit codes / decisions:
  0 — no JSON: not an install, or validated with nothing to change (the normal
      permission flow decides)
  0 + JSON — deny / ask; a command with inserted safety flags is returned as
      updatedInput with `ask`, so the user approves exactly what will run
  2 — internal error or unreadable input (blocks the tool call: fail closed)

Design decisions:
  - FAILS CLOSED: if any API call fails or times out, the install is denied;
    an internal error or invalid configuration blocks the call.
  - Never widens permissions: the hook never returns `allow`.
  - Uses only stdlib + urllib (no pip dependencies).
  - OSV.dev is the primary vulnerability source (free, no auth, no rate limits).
  - Publication age checked via the registry of each ecosystem.
  - Configurable allow/deny lists (optionally ecosystem-qualified).
  - All decisions logged (credentials redacted) for traceability.
  - Timeout budget: packages are validated concurrently, each API call is
    bounded by the remaining budget, and running out of the VALIDATION_BUDGET
    (well inside the 30s hook timeout) denies the install.

Configuration via environment variables:
  PACKAGE_GUARD_MIN_AGE_DAYS    — int  (default: 3, minimum: 1)
  PACKAGE_GUARD_ALLOWLIST       — comma-separated package names to always allow (max 200)
  PACKAGE_GUARD_DENYLIST        — comma-separated package names to always deny
  PACKAGE_GUARD_TEAM_ALLOWLIST  — when set, only these packages may be installed
  PACKAGE_GUARD_TEAM_DENYLIST   — checked before everything else
  PACKAGE_GUARD_NEW_DEP_GATE    — allow | ask | deny for packages missing from the lockfiles
  PACKAGE_GUARD_SOC2_AUDIT      — "true" writes a dependency-decision trail

  List entries may be qualified with an ecosystem (`pypi:requests`,
  `npm:lodash`, `crates:serde`, `go:golang.org/x/net`); an unqualified entry
  applies to every ecosystem. Names are compared the way each registry does
  (PyPI: PEP 503, so `Evil.Pkg` == `evil_pkg`; crates.io: `-` == `_`).
  An invalid value blocks installs with an explanation instead of being
  silently ignored.

Security invariants (not configurable):
  - Fail-closed mode is always enabled.
  - Minimum age cannot be set below 1 day.
  - Allowlist is capped at 200 entries.
"""

import base64
import binascii
import gzip
import json
import os
import queue
import re
import shlex
import sys
import threading
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path
from typing import NamedTuple, Optional

# Oldest interpreter the guard supports. Below it, block (exit 2) with a clear
# message instead of crashing with exit 1, which Claude Code treats as a
# non-blocking error (the install would proceed unguarded). The module must
# still COMPILE on older interpreters for this check to run: no `from
# __future__ import annotations` (a SyntaxError on 3.6) and no subscripted
# builtins (`list[str]`) in evaluated annotations.
_MIN_PYTHON = (3, 8)
if sys.version_info < _MIN_PYTHON:
    print(f"package-guard requires Python {'.'.join(map(str, _MIN_PYTHON))}+ "
          f"(found {sys.version.split()[0]}); blocking to fail closed.", file=sys.stderr)
    sys.exit(2)

# ---------------------------------------------------------------------------
# Configuration (environment variable overrides)
# ---------------------------------------------------------------------------

# Always fail closed on API errors. This is a security invariant that cannot
# be weakened via environment variables.
FAIL_CLOSED = True

# Invalid configuration values. main() blocks installs while any are present:
# a typo must not silently disable (or crash) a control.
CONFIG_ERRORS: list = []


def _int_env(name: str, default: int, minimum: int) -> int:
    raw = os.environ.get(name, "").strip()
    if not raw:
        return default
    try:
        return max(int(raw), minimum)
    except ValueError:
        CONFIG_ERRORS.append(f"{name}={raw!r} is not a whole number of days")
        return default


# Minimum publication age in days. Packages newer than this are blocked.
# 92% of PyPI malware is caught within 24 hours; 3 days is a strong default.
# The env var can increase strictness but cannot decrease below 1 day.
MIN_AGE_DAYS = _int_env("PACKAGE_GUARD_MIN_AGE_DAYS", 3, 1)

# Ecosystem qualifiers accepted in list entries (`pypi:requests`).
_ECOSYSTEM_QUALIFIERS: dict = {
    "npm": "npm", "pypi": "PyPI", "pip": "PyPI", "crates.io": "crates.io", "crates": "crates.io",
    "cargo": "crates.io", "go": "Go", "golang": "Go", "rubygems": "RubyGems", "gem": "RubyGems",
    "packagist": "Packagist", "composer": "Packagist", "nuget": "NuGet", "pub": "Pub",
}


def normalize_name(name: str, ecosystem: str) -> str:
    """A package name as its registry compares names: PyPI per PEP 503
    (case-insensitive, runs of `-_.` equal), crates.io treats `-` and `_` as
    the same crate, the others are compared case-insensitively."""
    name = name.strip()
    if ecosystem == "PyPI":
        return re.sub(r"[-_.]+", "-", name).lower()
    if ecosystem == "crates.io":
        return name.lower().replace("_", "-")
    return name.lower()


class NameList(NamedTuple):
    entries: tuple   # (ecosystem or "" for every ecosystem, raw name)

    def matches(self, name: str, ecosystem: str) -> bool:
        want = normalize_name(name, ecosystem)
        return any((not eco or eco == ecosystem) and normalize_name(raw, ecosystem) == want
                   for eco, raw in self.entries)

    def __bool__(self) -> bool:
        return bool(self.entries)


def _name_list(var: str, cap: Optional[int] = None) -> NameList:
    entries = []
    for item in os.environ.get(var, "").split(","):
        item = item.strip()
        if not item:
            continue
        qualifier, sep, rest = item.partition(":")
        if sep and qualifier.lower() in _ECOSYSTEM_QUALIFIERS and rest.strip():
            entries.append((_ECOSYSTEM_QUALIFIERS[qualifier.lower()], rest.strip()))
        else:
            entries.append(("", item))
    if cap is not None and len(entries) > cap:
        # An oversized allowlist defeats the purpose of the guard; say so
        # instead of silently dropping it.
        CONFIG_ERRORS.append(f"{var} has {len(entries)} entries (maximum {cap})")
        return NameList(())
    return NameList(tuple(entries))


# Packages that are always allowed without checks (lockfile deps, stdlib-
# adjacent packages). Capped at 200 entries.
_MAX_ALLOWLIST_SIZE = 200
ALLOWLIST: NameList = _name_list("PACKAGE_GUARD_ALLOWLIST", _MAX_ALLOWLIST_SIZE)

# Packages that are always denied, regardless of vulnerability status
# (known-malicious names, typosquats you've encountered, etc.).
DENYLIST: NameList = _name_list("PACKAGE_GUARD_DENYLIST")

# Team-level allowlist: packages pre-approved by the consulting firm.
# Packages in this list bypass vulnerability and age checks entirely.
TEAM_ALLOWLIST: NameList = _name_list("PACKAGE_GUARD_TEAM_ALLOWLIST")

# Team-level denylist: packages blocked by consulting firm policy.
# Checked before all other checks (including per-project allowlist).
TEAM_DENYLIST: NameList = _name_list("PACKAGE_GUARD_TEAM_DENYLIST")

# New-dependency gate: controls behavior for packages not in the project lockfile.
# "deny" = block new deps, "ask" = flag for review, "allow" = permit (default).
NEW_DEP_GATE: str = os.environ.get("PACKAGE_GUARD_NEW_DEP_GATE", "").strip().lower() or "allow"
if NEW_DEP_GATE not in ("allow", "ask", "deny"):
    CONFIG_ERRORS.append(f"PACKAGE_GUARD_NEW_DEP_GATE={NEW_DEP_GATE!r} is not one of allow, ask, deny")
    NEW_DEP_GATE = "deny"

# SOC 2 audit trail: when enabled, log dependency decisions to a separate file.
SOC2_AUDIT_ENABLED: bool = os.environ.get("PACKAGE_GUARD_SOC2_AUDIT", "").lower() == "true"

# Timeout per individual API call in seconds.
API_TIMEOUT: int = 10

# Tools whose tool_input.command runs in a shell. The hook's settings.json
# matcher must list exactly these tools (hook_registry.go shellToolMatcher;
# kept in sync by TestHookMatchersCoverScriptTools).
SHELL_TOOLS = ("Bash", "PowerShell", "Monitor")

# Wall-clock budget for all registry/OSV work. The hook is registered with a
# 30s timeout, and a hook Claude Code kills for running too long does NOT
# block the tool call, so the guard must decide (deny) well before that.
VALIDATION_BUDGET: float = 20.0
_MAX_WORKERS = 8
_deadline: Optional[float] = None


def _call_timeout() -> float:
    """Socket timeout for the next API call: API_TIMEOUT, cut to what is left
    of the validation budget. Raises TimeoutError once it is spent."""
    if _deadline is None:
        return API_TIMEOUT
    left = _deadline - time.monotonic()
    if left <= 0:
        raise TimeoutError("package validation time budget exhausted")
    return min(API_TIMEOUT, left)


# Audit log. It sits with the other hooks' logs under .claude/logs/, which
# `qsdev init` adds to .gitignore. Without a project directory it goes to the
# user's own ~/.claude/logs, never a shared location such as /tmp.
_PROJECT_DIR = os.environ.get("CLAUDE_PROJECT_DIR", "")
AUDIT_LOG: Path = Path(_PROJECT_DIR or os.path.expanduser("~")) / ".claude" / "logs" / "hook-audit.jsonl"

# SOC 2 dependency audit trail.
SOC2_AUDIT_DIR: Path = Path(
    os.environ.get("CLAUDE_AUDIT_DIR", os.path.expanduser("~/.claude/audit"))
)
SOC2_AUDIT_FILE: Path = SOC2_AUDIT_DIR / f"dependency-changes-{datetime.now(timezone.utc).strftime('%Y-%m')}.jsonl"

# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

# Credentials that appear in install commands: URL userinfo
# (https://user:token@index/simple), secret-named assignments and flags
# (NPM_TOKEN=..., //registry/:_authToken=..., --password x, --otp=123).
_SECRET_NAME = r"[A-Za-z0-9_.-]*(?:token|secret|passw(?:or)?d|api[_-]?key|_auth|otp|credential)[A-Za-z0-9_.-]*"
# A value: quoted (may contain spaces) or a bare word.
_SECRET_VALUE = r"""(?:'[^']*'|"[^"]*"|[^\s'"]+)"""
_REDACTIONS = (
    # Greedy up to the last `@` of the authority: a password may contain `@`.
    (re.compile(r"(?i)\b([a-z][a-z0-9+.-]*://)[^/\s'\"]+@"), r"\1***@"),
    (re.compile(r"(?i)((?:^|[\s;&|(:])-{0,2}" + _SECRET_NAME + r")(=)(?!=)" + _SECRET_VALUE), r"\1\2***"),
    (re.compile(r"(?i)((?:^|\s)--?" + _SECRET_NAME + r")(\s+)(?!-)" + _SECRET_VALUE), r"\1\2***"),
)


def redact(value):
    """Strip credentials from a log entry (strings, lists and dicts)."""
    if isinstance(value, str):
        for pattern, repl in _REDACTIONS:
            value = pattern.sub(repl, value)
        return value
    if isinstance(value, list):
        return [redact(v) for v in value]
    if isinstance(value, dict):
        return {k: redact(v) for k, v in value.items()}
    return value


def _append_private(path: Path, line: str) -> None:
    """Append to a 0600 file without following a planted symlink."""
    path.parent.mkdir(parents=True, exist_ok=True)
    fd = os.open(str(path), os.O_WRONLY | os.O_APPEND | os.O_CREAT | getattr(os, "O_NOFOLLOW", 0), 0o600)
    with os.fdopen(fd, "a") as f:
        f.write(line)


AUDIT_LOG_MAX_BYTES = 10 * 1024 * 1024


def audit_log(entry: dict) -> None:
    """Append a JSON entry to the audit log file, rotating it to <name>.1 once
    it exceeds AUDIT_LOG_MAX_BYTES."""
    try:
        try:
            if AUDIT_LOG.lstat().st_size > AUDIT_LOG_MAX_BYTES:
                os.replace(AUDIT_LOG, AUDIT_LOG.with_name(AUDIT_LOG.name + ".1"))
        except FileNotFoundError:
            pass
        entry["timestamp"] = datetime.now(timezone.utc).isoformat()
        _append_private(AUDIT_LOG, json.dumps(redact(entry)) + "\n")
    except OSError:
        # Logging failure must not block the hook decision.
        pass


def soc2_audit_log(entry: dict) -> None:
    """Append a dependency decision to the SOC 2 audit trail when enabled."""
    if not SOC2_AUDIT_ENABLED:
        return
    try:
        entry["timestamp"] = datetime.now(timezone.utc).isoformat()
        _append_private(SOC2_AUDIT_FILE, json.dumps(redact(entry)) + "\n")
    except OSError:
        pass  # SOC2 audit logging is best-effort; must not block hook decisions.


# ---------------------------------------------------------------------------
# Lockfiles (new-dependency gate)
# ---------------------------------------------------------------------------
#
# Each lockfile format is parsed for the package NAMES it locks, which are
# compared exactly after registry normalisation: `react` is not "present"
# because `react-dom` is locked, and `Flask_Cors` is `flask-cors`.

def _toml_package_names(text: str) -> set:
    """`name = "..."` of every [[package]] / [[packages]] table (uv.lock,
    poetry.lock, pdm.lock, pylock.toml, Cargo.lock)."""
    names: set = set()
    in_package = False
    for line in text.splitlines():
        stripped = line.strip()
        if stripped.startswith("["):
            in_package = stripped in ("[[package]]", "[[packages]]")
            continue
        m = re.match(r'^name\s*=\s*"([^"]+)"', stripped)
        if in_package and m:
            names.add(m.group(1))
            in_package = False
    return names


def _names_package_lock(text: str) -> set:
    data = json.loads(text)
    names = {key.rsplit("node_modules/", 1)[1] for key in (data.get("packages") or {})
             if "node_modules/" in key}

    def walk(deps) -> None:
        for name, info in (deps or {}).items():
            names.add(name)
            if isinstance(info, dict):
                walk(info.get("dependencies"))

    walk(data.get("dependencies"))
    return names


def _names_yarn_lock(text: str) -> set:
    """Entry headers of yarn v1 and Berry lockfiles:
    `lodash@^4.17.0, lodash@^4.17.21:` / `"lodash@npm:^4.17.21":`."""
    names: set = set()
    for line in text.splitlines():
        if not line or line[0] in " \t#" or not line.rstrip().endswith(":"):
            continue
        for descriptor in line.rstrip()[:-1].split(","):
            name = _npm_name_version(descriptor.strip().strip('"'))[0]
            if name:
                names.add(name)
    return names


def _names_pnpm_lock(text: str) -> set:
    """Package keys of pnpm-lock.yaml (`/lodash@4.17.21:` in v6,
    `/lodash/4.17.21:` in v5, `lodash@4.17.21(peer):` in v9)."""
    names: set = set()
    section = ""
    for line in text.splitlines():
        if line and not line[0].isspace():
            section = line.rstrip(":").strip()
            continue
        m = re.match(r"^  (\S.*?):\s*$", line)
        if section not in ("packages", "snapshots") or not m:
            continue
        key = re.sub(r"\(.*$", "", m.group(1).strip("'\"")).lstrip("/")
        name, version = _npm_name_version(key)
        if not version and "/" in key.lstrip("@"):
            name = key.rsplit("/", 1)[0]  # v5: /name/version
        names.add(name)
    return names


def _names_bun_lock(text: str) -> set:
    """bun.lock: `"key": ["name@version", ...]` entries."""
    return {_npm_name_version(m.group(1))[0]
            for m in re.finditer(r'"[^"\n]+":\s*\[\s*"([^"]+@[^"]*)"', text)}


def _names_pipfile_lock(text: str) -> set:
    data = json.loads(text)
    return {name for group in ("default", "develop") for name in (data.get(group) or {})}


def _names_requirements(text: str) -> set:
    names: set = set()
    for line in text.splitlines():
        line = line.split("#", 1)[0].strip()
        m = re.match(r"^([A-Za-z0-9][A-Za-z0-9._-]*)", line)
        if m and not line.startswith("-"):
            names.add(m.group(1))
    return names


def _names_go_mod(text: str) -> set:
    names: set = set()
    in_block = False
    for line in text.splitlines():
        line = line.split("//", 1)[0].strip()
        if line.startswith("require ("):
            in_block = True
            continue
        if in_block and line == ")":
            in_block = False
            continue
        if line.startswith("require "):
            line = line[len("require "):]
        elif not in_block:
            continue
        if line:
            names.add(line.split()[0])
    return names


def _names_go_sum(text: str) -> set:
    return {line.split()[0] for line in text.splitlines() if line.strip()}


def _names_gemfile_lock(text: str) -> set:
    return {m.group(1) for m in re.finditer(r"(?m)^    ([A-Za-z0-9_.-]+) \(", text)}


def _names_composer_lock(text: str) -> set:
    data = json.loads(text)
    return {p.get("name", "") for group in ("packages", "packages-dev") for p in data.get(group) or []}


def _names_nuget_lock(text: str) -> set:
    data = json.loads(text)
    return {name for deps in (data.get("dependencies") or {}).values() for name in deps or {}}


def _names_pubspec_lock(text: str) -> set:
    names: set = set()
    in_packages = False
    for line in text.splitlines():
        if line and not line[0].isspace():
            in_packages = line.strip() == "packages:"
            continue
        m = re.match(r"^  ([A-Za-z0-9_]+):\s*$", line)
        if in_packages and m:
            names.add(m.group(1))
    return names


# Lockfiles (glob patterns relative to the project root) per OSV ecosystem.
LOCKFILES: dict = {
    "npm": (("package-lock.json", _names_package_lock), ("npm-shrinkwrap.json", _names_package_lock),
            ("yarn.lock", _names_yarn_lock), ("pnpm-lock.yaml", _names_pnpm_lock),
            ("bun.lock", _names_bun_lock)),
    "PyPI": (("uv.lock", _toml_package_names), ("poetry.lock", _toml_package_names),
             ("pdm.lock", _toml_package_names), ("pylock*.toml", _toml_package_names),
             ("Pipfile.lock", _names_pipfile_lock), ("requirements*.txt", _names_requirements)),
    "crates.io": (("Cargo.lock", _toml_package_names),),
    "Go": (("go.mod", _names_go_mod), ("go.sum", _names_go_sum)),
    "RubyGems": (("Gemfile.lock", _names_gemfile_lock),),
    "Packagist": (("composer.lock", _names_composer_lock),),
    "NuGet": (("packages.lock.json", _names_nuget_lock),),
    "Pub": (("pubspec.lock", _names_pubspec_lock),),
}


def is_in_lockfile(package_name: str, ecosystem: str) -> Optional[bool]:
    """
    Check whether a package is locked by any of the project's lockfiles.
    Returns True if found, False if a lockfile exists but does not lock the
    package (an unreadable or malformed lockfile counts as not locking it),
    None if the project has no lockfile for the ecosystem (skip check).
    """
    root = Path(os.environ.get("CLAUDE_PROJECT_DIR", "."))
    want = normalize_name(package_name, ecosystem)
    found_lockfile = False
    for pattern, parse in LOCKFILES.get(ecosystem, ()):
        for path in sorted(root.glob(pattern)):
            if not path.is_file():
                continue
            found_lockfile = True
            try:
                names = parse(path.read_text(errors="replace"))
            except (OSError, ValueError, AttributeError, TypeError):
                continue
            for name in names:
                locked = normalize_name(name, ecosystem)
                # `go get` takes package paths inside a locked module.
                if locked == want or (ecosystem == "Go" and want.startswith(locked + "/")):
                    return True
    return False if found_lockfile else None


# ---------------------------------------------------------------------------
# API callers
# ---------------------------------------------------------------------------

class UnresolvableSpec(Exception):
    """The version requirement cannot be evaluated by the guard (an unsupported
    range syntax or ecosystem). main() asks for confirmation instead of checking
    some other version than the one that would be installed."""


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
    with urllib.request.urlopen(req, timeout=_call_timeout()) as resp:
        return json.loads(resp.read().decode("utf-8"))


def _get_json(url: str):
    """GET a registry JSON document. Raises on HTTP/network/decode failure.
    A gzip body is decompressed: NuGet's registration hive is stored
    gzip-encoded and served that way whatever Accept-Encoding says, and
    urllib does not decode Content-Encoding itself."""
    req = urllib.request.Request(
        url,
        headers={
            "Accept": "application/json",
            # crates.io requires a User-Agent header.
            "User-Agent": "claude-code-package-guardrail/1.0",
        },
    )
    with urllib.request.urlopen(req, timeout=_call_timeout()) as resp:
        body = resp.read()
    if body[:2] == b"\x1f\x8b":  # gzip magic; a JSON document never starts with it
        body = gzip.decompress(body)
    return json.loads(body.decode("utf-8"))


def _parse_time(value: str) -> datetime:
    """Parse an ISO-8601 registry timestamp. Fractional seconds are cut to six
    digits (the Go proxy reports nanoseconds, which fromisoformat rejects on
    the Python versions qsdev targets)."""
    value = value.strip().replace("Z", "+00:00")
    value = re.sub(r"(\.\d{6})\d+", r"\1", value)
    parsed = datetime.fromisoformat(value)
    if parsed.tzinfo is None:
        parsed = parsed.replace(tzinfo=timezone.utc)
    return parsed


def _age_days(value: Optional[str]) -> float:
    """Age in days of a registry publication timestamp. A missing timestamp is
    an error (fail closed), never an implicit pass."""
    if not value:
        raise ValueError("registry response has no publication time for the resolved version")
    return (datetime.now(timezone.utc) - _parse_time(value)).total_seconds() / 86400


# ---------------------------------------------------------------------------
# Version requirements
# ---------------------------------------------------------------------------
#
# One small comparator engine evaluates the requirement syntaxes of npm
# (node-semver), Cargo, PEP 440 (plus Poetry's ^/~), RubyGems (~>) and Pub, so
# the guard checks the version the manager would actually pick instead of a
# string with the operators stripped off.

_REQ_TOKEN_RE = re.compile(
    r"(===|==|!=|~=|~>|>=|<=|\^|~|>|<|=)?"
    r"[vV]?((?:\d+|[xX*])(?:\.(?:\d+|[xX*]))*)"
    r"((?:-|\.?(?=[A-Za-z]))[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?"
)


def _vparse(text: str) -> Optional[tuple]:
    """Parse a version into (release tuple, pre-release tag). Returns None for
    strings that are not versions."""
    m = re.match(r"^[vV]?(\d+(?:\.\d+)*)(.*)$", text.strip())
    if not m:
        return None
    release = tuple(int(x) for x in m.group(1).split("."))
    pre = m.group(2).split("+", 1)[0].lstrip("-.")
    return release, pre


def _vkey(parsed: tuple) -> tuple:
    """Sort key: release padded to six parts, then pre-release < release <
    post-release."""
    release, pre = parsed
    padded = (release + (0,) * 6)[:6]
    if not pre:
        return (padded, 1, "")
    if pre.startswith("post"):
        return (padded, 2, pre)
    return (padded, 0, pre)


def _bound(parts: list) -> tuple:
    """Lowest key for a release prefix, below all of its pre-releases."""
    return ((tuple(parts) + (0,) * 6)[:6], -1, "")


def _bump(parts: list, idx: int) -> list:
    return list(parts[:idx]) + [parts[idx] + 1]


def _parse_requirement(expr: str, default_op: str) -> list:
    """Parse a requirement into alternatives (joined by ||), each a list of
    primitive (op, key) constraints. Raises UnresolvableSpec on syntax the
    engine does not understand."""
    alternatives = []
    for alt in expr.split("||"):
        alt = alt.strip()
        hyphen = re.fullmatch(r"(\S+)\s+-\s+(\S+)", alt)
        if hyphen:
            alt = f">={hyphen.group(1)} <={hyphen.group(2)}"
        alt = re.sub(r"(===|==|!=|~=|~>|>=|<=|\^|~|>|<|=)\s+", r"\1", alt)
        constraints: list = []
        for tok in re.split(r"[\s,]+", alt):
            if not tok or tok in ("*", "x", "X"):
                continue
            m = _REQ_TOKEN_RE.fullmatch(tok)
            if not m:
                raise UnresolvableSpec(f"unsupported version requirement '{expr}'")
            op = m.group(1) or default_op
            raw_parts = m.group(2).split(".")
            parts: list = []
            for p in raw_parts:
                if not p.isdigit():
                    break
                parts.append(int(p))
            wild = len(parts) < len(raw_parts)
            pre = (m.group(3) or "").lstrip("-.")
            partial = wild or (default_op in ("=", "^") and len(parts) < 3 and op in ("=", "^", "~", "<="))
            if not parts:
                if op in ("=", "==", ""):
                    continue  # '*' / 'x': any version
                raise UnresolvableSpec(f"unsupported version requirement '{expr}'")
            exact_key = _vkey((tuple(parts), pre))
            if op in ("=", "==", "==="):
                if partial:
                    constraints += [(">=", _bound(parts)), ("<", _bound(_bump(parts, len(parts) - 1)))]
                else:
                    constraints.append(("==", exact_key))
            elif op == "!=":
                if wild:
                    constraints.append(("!range", (_bound(parts), _bound(_bump(parts, len(parts) - 1)))))
                else:
                    constraints.append(("!=", exact_key))
            elif op == "^":
                full = (parts + [0, 0, 0])[:3]
                idx = next((k for k, v in enumerate(full) if v != 0), 2)
                if len(parts) <= idx:
                    idx = len(parts) - 1
                constraints += [(">=", _bound(parts)), ("<", _bound(_bump(full, idx)))]
            elif op == "~":
                idx = 0 if len(parts) == 1 else 1
                constraints += [(">=", _bound(parts)), ("<", _bound(_bump(parts, idx)))]
            elif op in ("~=", "~>"):
                if len(parts) < 2:
                    raise UnresolvableSpec(f"unsupported version requirement '{expr}'")
                constraints += [(">=", _bound(parts) if not pre else exact_key),
                                ("<", _bound(_bump(parts, len(parts) - 2)))]
            elif op == ">=":
                constraints.append((">=", _bound(parts) if not pre else exact_key))
            elif op == ">":
                constraints.append((">", exact_key))
            elif op == "<":
                constraints.append(("<", _bound(parts) if not pre else exact_key))
            elif op == "<=":
                if partial:
                    constraints.append(("<", _bound(_bump(parts, len(parts) - 1))))
                else:
                    constraints.append(("<=", exact_key))
        alternatives.append(constraints)
    return alternatives


def _satisfies(key: tuple, alternatives: list) -> bool:
    for constraints in alternatives:
        ok = True
        for op, bound in constraints:
            if op == "==":
                ok = key == bound
            elif op == "!=":
                ok = key != bound
            elif op == "!range":
                ok = not (bound[0] <= key < bound[1])
            elif op == ">=":
                ok = key >= bound
            elif op == ">":
                ok = key > bound
            elif op == "<":
                ok = key < bound
            elif op == "<=":
                ok = key <= bound
            if not ok:
                break
        if ok:
            return True
    return False


def _pick_version(candidates, requirement: str, default_op: str) -> str:
    """Highest non-pre-release candidate satisfying `requirement`. Raises
    ValueError when nothing matches (the install would fail or pick something
    the guard cannot see)."""
    alternatives = _parse_requirement(requirement, default_op)
    best = None
    for version in candidates:
        parsed = _vparse(version)
        if parsed is None or (parsed[1] and not parsed[1].startswith("post")):
            continue
        key = _vkey(parsed)
        if _satisfies(key, alternatives) and (best is None or key > best[0]):
            best = (key, version)
    if best is None:
        raise ValueError(f"no published version satisfies '{requirement}'")
    return best[1]


def _match_exact(candidates, version: str) -> str:
    """The published version equal to `version` (PEP 440 style padding, so
    2.31 == 2.31.0). Raises ValueError when it is not published."""
    want = _vparse(version)
    for candidate in candidates:
        if candidate == version:
            return candidate
    if want is not None:
        for candidate in candidates:
            parsed = _vparse(candidate)
            if parsed is not None and _vkey(parsed) == _vkey(want):
                return candidate
    raise ValueError(f"version '{version}' is not published")


# ---------------------------------------------------------------------------
# Registry resolution: the version a manager would install, and its age
# ---------------------------------------------------------------------------

def _resolve_npm(name: str, kind: str, value: str) -> tuple:
    data = _get_json(f"https://registry.npmjs.org/{urllib.request.quote(name, safe='@/')}")
    versions = data.get("versions") or {}
    times = data.get("time") or {}
    dist_tags = data.get("dist-tags") or {}
    if kind in ("none", "tag"):
        version = dist_tags.get(value or "latest")
        if not version:
            raise ValueError(f"registry has no dist-tag '{value or 'latest'}'")
    elif kind == "exact":
        version = _match_exact(versions.keys(), value)
    else:
        version = _pick_version(versions.keys(), value, "=")
    return name, version, _age_days(times.get(version))


def _resolve_pypi(name: str, kind: str, value: str) -> tuple:
    data = _get_json(f"https://pypi.org/pypi/{urllib.request.quote(name)}/json")
    releases = data.get("releases") or {}
    live = [v for v, files in releases.items() if files and not all(f.get("yanked") for f in files)]
    if kind == "none":
        version = (data.get("info") or {}).get("version")
        if not version:
            raise ValueError("PyPI response has no current version")
    elif kind == "exact":
        version = _match_exact(releases.keys(), value)
    else:
        version = _pick_version(live, value, "==")
    uploads = [f.get("upload_time_iso_8601") for f in releases.get(version, []) if f.get("upload_time_iso_8601")]
    return name, version, _age_days(max(uploads, key=_parse_time) if uploads else None)


def _resolve_crates(name: str, kind: str, value: str) -> tuple:
    data = _get_json(f"https://crates.io/api/v1/crates/{urllib.request.quote(name)}")
    crate = data.get("crate") or {}
    entries = {v.get("num"): v for v in data.get("versions") or [] if v.get("num")}
    live = [num for num, v in entries.items() if not v.get("yanked")]
    if kind == "none":
        version = crate.get("max_stable_version") or crate.get("newest_version")
    elif kind == "exact":
        version = _match_exact(entries.keys(), value)
    else:
        version = _pick_version(live, value, "^")
    if not version or version not in entries:
        raise ValueError("crates.io response does not list the resolved version")
    return name, version, _age_days(entries[version].get("created_at"))


def _go_escape(path: str) -> str:
    """Module path escaping for the Go module proxy (uppercase -> !lower)."""
    return re.sub(r"[A-Z]", lambda m: "!" + m.group(0).lower(), path)


def _resolve_go(name: str, kind: str, value: str) -> tuple:
    """Resolve a package path to its module and version via the Go module
    proxy. `go get` accepts package paths inside a module, so the path is
    trimmed one element at a time until the proxy knows it."""
    path = re.sub(r"/\.\.\.$", "", name)
    if kind == "range":
        raise UnresolvableSpec(f"version query '{value}' for {name}")
    elements = path.split("/")
    while elements:
        module = "/".join(elements)
        base = f"https://proxy.golang.org/{_go_escape(module)}"
        url = f"{base}/@latest" if kind == "none" else f"{base}/@v/{urllib.request.quote(value)}.info"
        try:
            info = _get_json(url)
        except urllib.error.HTTPError as e:
            if e.code in (404, 410) and len(elements) > 1:
                elements.pop()
                continue
            raise
        version = info.get("Version")
        if not version:
            raise ValueError("Go proxy response has no Version")
        return module, version.lstrip("v"), _age_days(info.get("Time"))
    raise ValueError(f"no module found for '{name}'")


def _resolve_rubygems(name: str, kind: str, value: str) -> tuple:
    entries = _get_json(f"https://rubygems.org/api/v1/versions/{urllib.request.quote(name)}.json")
    by_number = {}
    for e in entries:
        by_number.setdefault(e.get("number"), e)
    if kind == "none":
        stable = [n for n, e in by_number.items() if n and not e.get("prerelease")]
        version = _pick_version(stable, ">=0", "=")
    elif kind == "exact":
        version = _match_exact(by_number.keys(), value)
    else:
        version = _pick_version(by_number.keys(), value, "=")
    return name, version, _age_days(by_number[version].get("created_at"))


def _resolve_packagist(name: str, kind: str, value: str) -> tuple:
    if kind in ("range", "tag"):
        raise UnresolvableSpec(f"Composer constraint '{value}'")
    data = _get_json(f"https://repo.packagist.org/p2/{name.lower()}.json")
    entries = (data.get("packages") or {}).get(name.lower()) or []
    published = {}
    last_time = None
    for e in entries:  # minified format: unchanged keys are omitted
        last_time = e.get("time", last_time)
        if e.get("version"):
            published[e["version"]] = last_time
    stable = [v for v in published if re.fullmatch(r"v?\d+(\.\d+)*", v)]
    if kind == "none":
        if not stable:
            raise ValueError("Packagist lists no stable release")
        version = _pick_version(stable, ">=0", "=")
    else:
        version = _match_exact(published.keys(), value)
    return name, version, _age_days(published.get(version))


# nuget.org's registration hive with SemVer 2.0.0 packages included (the
# hive NuGet clients use). Unlike the flat container it carries each version's
# publication time and listed state.
_NUGET_REGISTRATION = "https://api.nuget.org/v3/registration5-gz-semver2"
# NuGet floating versions the guard evaluates: `*`, `1.*` (newest stable under
# the prefix) and `*-*`, `1.*-*` (newest including pre-releases). NuGet
# resolves an interval such as `[1.0,2.0)` to its LOWEST match instead, which
# the guard leaves to the user.
_NUGET_FLOAT_RE = re.compile(r"^((?:\d+\.)*)\*(-\*)?$")
# The floating version `--prerelease` selects: dotnet adds the newest release,
# pre-releases included.
NUGET_PRERELEASE_FLOAT = "*-*"


def _nuget_key(version: str) -> Optional[tuple]:
    """SemVer 2.0.0 precedence key for a NuGet version (build metadata
    ignored, pre-release labels compared case-insensitively and numerically
    where numeric, so 1.0.0-beta.10 > 1.0.0-beta.9). None when not a version."""
    base = version.strip().split("+", 1)[0]
    release, dash, pre = base.partition("-")
    if not re.fullmatch(r"[0-9]+(?:\.[0-9]+){0,3}", release):  # NuGet versions have at most 4 parts
        return None
    padded = (tuple(int(x) for x in release.split(".")) + (0,) * 4)[:4]
    if not dash:
        return padded, 1, ()
    labels = tuple((0, int(p), "") if p.isdigit() else (1, 0, p.lower()) for p in pre.split("."))
    return padded, 0, labels


def _nuget_page_holds(page: dict, key: tuple) -> bool:
    lower, upper = _nuget_key(page.get("lower") or ""), _nuget_key(page.get("upper") or "")
    return lower is None or upper is None or lower <= key <= upper


def _nuget_pages(name: str, want: Optional[tuple] = None):
    """Yield the leaves of each page of a package's registration index,
    newest page first. Pages of large packages are not inlined and are
    fetched on demand, skipping those whose version bounds exclude `want`. A
    malformed index raises (fail closed)."""
    index = _get_json(f"{_NUGET_REGISTRATION}/{urllib.request.quote(name.lower(), safe='')}/index.json")
    for page in reversed(index.get("items") or []):
        leaves = page.get("items")
        if leaves is None:
            if want is not None and not _nuget_page_holds(page, want):
                continue
            leaves = _get_json(page["@id"]).get("items") or []
        yield leaves


def _nuget_find(name: str, kind: str, value: str) -> dict:
    """The catalog entry of the version the install would pick."""
    if kind == "exact":
        want = _nuget_key(value)
        if want is None:
            raise UnresolvableSpec(f"NuGet version '{value}'")
        for leaves in _nuget_pages(name, want):
            for leaf in leaves:
                entry = leaf.get("catalogEntry") or {}
                if _nuget_key(entry.get("version") or "") == want:
                    return entry
        raise ValueError(f"version '{value}' is not published")
    if kind == "none":
        prefix, prerelease = (), False
    else:
        m = _NUGET_FLOAT_RE.match(value.strip()) if kind == "range" else None
        if not m:
            raise UnresolvableSpec(f"NuGet version range '{value}'")
        prefix = tuple(int(x) for x in m.group(1).split(".") if x)
        prerelease = bool(m.group(2))
    # Pages are ordered and disjoint, so the first page (newest first) with a
    # match holds the newest match. Unlisted versions are never picked for a
    # latest or floating version.
    for leaves in _nuget_pages(name):
        best = None
        for leaf in leaves:
            entry = leaf.get("catalogEntry") or {}
            key = _nuget_key(entry.get("version") or "")
            if key is None or entry.get("listed") is False or (key[1] == 0 and not prerelease):
                continue
            if key[0][:len(prefix)] == prefix and (best is None or key > best[0]):
                best = (key, entry)
        if best is not None:
            return best[1]
    raise ValueError("NuGet lists no " + ("release" if prerelease else "stable release")
                     + (f" matching '{value}'" if value else ""))


def _nuget_published(entry: dict) -> Optional[str]:
    """Publication time of a registration catalog entry. nuget.org stamps an
    unlisted version's `published` with 1900-01-01, which would pass any age
    gate, so its catalog leaf's `created` time is used instead."""
    published = entry.get("published")
    if entry.get("listed") is False or (published and _parse_time(published).year <= 1900):
        return _get_json(entry["@id"]).get("created")
    return published


def _resolve_nuget(name: str, kind: str, value: str) -> tuple:
    entry = _nuget_find(name, kind, value)
    version = entry.get("version")
    if not version:
        raise ValueError("NuGet registration entry has no version")
    return entry.get("id") or name, version, _age_days(_nuget_published(entry))


def _resolve_pub(name: str, kind: str, value: str) -> tuple:
    data = _get_json(f"https://pub.dev/api/packages/{urllib.request.quote(name)}")
    published = {v.get("version"): v.get("published") for v in data.get("versions") or []}
    if kind == "none":
        version = (data.get("latest") or {}).get("version")
        if not version:
            raise ValueError("pub.dev response has no latest version")
    elif kind == "exact":
        version = _match_exact(published.keys(), value)
    else:
        version = _pick_version(published.keys(), value, "=")
    return name, version, _age_days(published.get(version))


_RESOLVERS = {
    "npm": _resolve_npm,
    "PyPI": _resolve_pypi,
    "crates.io": _resolve_crates,
    "Go": _resolve_go,
    "RubyGems": _resolve_rubygems,
    "Packagist": _resolve_packagist,
    "NuGet": _resolve_nuget,
    "Pub": _resolve_pub,
}


def resolve_package(name: str, ecosystem: str, kind: str, value: str) -> tuple:
    """Return (canonical name, version the manager would install, age in days
    of that version or None when the registry publishes no dates). Raises
    UnresolvableSpec when the requirement cannot be evaluated, and network /
    decode errors (which fail closed) otherwise."""
    resolver = _RESOLVERS.get(ecosystem)
    if resolver is None:
        raise UnresolvableSpec(f"no registry resolver for {ecosystem}")
    return resolver(name, kind, value)


# ---------------------------------------------------------------------------
# Package specifier parsing
# ---------------------------------------------------------------------------

_EXACT_SEMVER_RE = re.compile(r"^[=vV]?\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$")
_DIST_TAG_RE = re.compile(r"^[A-Za-z][A-Za-z0-9._-]*$")
_PEP508_NAME_RE = re.compile(r"^([A-Za-z0-9](?:[A-Za-z0-9._-]*[A-Za-z0-9])?)\s*(\[[^\]]*\])?\s*(.*)$")


def _npm_name_version(spec: str) -> tuple:
    """Split an npm spec into (name, version part) after an optional scope."""
    at = spec.find("@", 1 if spec.startswith("@") else 0)
    return (spec, "") if at < 0 else (spec[:at], spec[at + 1:])


def parse_spec(ecosystem: str, manager: str, spec: str) -> tuple:
    """Parse a registry package specifier (already classified as a registry
    package by the detector) into (name, kind, value, alias). kind is one of
    none / exact / range / tag / remove; alias is the npm alias name for
    `alias@npm:target` (the target is what is installed and validated)."""
    alias = ""
    if ecosystem == "npm":
        name, ver = _npm_name_version(spec)
        if ver.startswith("npm:"):
            alias = name
            name, ver = _npm_name_version(ver[4:])
        ver = ver.strip()
        if not ver:
            return name, "none", "", alias
        if _EXACT_SEMVER_RE.match(ver):
            return name, "exact", ver.lstrip("=vV"), alias
        if _DIST_TAG_RE.match(ver) and ver.lower() != "x":
            return name, "tag", ver, alias
        return name, "range", ver, alias
    if ecosystem == "PyPI":
        m = _PEP508_NAME_RE.match(spec.split(";", 1)[0].strip())
        if not m:
            raise UnresolvableSpec(f"'{spec}' is not a PEP 508 requirement")
        name, rest = m.group(1), m.group(3).strip()
        if rest.startswith("@"):  # Poetry's name@constraint form
            rest = rest[1:].strip()
            if _vparse(rest) is not None and re.fullmatch(r"[\d.]+", rest):
                rest = "==" + rest
        if not rest:
            return name, "none", "", alias
        exact = re.fullmatch(r"(?:===?)\s*([^,*\s]+)", rest)
        if exact:
            return name, "exact", exact.group(1), alias
        return name, "range", rest, alias
    if ecosystem == "Go":
        name, _, ver = spec.partition("@")
        if ver in ("", "latest", "upgrade", "patch"):
            return name, "none", "", alias
        if ver == "none":
            return name, "remove", "", alias
        if re.match(r"^v\d+\.\d+\.\d+", ver):
            return name, "exact", ver, alias
        if ver[:1] in "<>":
            return name, "range", ver, alias
        return name, "tag", ver, alias
    if ecosystem == "NuGet":
        spec = spec.replace("::", "@", 1)  # `dotnet new install Pkg::1.0.0`
    sep = {"RubyGems": ":", "Packagist": ":", "Pub": ":"}.get(ecosystem, "@")
    name, _, ver = spec.partition(sep)
    if ecosystem == "Packagist" and not ver:
        for alt in ("=", " "):
            if alt in name:
                name, _, ver = name.partition(alt)
                break
    if ecosystem == "Pub":
        name = re.sub(r"^(?:dev|override):", "", name)
    ver = ver.strip().strip("'\"")
    if not ver:
        return name, "none", "", alias
    if ecosystem == "crates.io":
        if ver.startswith("=") and _EXACT_SEMVER_RE.match(ver[1:]):
            return name, "exact", ver[1:], alias
        # `cargo install x@1.2.3` is exact; `cargo add x@1.2.3` writes the
        # caret requirement ^1.2.3 to Cargo.toml.
        if manager == "cargo" and _EXACT_SEMVER_RE.match(ver):
            return name, "exact", ver, alias
        return name, "range", ver, alias
    if re.fullmatch(r"[vV]?\d+(?:\.\d+)*(?:[-.][0-9A-Za-z.]+)?", ver):
        return name, "exact", ver, alias
    return name, "range", ver, alias


# ---------------------------------------------------------------------------
# Shell lexing
# ---------------------------------------------------------------------------
#
# Package names are extracted from the *argv* of a genuine install invocation,
# never from arbitrary substrings of the command line: each command segment is
# tokenized like the shell does (quotes, escapes, ANSI-C strings, redirections)
# and the executable + subcommand verb must actually BE an install before any
# operand is read. A quoted argument such as "npm install foo" is one word, so
# it can never be read as an `npm` executable followed by an `install` verb.
#
# The guard never evaluates expansions. A word whose value the shell computes
# ($var, $(...), globs, brace expansion) is marked dynamic, and a dynamic word
# in a position that decides what runs (the command name, an install verb, a
# package operand) is treated as unverifiable instead of as literal text.

# Characters that end a heredoc delimiter word (`cat <<EOF;` / `<<EOF)`).
_HEREDOC_WORD_END = set(" \t\r\n;&|<>()")

# Stands in for every command/process substitution span removed by
# _extract_substitutions. It keeps offsets aligned with the original command
# and marks the words it lands in as dynamic.
_SUBST = "\0"

_REDIRECT_OPS = ("&>>", "&>", "<<<", "<<-", "<<", "<>", "<&", ">&", ">>", ">|", "<", ">")
_FD_PREFIX_RE = re.compile(r"\d+|\{[A-Za-z_][A-Za-z0-9_]*\}")
_BRACE_EXPANSION_RE = re.compile(r"\{[^{}]*(?:,|\.\.)[^{}]*\}")
_BRACKET_GLOB_RE = re.compile(r"\[[^\]]+\]")

_ANSI_C_ESCAPES = {
    "a": "\a", "b": "\b", "e": "\x1b", "E": "\x1b", "f": "\f", "n": "\n",
    "r": "\r", "t": "\t", "v": "\v", "\\": "\\", "'": "'", '"': '"', "?": "?",
}


class Word(NamedTuple):
    text: str        # the word after quote removal (expansions left literal)
    start: int       # offset of the word in the segment text
    end: int
    dynamic: bool    # the shell computes (part of) this word
    bracket: bool    # contains an unquoted [..] glob


class Segment(NamedTuple):
    text: str
    heredocs: list   # heredoc bodies opened by this segment
    piped: bool      # stdin is the previous pipeline stage
    start: int       # offset of `text` in the command
    verbatim: bool   # `text` is command[start:start+len(text)] unchanged


def _read_ansi_c(s: str, i: int) -> tuple:
    """Decode a $'...' ANSI-C string whose body starts at s[i]. Returns
    (decoded text, index past the closing quote)."""
    out: list = []
    n = len(s)
    while i < n:
        c = s[i]
        if c == "'":
            return "".join(out), i + 1
        if c != "\\" or i + 1 >= n:
            out.append(c)
            i += 1
            continue
        e = s[i + 1]
        if e in _ANSI_C_ESCAPES:
            out.append(_ANSI_C_ESCAPES[e])
            i += 2
        elif e in "01234567":
            m = re.match(r"[0-7]{1,3}", s[i + 1:])
            out.append(chr(int(m.group(0), 8) & 0xFF))
            i += 1 + len(m.group(0))
        elif e in "xuU":
            width = {"x": 2, "u": 4, "U": 8}[e]
            m = re.match(r"[0-9A-Fa-f]{1,%d}" % width, s[i + 2:])
            if m:
                out.append(chr(int(m.group(0), 16)))
                i += 2 + len(m.group(0))
            else:
                out.append("\\" + e)
                i += 2
        elif e == "c" and i + 2 < n:
            out.append(chr(ord(s[i + 2]) & 0x1F))
            i += 3
        else:
            out.append("\\" + e)
            i += 2
    raise ValueError("unterminated $'...' string")


def _read_word(s: str, i: int) -> tuple:
    """Read one shell word starting at s[i]. Returns (text, mask, end) where
    mask mirrors the raw characters with quoted/escaped ones replaced by '_',
    except that `$`/backtick inside double quotes stay (they still expand).
    Stops at a blank or an unquoted redirection operator."""
    n = len(s)
    out: list = []
    mask: list = []
    while i < n:
        c = s[i]
        if c in " \t\r\n" or c in "<>" or (c == "&" and s[i + 1:i + 2] == ">"):
            break
        if c == "\\":
            if i + 1 < n:
                if s[i + 1] != "\n":
                    out.append(s[i + 1])
                    mask.append("_")
                i += 2
            else:
                out.append(c)
                mask.append("_")
                i += 1
            continue
        if c == "'":
            j = s.find("'", i + 1)
            if j < 0:
                raise ValueError("unbalanced single quote")
            out.append(s[i + 1:j])
            mask.append("_" * (j - i - 1))
            i = j + 1
            continue
        if c == "$" and s[i + 1:i + 2] == "'":
            text, i = _read_ansi_c(s, i + 2)
            out.append(text)
            mask.append("_" * len(text))
            continue
        if c == "$" and s[i + 1:i + 2] == '"':
            i += 1
            c = '"'
        if c == '"':
            i += 1
            while True:
                if i >= n:
                    raise ValueError("unbalanced double quote")
                ch = s[i]
                if ch == '"':
                    i += 1
                    break
                if ch == "\\" and i + 1 < n and s[i + 1] in '$`"\\\n':
                    if s[i + 1] != "\n":
                        out.append(s[i + 1])
                        mask.append("_")
                    i += 2
                    continue
                out.append(ch)
                mask.append(ch if ch in "$`" + _SUBST else "_")
                i += 1
            continue
        out.append(c)
        mask.append(c)
        i += 1
    return "".join(out), "".join(mask), i


def _mask_dynamic(mask: str) -> bool:
    return (
        _SUBST in mask or "`" in mask or "*" in mask or "?" in mask
        or re.search(r"\$.", mask) is not None
        or _BRACE_EXPANSION_RE.search(mask) is not None
    )


def _tokenize(s: str) -> tuple:
    """Split a segment into Words, dropping redirections (`2>&1`, `>log`,
    `<in`, `<<EOF`) the way the shell does. Returns (words, herestrings) where
    herestrings are the `<<< word` operands as (text, dynamic). Raises
    ValueError on unbalanced quoting."""
    words: list = []
    heres: list = []
    i, n = 0, len(s)
    while i < n:
        if s[i] in " \t\r\n":
            i += 1
            continue
        start = i
        text, mask, j = _read_word(s, i)
        if j < n and (s[j] in "<>" or s[j] == "&"):
            if j > start and not _FD_PREFIX_RE.fullmatch(s[start:j]):
                words.append(Word(text, start, j, _mask_dynamic(mask), bool(_BRACKET_GLOB_RE.search(mask))))
            op = next(o for o in _REDIRECT_OPS if s.startswith(o, j))
            j += len(op)
            while j < n and s[j] in " \t":
                j += 1
            target, tmask, k = _read_word(s, j)
            if op == "<<<" and k > j:
                heres.append((target, _mask_dynamic(tmask)))
            i = max(k, j)
            continue
        words.append(Word(text, start, j, _mask_dynamic(mask), bool(_BRACKET_GLOB_RE.search(mask))))
        i = j
    return words, heres


def _read_heredoc_delimiter(command: str, i: int) -> tuple:
    """Read the heredoc delimiter word starting at command[i] (just past `<<` /
    `<<-` and any blanks). Quotes and backslashes are removed, as the shell does
    when matching the terminator line. Returns (delimiter, index past the word,
    quoted) -- a quoted delimiter makes the body literal (no expansions).
    """
    delim: list = []
    quoted = False
    n = len(command)
    while i < n and command[i] not in _HEREDOC_WORD_END:
        c = command[i]
        if c in ("'", '"'):
            quoted = True
            j = command.find(c, i + 1)
            if j == -1:
                j = n
            delim.append(command[i + 1:j])
            i = j + 1
            continue
        if c == "\\" and i + 1 < n:
            quoted = True
            delim.append(command[i + 1])
            i += 2
            continue
        delim.append(c)
        i += 1
    return "".join(delim), min(i, n), quoted


def _read_heredoc_body(command: str, i: int, delim: str, strip_tabs: bool) -> tuple:
    """Read a heredoc body starting at command[i] (the first character after the
    newline that ends the introducing line), up to the line equal to `delim`.
    Returns (body, index past the terminator line). An unterminated body runs to
    end-of-string, as in the shell."""
    lines: list = []
    n = len(command)
    while i < n:
        j = command.find("\n", i)
        end = n if j == -1 else j
        line = command[i:end]
        i = n if j == -1 else j + 1
        check = line.rstrip("\r")
        if strip_tabs:
            check = check.lstrip("\t")
        if check == delim:
            break
        lines.append(line)
    return "\n".join(lines), i


def _split_segments(command: str) -> list:
    """Split a (possibly compound or multi-line) command into independent
    Segments. Each stage is a separate command validated on its own --
    `echo x | npm install evil` and a multi-line `echo hi\\nnpm install evil`
    both still check the install.

    The scan is quote-aware: control operators (&&, ||, ;, |, |&, &), subshell
    parentheses and newlines split segments only OUTSIDE single/double quotes and
    when not backslash-escaped, so a quoted operator (`jq '.a | .b'`,
    `git commit -m "a; b"`, a multi-line -m message) stays inside its argument.
    Redirections that contain `&` or `|` (`2>&1`, `&>`, `>|`) are not operators.
    Backslash-newline line continuations are removed, as the shell does (the
    segment is then no longer a verbatim slice of the command).

    Heredoc bodies (`<<EOF` ... `EOF`) are data rather than command text, so they
    are returned separately, attached to the segment that opened them; the caller
    decides whether they are scripts. An unbalanced quote leaves the remainder in
    one segment, which the tokenizer then rejects so the caller fails closed.

    Because a heredoc hides the lines that follow it, `<<` is only treated as one
    where the shell does: never inside a `#` comment, `((...))` / `$[...]`
    arithmetic or a `${...}` expansion.
    """
    segments: list = []
    buf: list = []
    # Heredocs opened on the current line: (delimiter, strip_tabs, segment index).
    pending: list = []
    # Closers of the open arithmetic/expansion contexts in which `<<` is a shift
    # operator or literal text rather than a heredoc. Leaving one open by mistake
    # only suppresses heredoc detection, so more text is scanned, never less.
    no_heredoc: list = []
    quote: Optional[str] = None
    i, n = 0, len(command)
    state = {"start": 0, "verbatim": True, "piped": False, "next_piped": False}

    def begin(at: int) -> None:
        if not buf:
            state["start"] = at
            state["verbatim"] = True

    def flush(next_piped: bool = False) -> None:
        raw = "".join(buf)
        buf.clear()
        text = raw.strip()
        if text:
            lead = len(raw) - len(raw.lstrip())
            segments.append(Segment(text, [], state["piped"], state["start"] + lead, state["verbatim"]))
        state["piped"] = next_piped

    while i < n:
        c = command[i]
        nxt = command[i + 1] if i + 1 < n else ""

        if quote == "'":
            buf.append(c)
            if c == "'":
                quote = None
            i += 1
            continue

        if c == "\\":
            if nxt == "\n" or (nxt == "\r" and command[i + 2:i + 3] == "\n"):
                state["verbatim"] = False  # line continuation: removed outside single quotes
                i += 2 if nxt == "\n" else 3
                continue
            begin(i)
            buf.append(command[i:i + 2])
            i += 2
            continue

        if quote == '"':
            buf.append(c)
            if c == '"':
                quote = None
            i += 1
            continue

        if c in ("'", '"'):
            begin(i)
            quote = c
            buf.append(c)
            i += 1
            continue

        # A `#` starting a word comments out the rest of the line (no line
        # continuation). An escaped blank (`a\ #`) is a 2-char entry in buf, so
        # it does not count as a word break.
        if c == "#" and (not buf or buf[-1] in (" ", "\t")):
            while i < n and command[i] not in "\r\n":
                i += 1
            continue

        if c == "$" and nxt in ("{", "["):
            begin(i)
            no_heredoc.append("}" if nxt == "{" else "]")
            buf.append(c + nxt)
            i += 2
            continue
        if c == "(" and nxt == "(":
            no_heredoc.append("))")
            flush()
            i += 2
            continue
        if no_heredoc:
            closer = no_heredoc[-1]
            if closer != "))" and c == {"}": "{", "]": "["}[closer]:
                no_heredoc.append(closer)  # nested brace/bracket
            elif command.startswith(closer, i):
                no_heredoc.pop()
                if closer == "))":
                    flush()
                else:
                    buf.append(c)
                i += len(closer)
                continue

        if c == "<" and nxt == "<" and not no_heredoc:
            begin(i)
            if command[i + 2:i + 3] == "<":
                buf.append("<<<")  # here-string, not a heredoc
                i += 3
                continue
            j = i + 2
            strip_tabs = command[j:j + 1] == "-"
            if strip_tabs:
                j += 1
            while j < n and command[j] in " \t":
                j += 1
            delim, j, _ = _read_heredoc_delimiter(command, j)
            buf.append(command[i:j])
            if delim:
                # This segment's buffer is non-empty, so it lands at this index.
                pending.append((delim, strip_tabs, len(segments)))
            i = j
            continue

        if c in "\r\n":
            flush()
            i += 2 if (c == "\r" and nxt == "\n") else 1
            for delim, strip_tabs, seg_idx in pending:
                body, i = _read_heredoc_body(command, i, delim, strip_tabs)
                if seg_idx < len(segments):
                    segments[seg_idx].heredocs.append(body)
            pending.clear()
            continue

        prev = buf[-1][-1:] if buf else ""
        if c == "&" and (prev in ("<", ">") or nxt == ">"):
            begin(i)
            buf.append(c)  # redirection: 2>&1, <&3, &>file, &>>file
            i += 1
            continue
        if c == "|" and prev == ">":
            buf.append(c)  # >| noclobber override
            i += 1
            continue
        if c in ";&|()":
            pipe = c == "|" and nxt != "|"
            flush(next_piped=pipe)
            i += 2 if (c in "&|" and nxt in ("&", "|")) or (c == ";" and nxt == ";") else 1
            continue

        begin(i)
        buf.append(c)
        i += 1

    flush()
    return segments


def _extract_substitutions(text: str) -> tuple:
    """Pull command/process substitutions out of `text`, returning
    (inner_scripts, cleaned_text). Each `$(...)`, backtick `` `...` ``, `<(...)`
    and `>(...)` is a shell command in its own right and must be scanned; its
    span is replaced by the same number of _SUBST characters, so the cleaned
    text keeps the original offsets, the substitution's inner operators do not
    fragment surrounding segments, and the word it sits in reads as dynamic.

    Only spans the shell would execute are extracted: single-quoted text,
    backslash-escaped characters, `#` comments and the bodies of heredocs
    with a quoted delimiter (<<'EOF') are literal. An unbalanced construct runs
    to end-of-string (fail closed: the remainder is still scanned).
    """
    # Fast path: this runs on EVERY hook invocation, and the overwhelmingly
    # common command contains no substitution at all.
    if "`" not in text and "$(" not in text and "<(" not in text and ">(" not in text:
        return [], text
    scripts: list = []
    out = list(text)
    literal_heredocs: list = []  # quoted-delimiter heredocs opened on this line
    i, n = 0, len(text)
    quote: Optional[str] = None
    while i < n:
        c = text[i]
        if quote == "'":
            if c == "'":
                quote = None
            i += 1
            continue
        if c == "\\":
            i += 2
            continue
        if quote is None:
            if c == "'":
                quote = "'"
                i += 1
                continue
            if c == "$" and text[i + 1:i + 2] == "'":
                _, i = _read_ansi_c(text, i + 2) if "'" in text[i + 2:] else ("", n)
                continue
            if c == "#" and (i == 0 or text[i - 1] in " \t\n;&|("):
                while i < n and text[i] != "\n":
                    i += 1
                continue
            if c == "<" and text.startswith("<<", i) and not text.startswith("<<<", i):
                j = i + 2 + (1 if text[i + 2:i + 3] == "-" else 0)
                while j < n and text[j] in " \t":
                    j += 1
                delim, j, quoted = _read_heredoc_delimiter(text, j)
                if quoted and delim:
                    literal_heredocs.append((delim, text[i + 2:i + 3] == "-"))
                i = j
                continue
            if c == "\n" and literal_heredocs:
                i += 1
                for delim, strip_tabs in literal_heredocs:
                    _, i = _read_heredoc_body(text, i, delim, strip_tabs)
                literal_heredocs.clear()
                continue
        if c == '"':
            quote = None if quote == '"' else '"'
            i += 1
            continue
        if c == "`":
            j = text.find("`", i + 1)
            end = n if j == -1 else j + 1
            scripts.append(text[i + 1:end - 1] if j != -1 else text[i + 1:])
            out[i:end] = _SUBST * (end - i)
            i = end
            continue
        if text[i + 1:i + 2] == "(" and (c == "$" or (c in "<>" and quote is None)):
            # Skip the opening `$(` / `<(` / `>(` and find the matching `)`,
            # counting nested parens so inner substitutions stay intact.
            start = i + 2
            depth = 1
            j = start
            while j < n and depth > 0:
                if text[j] == "(":
                    depth += 1
                elif text[j] == ")":
                    depth -= 1
                j += 1
            scripts.append(text[start:j - 1] if depth == 0 else text[start:])
            out[i:j] = _SUBST * (j - i)
            i = j
            continue
        i += 1
    return scripts, "".join(out)


# ---------------------------------------------------------------------------
# Package manager grammar
# ---------------------------------------------------------------------------
#
# Every manager gets its own option table taken from its CLI reference, with
# the arity of each option: a boolean flag never swallows the package that
# follows it (`npm install --save-exact pkg`), and a value option never leaks
# its value into the package list (`npm install --omit dev`). An option the
# table does not know, placed before an operand, makes the parse ambiguous and
# is sent to the user (ask) instead of guessed.

class OptSpec(NamedTuple):
    value: frozenset     # options that consume the next token as their value
    boolean: frozenset   # options that take no value
    source: frozenset    # value options selecting a non-registry package source
    local: frozenset     # value options naming a local path (a URL there is a source)
    version: frozenset   # value options giving the version of the named package(s)
    pkg: frozenset       # value options naming the package(s) to run (replace the command operand)
    extra: frozenset     # value options naming additional packages
    reqfile: frozenset   # value options naming a requirements/constraints file
    script: frozenset    # value options whose value is a shell script
    prerelease: frozenset  # boolean options making an unpinned NuGet install pick the newest pre-release


def _opts(value: str = "", boolean: str = "", source: str = "", local: str = "",
          version: str = "", pkg: str = "", extra: str = "", reqfile: str = "",
          script: str = "", prerelease: str = "", base: Optional[OptSpec] = None) -> OptSpec:
    fields = [frozenset(x.split()) for x in (value, boolean + " " + prerelease, source, local, version, pkg, extra,
                                             reqfile, script, prerelease)]
    if base is not None:
        fields = [f | b for f, b in zip(fields, base)]
    return OptSpec(*fields)


def _value_opts(spec: OptSpec) -> frozenset:
    return spec.value | spec.source | spec.local | spec.version | spec.pkg | spec.extra | spec.reqfile | spec.script


_NO_OPTS = _opts()

# Options that consume TWO following tokens (nix `--option name value`).
_TWO_VALUE_OPTS = frozenset({"--option", "--arg", "--argstr", "--override-input", "--override-flake"})

# --- npm family -----------------------------------------------------------
_NPM_OPTS = _opts(
    value="--tag --prefix -C --omit --include --workspace -w --cache --install-strategy "
          "--loglevel --save-prefix --before --cpu --os --libc "
          "--location --script-shell --otp --scope --auth-type --fetch-retries --fetch-timeout "
          "--maxsockets --lockfile-version --init-module --node-options",
    boolean="-S --save --no-save -D --save-dev -O --save-optional --save-peer -P --save-prod "
            "-E --save-exact -B --save-bundle -g --global --global-style --legacy-bundling "
            "--legacy-peer-deps --strict-peer-deps --prefer-offline --prefer-online --offline "
            "--package-lock-only --package-lock --foreground-scripts --ignore-scripts --audit --fund "
            "--dry-run -f --force --bin-links --install-links --include-workspace-root --workspaces "
            "-ws --production --silent -s -q --quiet --verbose -d -dd -ddd --json --progress "
            "--color --strict-ssl --update-notifier --long -l --parseable --engine-strict --yes -y",
    source="--registry --userconfig --globalconfig",
)
_NPX_OPTS = _opts(
    value="--cache --prefix -w --workspace --loglevel",
    boolean="-y --yes --no --no-install --ignore-existing -q --quiet --silent --workspaces -ws "
            "--include-workspace-root",
    source="--registry --userconfig --globalconfig", pkg="-p --package", script="-c --call",
)
# npm init / create, yarn|pnpm|bun create: `<initializer>` runs create-<initializer>.
_CREATE_OPTS = _opts(
    value="-w --workspace --scope --init-author-name --init-author-email --init-author-url "
          "--init-license --init-module --init-version --cwd",
    boolean="-y --yes -f --force --workspaces -ws --include-workspace-root --private -q --quiet "
            "--silent --offline --prefer-offline",
    source="--registry --userconfig --globalconfig",
)
_PNPM_GLOBAL = _opts(
    value="-C --dir -F --filter --filter-prod --reporter --loglevel --workspace-concurrency "
          "--store-dir --virtual-store-dir --test-pattern --changed-files-ignore-pattern "
          "--config-dir --modules-dir --lockfile-dir --node-linker",
    boolean="-w --workspace-root -r --recursive --silent -s --stream --parallel --aggregate-output "
            "--color --use-stderr --fail-if-no-match --include-workspace-root --bail --sort --reverse",
    source="--registry", pkg="--package",
)
_PNPM_ADD = _opts(
    value="--allow-build --save-catalog-name",
    boolean="-D --save-dev -P --save-prod -O --save-optional -E --save-exact --save-peer -g --global "
            "--workspace --ignore-scripts --offline --prefer-offline --frozen-lockfile --lockfile-only "
            "--fix-lockfile --force --prod --dev --shamefully-hoist --ignore-workspace-root-check "
            "--save-catalog --resolution-only --ignore-pnpmfile --strict-peer-dependencies --latest -L "
            "--interactive -i --depth",
    base=_PNPM_GLOBAL,
)
_PNPM_DLX = _opts(boolean="-c --shell-mode --silent -s", value="--allow-build", pkg="--package",
                  source="--registry")
_YARN_GLOBAL = _opts(
    value="--cwd --include --exclude --from -j --jobs",
    boolean="--silent --verbose --offline --json --non-interactive -A --all -R --recursive -p "
            "--parallel -i --interlaced -v -t --topological --topological-dev -W --worktree",
)
_YARN_ADD = _opts(
    value="--mode --network-timeout --modules-folder --cache-folder --network-concurrency",
    boolean="-D --dev -P --peer -O --optional -E --exact -T --tilde -W --ignore-workspace-root-check "
            "--prefer-dev -i --interactive --cached --audit --ignore-scripts --frozen-lockfile "
            "--immutable --immutable-cache --pure-lockfile --production --ignore-engines "
            "--ignore-optional --check-files --force --inline-builds --refresh-lockfile --check-cache "
            "-R --recursive",
    source="--registry", base=_YARN_GLOBAL,
)
_YARN_DLX = _opts(boolean="-q --quiet", pkg="-p --package")
_BUN_ADD = _opts(
    value="--cwd -c --config --backend --cache-dir --cpu --os --concurrent-scripts "
          "--network-concurrency --omit --linker",
    boolean="-d -D --dev --optional --peer -E --exact -g --global -p --production --frozen-lockfile "
            "--dry-run -f --force --save --ignore-scripts --trust --silent --verbose --yarn "
            "--no-verify --analyze -a --latest",
    source="--registry --ca --cafile",
)
_BUNX_OPTS = _opts(boolean="-b --bun --silent --verbose", pkg="-p --package", value="--cwd",
                   source="--registry")

# --- Python ---------------------------------------------------------------
_PIP_GLOBAL = _opts(
    value="--python --log --proxy --retries --timeout --exists-action --cert --client-cert "
          "--cache-dir --keyring-provider --use-feature --use-deprecated --resume-retries",
    boolean="-q -v -qq -vv -vvv --quiet --verbose --isolated --disable-pip-version-check --no-color "
            "--no-input --require-virtualenv --no-python-version-warning --debug",
    source="--trusted-host",
)
_PIP_INSTALL = _opts(
    value="-t --target --prefix --root --platform --python-version --implementation --abi "
          "--upgrade-strategy --src --report --progress-bar --config-settings -C --global-option "
          "--root-user-action --group --no-binary --only-binary",
    boolean="-U --upgrade --user --no-deps --pre --force-reinstall -I --ignore-installed "
            "--no-build-isolation --require-hashes --dry-run --no-index --no-warn-script-location "
            "--break-system-packages --compile --prefer-binary --use-pep517 --no-clean "
            "--no-warn-conflicts --ignore-requires-python --check-build-dependencies",
    source="-i --index-url --extra-index-url -f --find-links",
    local="-e --editable", reqfile="-r --requirement -c --constraint",
    base=_PIP_GLOBAL,
)
_PIP_DOWNLOAD = _opts(value="-d --dest -w --wheel-dir", base=_PIP_INSTALL)
_UV_GLOBAL = _opts(
    value="--directory --project --cache-dir --color --python-preference",
    boolean="-q -v -qq -vv --quiet --verbose --offline --no-cache -n --native-tls --no-progress "
            "--preview --no-config --no-python-downloads",
    source="--allow-insecure-host --config-file",
)
_UV_RESOLVER = _opts(
    value="-p --python --index-strategy --resolution --prerelease --exclude-newer --python-platform "
          "--link-mode --config-setting --reinstall-package --upgrade-package -P --no-build-package "
          "--no-binary-package --refresh-package --keyring-provider --python-version",
    boolean="--system --reinstall --compile-bytecode --strict --refresh --no-build --no-sources "
            "-U --upgrade --all-extras --no-build-isolation --force --isolated --editable --lfs",
    source="--index --default-index --index-url --extra-index-url -f --find-links",
    base=_UV_GLOBAL,
)
_UV_PIP = _opts(value="-t --target --prefix --only-binary --no-binary --group -C",
                boolean="--user --no-deps --pre --break-system-packages --dry-run --no-index "
                        "--require-hashes",
                local="-e --editable", reqfile="-r --requirement -c --constraint -b --build-constraint",
                base=_UV_RESOLVER)
_UV_ADD = _opts(value="--optional --group --bounds --rev --tag --branch --extra --package --script "
                      "--marker -m",
                boolean="--dev --no-editable --raw --raw-sources --frozen --locked --no-sync "
                        "--workspace --no-workspace --active --no-install-project",
                reqfile="-r --requirements -c --constraints", base=_UV_RESOLVER)
_UV_TOOL = _opts(value="--with-executables-from --env-file",
                 boolean="-e",
                 pkg="--from", extra="--with --with-editable",
                 reqfile="--with-requirements -c --constraints --overrides --build-constraints",
                 base=_UV_RESOLVER)
_UV_RUN = _opts(value="--extra --group --package --env-file --only-group --no-group",
                boolean="--dev --no-dev --all-packages --no-project --no-sync --frozen --locked "
                        "--script -s --module -m --gui-script --all-groups --no-default-groups "
                        "--exact --active --no-editable",
                extra="--with --with-editable", reqfile="--with-requirements",
                base=_UV_RESOLVER)
_PIPX_INSTALL = _opts(value="--python --suffix --preinstall",
                      boolean="--force -f --include-deps --system-site-packages -e --editable "
                              "--global -q -v --quiet --verbose",
                      source="--index-url -i --pip-args")
_PIPX_RUN = _opts(value="--python", boolean="--no-cache --pypackages -q -v --quiet --verbose",
                  pkg="--spec", source="--index-url -i --pip-args")
_POETRY_GLOBAL = _opts(value="-C --directory -P --project",
                       boolean="-q -v -vv -vvv --quiet --verbose -n --no-interaction --no-plugins "
                               "--no-cache --ansi --no-ansi")
_POETRY_ADD = _opts(value="-G --group -E --extras --python --platform --markers",
                    boolean="-D --dev -e --editable --optional --allow-prereleases --dry-run --lock",
                    source="--source", base=_POETRY_GLOBAL)
_PDM_ADD = _opts(value="-G --group -p --project -L --lockfile -k --skip --venv",
                 boolean="-d --dev --no-sync --no-self --no-editable --no-isolation --dry-run "
                         "--prerelease --stable -u --unconstrained --save-compatible --save-wildcard "
                         "--save-exact --save-minimum --update-reuse --update-eager --update-all "
                         "--frozen-lockfile -q -v --no-lock --fail-fast",
                 local="-e --editable")
_PIPENV_INSTALL = _opts(value="--python --categories",
                       boolean="-d --dev --system --deploy --ignore-pipfile --skip-lock --pre "
                               "--keep-outdated --selective-upgrade --site-packages --clear -v "
                               "--verbose -q --quiet",
                       source="-i --index --extra-index-url --pypi-mirror", local="-e --editable",
                       reqfile="-r --requirements")

# --- Rust -----------------------------------------------------------------
_CARGO_GLOBAL = _opts(value="-Z --config -C --color --explain",
                      boolean="-q -v -vv --quiet --verbose --frozen --locked --offline")
_CARGO_ADD = _opts(
    value="-F --features --rename --target -p --package --manifest-path --branch --tag --rev --base "
          "--lockfile-path",
    boolean="--dev --build --optional --no-optional --no-default-features --default-features "
            "--dry-run --public --no-public",
    source="--registry --git", local="--path", base=_CARGO_GLOBAL,
)
_CARGO_INSTALL = _opts(
    value="-F --features --target --target-dir --profile --bin --example -j --jobs --root --branch "
          "--tag --rev",
    boolean="-f --force --no-default-features --all-features --debug --list --no-track --bins "
            "--examples --keep-going --timings",
    source="--index --registry --git", local="--path", version="--version --vers",
    base=_CARGO_GLOBAL,
)
_CARGO_BINSTALL = _opts(
    value="--targets --root --install-path --bin-dir --pkg-fmt --strategies --disable-strategies "
          "--min-tls-version --github-token --rate-limit --log-level",
    boolean="-y --no-confirm --force --locked --no-symlinks --dry-run --no-cleanup "
            "--no-discover-github-token --only-signed --skip-signatures --disable-telemetry -q -v "
            "--quiet --verbose --no-track --continue-on-failure",
    source="--git --registry --index --manifest-path --pkg-url", version="--version",
)

# --- Go -------------------------------------------------------------------
_GO_BUILD = _opts(
    value="-modfile -tags -o -gcflags -ldflags -asmflags -pkgdir -mod -overlay -p -toolexec "
          "-exec -buildmode -compiler -installsuffix -covermode -coverpkg -pgo",
    boolean="-d -t -u -v -x -a -n -race -msan -asan -cover -work -trimpath -linkshared "
            "-modcacherw -tool -insecure -buildvcs",
)

# --- Ruby / PHP / .NET / Dart / Deno --------------------------------------
_GEM_INSTALL = _opts(
    value="-i --install-dir -n --bindir --platform -P --trust-policy -g --file --document "
          "--build-root -p --http-proxy --without --target-rbconfig --config-file",
    boolean="-f --force --no-document -N --user-install --no-user-install --conservative "
            "--minimal-deps -E --env-shebang --development --development-all --prerelease --pre "
            "-q -V --verbose --quiet --ignore-dependencies --no-ri --no-rdoc --explain -l --local "
            "-r --remote -b --both --clear-sources -u --update-sources --lock --suggestions "
            "--default --post-install-message -w --wrappers --vendor --norc --backtrace --debug",
    source="-s --source", version="-v --version",
)
_BUNDLE_ADD = _opts(value="-g --group -r --require --branch --ref --glob",
                    boolean="--skip-install --strict --optimistic --quiet",
                    source="-s --source --git --github", local="--path", version="-v --version")
_COMPOSER_GLOBAL = _opts(value="-d --working-dir",
                         boolean="-q -v -vv -vvv --quiet --verbose -n --no-interaction --no-plugins "
                                 "--no-scripts --no-cache --ansi --no-ansi --profile")
_COMPOSER_REQUIRE = _opts(
    value="--ignore-platform-req --apcu-autoloader-prefix --audit-format --prefer-install",
    boolean="--dev --no-update --no-install --no-progress -W --update-with-all-dependencies -w "
            "--update-with-dependencies --with-all-dependencies --with-dependencies --prefer-source "
            "--prefer-dist --prefer-stable --prefer-lowest --sort-packages -o --optimize-autoloader "
            "-a --classmap-authoritative --apcu-autoloader --ignore-platform-reqs --dry-run --fixed "
            "--no-audit -m --minimal-changes --no-security-blocking",
    base=_COMPOSER_GLOBAL,
)
_DOTNET_ADD_PACKAGE = _opts(value="-f --framework --package-directory",
                            boolean="-n --no-restore --interactive", prerelease="--prerelease",
                            source="-s --source", version="-v --version")
_DOTNET_PACKAGE_ADD = _opts(value="--project", base=_DOTNET_ADD_PACKAGE)
_DOTNET_TOOL_INSTALL = _opts(value="--tool-path --framework -a --arch --verbosity",
                             boolean="-y --yes --allow-roll-forward -g --global --local --create-manifest-if-needed "
                                     "--allow-downgrade --disable-parallel --ignore-failed-sources "
                                     "--no-cache --interactive", prerelease="--prerelease",
                             source="--add-source --configfile", version="--version")
_NUGET_INSTALL = _opts(value="-OutputDirectory -Framework -Verbosity -DependencyVersion",
                       boolean="-NoCache -NonInteractive -ExcludeVersion -DirectDownload",
                       prerelease="-Prerelease -prerelease",
                       source="-Source -ConfigFile -FallbackSource", version="-Version")
_PUB_ADD = _opts(value="--directory -C",
                 boolean="--dev -d --dry-run -n --offline --precompile --example",
                 source="--git-url --git-ref --git-path --hosted-url --sdk", local="--path")
_PUB_ACTIVATE = _opts(value="--executable -x --features", boolean="--overwrite --no-executables",
                      source="--source -s --hosted-url --git-path --git-ref")
_COMPOSER_CREATE = _opts(value="-s --stability --add-repository",
                         boolean="--prefer-source --prefer-dist --prefer-install --dev --no-dev "
                                 "--no-plugins --no-scripts --no-progress --no-secure-http --keep-vcs "
                                 "--remove-vcs --no-install --no-audit --ignore-platform-reqs --ask",
                         source="--repository --repository-url", base=_COMPOSER_GLOBAL)


class Verb(NamedTuple):
    tokens: tuple        # subcommand tokens following the manager's global options
    ecosystem: str       # OSV ecosystem, or "" when the guard cannot verify it
    manager: str         # safety-flag label (see SAFETY_FLAGS)
    opts: OptSpec
    kind: str = "packages"
    # packages:     every operand is a package specifier
    # runner:       the first operand is the package to run (npx, uvx, dlx ...)
    # with:         only `extra` option values are packages (uv run --with)
    # gorun:        `go run mod@version` downloads; a local package does not
    # create:       `npm init <x>` / `yarn create <x>`: runs the create-<x> package
    # inject:       `pipx inject <venv> <pkgs>`: packages after the first operand
    # unverifiable: packages from a registry the guard cannot check -> ask


class Manager(NamedTuple):
    globals: OptSpec     # options accepted between the executable and the verb
    prefixes: dict       # subcommand -> positional args it takes before the real verb
    verbs: tuple


def _verbs(names: str, ecosystem: str, manager: str, opts: OptSpec, kind: str = "packages") -> list:
    """One Verb per alias; multi-token verbs are joined with '+' in `names`."""
    return [Verb(tuple(n.split("+")), ecosystem, manager, opts, kind) for n in names.split()]


# npm's own alias table (lib/utils/cmd-list.js), so misspelled and short
# aliases that npm accepts are recognised too.
_NPM_INSTALL_ALIASES = ("install i in ins inst insta instal isnt isnta isntal isntall add "
                        "install-test it install-ci-test cit sit ci clean-install ic install-clean "
                        "isntall-clean update up upgrade udpate")

MANAGERS: dict = {
    "npm": Manager(_NPM_OPTS, {}, tuple(
        _verbs(_NPM_INSTALL_ALIASES, "npm", "npm", _NPM_OPTS)
        + _verbs("exec x", "npm", "npx", _NPX_OPTS, "runner")
        + _verbs("init create innit", "npm", "npx", _CREATE_OPTS, "create"))),
    "npx": Manager(_NO_OPTS, {}, (Verb((), "npm", "npx", _NPX_OPTS, "runner"),)),
    "pnpm": Manager(_PNPM_GLOBAL, {}, tuple(
        _verbs("add install i update up upgrade", "npm", "pnpm", _PNPM_ADD)
        + _verbs("dlx", "npm", "npx", _PNPM_DLX, "runner")
        + _verbs("create", "npm", "npx", _CREATE_OPTS, "create"))),
    "pnpx": Manager(_NO_OPTS, {}, (Verb((), "npm", "npx", _PNPM_DLX, "runner"),)),
    "yarn": Manager(_YARN_GLOBAL, {"workspace": 1, "global": 0, "workspaces": 0, "foreach": 0}, tuple(
        _verbs("add install up upgrade", "npm", "yarn", _YARN_ADD)
        + _verbs("dlx", "npm", "npx", _YARN_DLX, "runner")
        + _verbs("create", "npm", "npx", _CREATE_OPTS, "create"))),
    "bun": Manager(_opts(value="--cwd -c --config", boolean="--silent --verbose --bun"), {}, tuple(
        _verbs("add a install i update", "npm", "bun", _BUN_ADD)
        + _verbs("x", "npm", "npx", _BUNX_OPTS, "runner")
        + _verbs("create c", "npm", "npx", _CREATE_OPTS, "create"))),
    "bunx": Manager(_NO_OPTS, {}, (Verb((), "npm", "npx", _BUNX_OPTS, "runner"),)),
    "pip": Manager(_PIP_GLOBAL, {}, tuple(
        _verbs("install", "PyPI", "pip", _PIP_INSTALL)
        + _verbs("download wheel", "PyPI", "pip", _PIP_DOWNLOAD))),
    "uv": Manager(_UV_GLOBAL, {}, tuple(
        _verbs("pip+install", "PyPI", "uv-pip", _UV_PIP)
        + _verbs("add", "PyPI", "uv-add", _UV_ADD)
        + _verbs("tool+install", "PyPI", "uv-tool", _UV_TOOL)
        + _verbs("tool+run", "PyPI", "uv-tool", _UV_TOOL, "runner")
        + _verbs("run", "PyPI", "uv-run", _UV_RUN, "with"))),
    "uvx": Manager(_NO_OPTS, {}, (Verb((), "PyPI", "uv-tool", _UV_TOOL, "runner"),)),
    "pipx": Manager(_NO_OPTS, {}, tuple(
        _verbs("install", "PyPI", "pipx", _PIPX_INSTALL)
        + _verbs("run", "PyPI", "pipx", _PIPX_RUN, "runner")
        + _verbs("inject", "PyPI", "pipx", _PIPX_INSTALL, "inject"))),
    "poetry": Manager(_POETRY_GLOBAL, {"self": 0}, tuple(_verbs("add", "PyPI", "poetry", _POETRY_ADD))),
    "pipenv": Manager(_NO_OPTS, {}, tuple(_verbs("install", "PyPI", "pipenv", _PIPENV_INSTALL))),
    "pdm": Manager(_opts(value="-c --config", boolean="-v -q --verbose --quiet"), {"self": 0}, tuple(
        _verbs("add", "PyPI", "pdm", _PDM_ADD))),
    "cargo": Manager(_CARGO_GLOBAL, {}, tuple(
        _verbs("add", "crates.io", "cargo-add", _CARGO_ADD)
        + _verbs("install", "crates.io", "cargo", _CARGO_INSTALL)
        + _verbs("binstall", "crates.io", "cargo-binstall", _CARGO_BINSTALL))),
    "cargo-binstall": Manager(_NO_OPTS, {}, (Verb((), "crates.io", "cargo-binstall", _CARGO_BINSTALL),)),
    "go": Manager(_opts(value="-C"), {}, tuple(
        _verbs("get install", "Go", "go", _GO_BUILD)
        + _verbs("run", "Go", "go", _GO_BUILD, "gorun"))),
    "gem": Manager(_NO_OPTS, {}, tuple(_verbs("install i", "RubyGems", "gem", _GEM_INSTALL))),
    "bundle": Manager(_NO_OPTS, {}, tuple(_verbs("add", "RubyGems", "bundle", _BUNDLE_ADD))),
    "composer": Manager(_COMPOSER_GLOBAL, {"global": 0}, tuple(
        _verbs("require req", "Packagist", "composer", _COMPOSER_REQUIRE)
        + _verbs("create-project", "Packagist", "composer", _COMPOSER_CREATE, "runner"))),
    "dotnet": Manager(_NO_OPTS, {}, tuple(
        _verbs("add+package", "NuGet", "dotnet", _DOTNET_ADD_PACKAGE)
        + _verbs("package+add", "NuGet", "dotnet", _DOTNET_PACKAGE_ADD)
        + _verbs("tool+install tool+update tool+exec tool+run new+install new+-i new+--install", "NuGet",
                 "dotnet", _DOTNET_TOOL_INSTALL)
        + _verbs("dnx", "NuGet", "dotnet", _DOTNET_TOOL_INSTALL, "runner"))),
    "dnx": Manager(_NO_OPTS, {}, (Verb((), "NuGet", "dotnet", _DOTNET_TOOL_INSTALL, "runner"),)),
    "nuget": Manager(_NO_OPTS, {}, tuple(_verbs("install", "NuGet", "nuget", _NUGET_INSTALL))),
    "dart": Manager(_NO_OPTS, {}, tuple(
        _verbs("pub+add", "Pub", "pub", _PUB_ADD)
        + _verbs("pub+global+activate", "Pub", "pub", _PUB_ACTIVATE))),
    # Registries the guard has no OSV/age resolver for: an install through
    # them cannot be verified, so it is surfaced to the user (ask).
    "mix": Manager(_NO_OPTS, {}, tuple(_verbs("archive.install escript.install", "", "mix", _NO_OPTS,
                                               "unverifiable"))),
    "cabal": Manager(_NO_OPTS, {}, tuple(_verbs("install", "", "cabal", _NO_OPTS, "unverifiable"))),
    "stack": Manager(_NO_OPTS, {}, tuple(_verbs("install", "", "stack", _NO_OPTS, "unverifiable"))),
    "luarocks": Manager(_NO_OPTS, {}, tuple(_verbs("install", "", "luarocks", _NO_OPTS, "unverifiable"))),
    "lux": Manager(_NO_OPTS, {}, tuple(_verbs("add install", "", "lux", _NO_OPTS, "unverifiable"))),
    "vcpkg": Manager(_NO_OPTS, {}, tuple(_verbs("install add+port", "", "vcpkg", _NO_OPTS, "unverifiable"))),
    "conan": Manager(_NO_OPTS, {}, tuple(_verbs("install", "", "conan",
                                                _opts(extra="--requires --tool-requires",
                                                      source="-r --remote"), "unverifiable"))),
    "ansible-galaxy": Manager(_NO_OPTS, {}, tuple(_verbs(
        "install collection+install role+install", "", "ansible-galaxy",
        _opts(reqfile="-r --role-file --requirements-file", value="-p --roles-path -s --server",
              boolean="-f --force --force-with-deps -n --no-deps"), "unverifiable"))),
    "cpanm": Manager(_NO_OPTS, {}, (Verb((), "", "cpanm", _NO_OPTS, "unverifiable"),)),
    "cpan": Manager(_NO_OPTS, {}, (Verb((), "", "cpan", _NO_OPTS, "unverifiable"),)),
    "conda": Manager(_NO_OPTS, {}, tuple(_verbs("install create", "", "conda", _NO_OPTS, "unverifiable"))),
    "mamba": Manager(_NO_OPTS, {}, tuple(_verbs("install create", "", "conda", _NO_OPTS, "unverifiable"))),
    "micromamba": Manager(_NO_OPTS, {}, tuple(_verbs("install create", "", "conda", _NO_OPTS,
                                                      "unverifiable"))),
    "swift": Manager(_NO_OPTS, {}, tuple(_verbs("package+add-dependency", "", "spm", _NO_OPTS,
                                                 "unverifiable"))),
    "zig": Manager(_NO_OPTS, {}, tuple(_verbs("fetch", "", "zig", _NO_OPTS, "unverifiable"))),
}
MANAGERS["pip3"] = MANAGERS["pip"]
MANAGERS["bundler"] = MANAGERS["bundle"]
MANAGERS["flutter"] = MANAGERS["dart"]

# Safety flags inserted after the install verb via updatedInput, keyed by the
# Verb.manager label. Only labels listed here are rewritten.
SAFETY_FLAGS: dict = {
    "npm": "--ignore-scripts",
    # bun runs lifecycle scripts for its built-in list of ~370 default-trusted
    # packages (esbuild, sharp, puppeteer, ...), so scripts must be disabled.
    "bun": "--ignore-scripts",
    "pip": "--only-binary :all:",
    "uv-pip": "--only-binary :all:",
    "cargo": "--locked",  # cargo install only: `cargo add --locked` fails by design
}

# Package managers declared by qsdev's ecosystem modules
# (PackageManagerInfo.Name) mapped to the executables this guard models for
# them. package_guard_test.go fails when a module declares a manager missing
# here. An empty tuple means the manager has no command that adds a package:
# dependencies change only by editing the committed manifest.
CATALOG_MANAGER_EXECUTABLES: dict = {
    "npm": ("npm", "npx"), "pnpm": ("pnpm", "pnpx"), "yarn": ("yarn",), "bun": ("bun", "bunx"),
    "pip": ("pip", "python"), "uv": ("uv", "uvx"), "poetry": ("poetry",),
    "cargo": ("cargo", "cargo-binstall"), "go modules": ("go",), "bundler": ("bundle", "gem"),
    "composer": ("composer",), "nuget": ("dotnet", "dnx", "nuget"), "pub": ("dart", "flutter"),
    "mix": ("mix",), "cabal": ("cabal",), "stack": ("stack",), "luarocks": ("luarocks",),
    "lux": ("lux",), "conan": ("conan",), "vcpkg": ("vcpkg",), "ansible-galaxy": ("ansible-galaxy",),
    "helm": ("helm",), "maven": ("mvn",), "carton": ("cpanm", "cpan"), "psgallery": ("pwsh", "powershell"),
    "renv": ("rscript", "r"), "spm": ("swift",), "zig-build": ("zig",), "tools-deps": ("clojure", "clj"),
    "nix-flake": ("nix", "nix-env", "nix-shell"),
    "gradle": (), "sbt": (), "bzlmod": (), "leiningen": (), "terraform-registry": (),
}

# `npm`/`pnpm`/... followed by one of these is an install; used to recognise an
# install whose command name or verb the shell computes.
_INSTALL_VERB_WORDS: frozenset = frozenset(
    v.tokens[0] for m in MANAGERS.values() for v in m.verbs if v.tokens)


# ---------------------------------------------------------------------------
# Command wrappers and shells
# ---------------------------------------------------------------------------

class WrapperSpec(NamedTuple):
    value: frozenset         # this wrapper's options that take a value (short ones may be clustered)
    positional: str = ""     # "duration" / "one": a positional precedes the command
    runtime: bool = False    # appends arguments read at run time (xargs, parallel)
    script: frozenset = frozenset()   # options whose value is a shell script (flock -c)
    no_exec: frozenset = frozenset()  # options that make it NOT run the command (command -v)
    joins: bool = False      # runs its remaining arguments joined as a `sh -c` script (watch)


def _wrapper(value: str = "", positional: str = "", runtime: bool = False,
             script: str = "", no_exec: str = "", joins: bool = False) -> WrapperSpec:
    return WrapperSpec(frozenset(value.split()), positional, runtime,
                       frozenset(script.split()), frozenset(no_exec.split()), joins)


# Exec wrappers that run the command that follows them, each with its OWN
# option grammar: a flag that takes a value for one wrapper (`sudo -p PROMPT`)
# is a boolean for another (`time -p`, `xargs -r`, `command -p`), so a shared
# table would let the executable be swallowed as a flag value.
WRAPPERS: dict = {
    "sudo": _wrapper("-u --user -g --group -C --close-from -p --prompt -r --role -t --type -U "
                     "--other-user -D --chdir -R --chroot -T --command-timeout --host"),
    "doas": _wrapper("-u -C"),
    "env": _wrapper("-u --unset -C --chdir -P"),
    "command": _wrapper(no_exec="-v -V"),
    "builtin": _wrapper(),
    "exec": _wrapper("-a"),
    "time": _wrapper("-f --format -o --output"),
    "nice": _wrapper("-n --adjustment"),
    "nohup": _wrapper(),
    "stdbuf": _wrapper("-i --input -o --output -e --error"),
    "setsid": _wrapper(),
    "ionice": _wrapper("-c --class -n --classdata", no_exec="-p --pid -P --pgid -u --uid"),
    "timeout": _wrapper("-k --kill-after -s --signal", positional="duration"),
    "strace": _wrapper("-e -o -p -s -u -E -a -b -I -O -S -P -X -U"),
    "ltrace": _wrapper("-e -o -p -s -u -n -a -A -D -F -l -x"),
    "catchsegv": _wrapper(),
    "proot": _wrapper("-r --rootfs -b --bind -m --mount -w --cwd -q --qemu -k --kernel-release"),
    "firejail": _wrapper(),
    "flock": _wrapper("-w --timeout -E --conflict-exit-code", positional="one", script="-c --command"),
    "unshare": _wrapper("-S --setuid -G --setgid -R --root -w --wd --map-user --map-group "
                        "--propagation --setgroups"),
    "chrt": _wrapper("-T --sched-runtime -P --sched-period -D --sched-deadline", positional="one",
                     no_exec="-p --pid"),
    "taskset": _wrapper(positional="one", no_exec="-p --pid"),
    "xargs": _wrapper("-a --arg-file -d --delimiter -E -I -L --max-lines -n --max-args -P "
                      "--max-procs -s --max-chars --process-slot-var", runtime=True),
    "parallel": _wrapper("-j --jobs -S --sshlogin --joblog --results -a --arg-file --delay "
                         "--timeout --colsep -N --max-args", runtime=True),
    "setpriv": _wrapper("--reuid --regid --ruid --rgid --euid --egid --groups --inh-caps "
                        "--ambient-caps --bounding-set --securebits --pdeathsig --selinux-label "
                        "--apparmor-profile --landlock-access --landlock-rule"),
    "nsenter": _wrapper("-t --target -S --setuid -G --setgid -w --wd -r --root"),
    "systemd-run": _wrapper("-p --property -u --unit --description --slice -E --setenv --uid --gid "
                            "--working-directory -M --machine -H --host"),
    "busybox": _wrapper(),
    # `mono nuget.exe install ...`: Mono's options are `--opt` or `--opt=value`, except --config.
    "mono": _wrapper("--config"),
    "watch": _wrapper("-n --interval -q --equexit", joins=True),
    "script": _wrapper("-E --echo -I --log-in -O --log-out -B --log-io -T --log-timing -m "
                       "--logging-format", script="-c --command"),
}

# Shell reserved words and grouping tokens that can precede a command in the
# same segment (`if x; then npm install y; fi`, `! npm install y`, `{ ...; }`).
_RESERVED_PREFIXES = frozenset({"!", "{", "if", "then", "else", "elif", "do", "while", "until",
                                "coproc"})

# Builtins that never execute their arguments: an install-looking word there
# is data (`echo Run npm install next`), so the embedded-install fallback skips
# them.
_NON_EXEC_COMMANDS = frozenset({"echo", "printf", ":", "true", "false", "test", "[", "cat",
                                "type", "which", "whereis", "hash", "help", "man"})

# Shells whose `-c "<script>"` argument is itself a command line.
_SHELLS: set = {"sh", "bash", "zsh", "dash", "ash", "ksh", "mksh", "yash", "posh", "fish", "csh", "tcsh",
                 "nu", "elvish", "xonsh"}

# Privilege launchers whose `-c "<script>"` (or `--command`) argument is a shell
# script, exactly like a shell's -c (e.g. `su -c "npm install evil"`). Unlike a
# shell, the target user may appear as a positional BEFORE -c, so every token is
# scanned for the flag.
_PRIV_C_RUNNERS: set = {"su", "runuser"}
_PRIV_VALUE_FLAGS: set = {
    "-u", "--user", "-g", "--group", "-G", "--supp-group",
    "-s", "--shell", "-w", "--whitelist-environment",
}

# Commands that may execute text they read (stdin, a file or their arguments) as
# a shell script. A heredoc body, or a substitution's output, reaching one of
# these in the same command is scanned as a script rather than treated as data.
_SCRIPT_READERS: set = _SHELLS | _PRIV_C_RUNNERS | {"eval", "source", "."}

# Bound on recursive shell-script scanning (shell -c, eval, su -c, command and
# process substitutions). Exceeding it FAILS CLOSED (a suspicious marker is
# emitted so main() blocks for validation) rather than silently dropping the
# deeper script -- a dropped script is a fail-open bypass.
_MAX_SHELL_RECURSION = 6

# Manager label for a segment that could not be safely analyzed (unparseable, or
# nested past the recursion cap). main() denies these to fail closed.
_SUSPICIOUS_MANAGER = "__suspicious__"
# Manager label for an install the guard recognised but cannot verify
# (dynamic names, unknown options, non-registry sources, unverifiable
# registries). main() asks the user about these.
_UNVERIFIED_MANAGER = "__unverified__"
# Manager label for a command that installs nothing itself but runs nested
# scripts (nix develop -c, nix-shell --run, pwsh -Command): only those matter.
_SCRIPTS_ONLY = "__scripts__"

# A leading VAR=value environment assignment (e.g. `FOO=bar npm install ...`).
_ENV_ASSIGN_RE = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*=")

# Environment variables that point a package manager at another registry or
# index, or weaken checksum verification: the name checked on the public
# registry would not be what gets installed.
_SOURCE_ENV_RE = re.compile(
    r"^(npm_config_\w*registry|yarn_npm_registry_server|yarn_registry|bun_config_registry|"
    r"pip_(?:extra_)?index_url|pip_find_links|pip_trusted_host|uv_(?:extra_)?index(?:_url)?|"
    r"uv_default_index|uv_find_links|uv_insecure_host|cargo_registries_\w+|cargo_registry_\w+|"
    r"goproxy|gonosumdb|gonosumcheck|goinsecure|gosumdb|goprivate|goflags|"
    r"poetry_repositories_\w+|pdm_pypi_url|nuget_\w*source\w*|npm_config_(?:user|global)config|"
    r"pip_config_file|uv_config_file|bundle_mirror__\w+)$",
    re.IGNORECASE,
)

_URLISH_RE = re.compile(
    r"^(?:[A-Za-z][A-Za-z0-9+.-]*://|git\+|git@|(?:git|github|gitlab|bitbucket|gist|ssh|svn|hg|bzr|"
    r"http|https|oci):)",
    re.IGNORECASE,
)
# Archive operands every manager treats as a local file rather than a name.
_ARCHIVE_SUFFIXES = (".tgz", ".tar.gz", ".tar", ".tar.bz2", ".zip", ".whl", ".gem", ".crate", ".nupkg")
# Manifest/recipe files named by the unverifiable managers (conan, ansible ...).
_MANIFEST_SUFFIXES = (".txt", ".toml", ".yml", ".yaml", ".json", ".lock", ".py", ".cfg")
_PYTHON_OPT_WITH_VALUE = set("WXQc")


class Unwrapped(NamedTuple):
    argv: list
    dyn: list          # per-token: the shell computes this word
    cmd_dyn: bool      # argv[0] is computed (including [..] globs)
    off: int           # index of argv[0] in the segment's words (-1: argv was re-split)
    env: list          # VAR=value assignments applying to the command
    runtime: bool      # arguments are appended at run time (xargs/parallel)
    script: Optional[str]  # a wrapper's own shell-script option value (flock -c)


class Install(NamedTuple):
    ecosystem: str
    manager: str
    packages: list     # registry package specifiers to validate
    issues: list       # reasons the install needs the user's confirmation
    verb_end: int      # argv index just past the install verb (safety flags go here)
    scripts: list      # nested shell scripts to scan (npx -c, nix develop -c ...)


class Detection(NamedTuple):
    ecosystem: str
    manager: str
    segment: str
    packages: list
    issues: tuple = ()
    flag: str = ""        # safety flag that still has to be added
    insert_at: int = -1   # offset in the original command for `flag` (-1: cannot rewrite)
    denials: tuple = ()   # reasons the install must be blocked outright


def _exe_name(token: str) -> str:
    """Normalise an executable token for lookup: basename, lower-cased (macOS
    file systems are case-insensitive, so `NPM install` runs npm), with
    version suffixes of versioned binaries removed (pip3.12 -> pip)."""
    name = os.path.basename(token.replace("\\", "/")).lower()
    for suffix in (".exe", ".cmd", ".bat", ".ps1", ".phar"):
        if name.endswith(suffix):
            name = name[:-len(suffix)]
    return re.sub(r"^(pip|python|pipx)[0-9][0-9.]*$", r"\1", name)


def _is_urlish(value: str) -> bool:
    return bool(_URLISH_RE.match(value)) or "://" in value


def _is_local(value: str, manifests: bool = False) -> bool:
    suffixes = _ARCHIVE_SUFFIXES + (_MANIFEST_SUFFIXES if manifests else ())
    return value in (".", "..") or value.startswith(("./", "../", "/", "~")) or (
        not _is_urlish(value) and value.lower().endswith(suffixes))


def _short_cluster(flag: str, spec: WrapperSpec, name: str) -> tuple:
    """Split a short-option token for a wrapper into (option, eq, inline value).
    A cluster (`-Eu`, `-iS`, `-qc`, `-n10`) is read letter by letter up to the
    first option that takes a value or stops execution; the rest of the token
    is that option's inline value. Returns ("", "", "") for a cluster of
    booleans. Short options take no `=`: `-S'A=1 npm i'` keeps its `=`."""
    takes_value = spec.value | spec.script | ({"-S"} if name == "env" else set())
    if flag in takes_value or flag in spec.no_exec or len(flag) <= 2:
        return flag, "", ""
    for k in range(1, len(flag)):
        opt = "-" + flag[k]
        if opt in spec.no_exec:
            return opt, "", ""
        if opt in takes_value:
            inline = flag[k + 1:]
            return opt, ("=" if inline else ""), inline
    return "", "", ""


def _unwrap(words: list) -> Unwrapped:
    """Strip leading environment assignments, reserved words and exec wrappers
    (each with its own option grammar), so argv[0] is the real executable."""
    argv = [w.text for w in words]
    dyn = [w.dynamic for w in words]
    brk = [w.bracket for w in words]
    i, n = 0, len(argv)
    env: list = []
    runtime = False
    resplit = False
    script: Optional[str] = None
    while i < n:
        tok = argv[i]
        if _ENV_ASSIGN_RE.match(tok) and not dyn[i]:
            env.append(tok)
            i += 1
            continue
        if tok in _RESERVED_PREFIXES:
            i += 1
            continue
        if tok == "function" and i + 1 < n:
            i += 2  # `function name { ...`: the body follows
            continue
        name = _exe_name(tok)
        if name == "devenv" and argv[i + 1:i + 2] == ["shell"]:
            i += 2
            if argv[i:i + 1] == ["--"]:
                i += 1
            continue
        spec = WRAPPERS.get(name)
        if spec is None or dyn[i]:
            break
        runtime = runtime or spec.runtime
        j = i + 1
        restart = False
        while j < n and argv[j].startswith("-") and argv[j] != "-":
            flag = argv[j]
            j += 1
            if flag == "--":
                break
            if flag.startswith("--"):
                base, eq, inline = flag.partition("=")
            else:
                base, eq, inline = _short_cluster(flag, spec, name)
                if not base:
                    continue  # a cluster of boolean flags (`-Ei`)
            if base in spec.no_exec:
                return Unwrapped([], [], False, 0, env, runtime, None)
            if name == "env" and base in ("--split-string", "-S"):
                # env -S 'npm install x': the string is split into the command.
                if eq:
                    value = inline
                else:
                    value = argv[j] if j < n else ""
                    j += 1
                split = shlex.split(value)  # ValueError propagates: fail closed
                argv = split + argv[j:]
                dyn = [_SUBST in t or "$" in t for t in split] + dyn[j:]
                brk = [False] * len(split) + brk[j:]
                i, n = 0, len(argv)
                resplit = restart = True
                break
            if base in spec.script:
                if not eq:
                    inline = argv[j] if j < n else ""
                    j += 1
                script = inline
                continue
            if not eq and base in spec.value and j < n:
                j += 1
        if restart:
            continue
        if spec.joins:
            script = " ".join(argv[j:])  # `watch 'npm install x'` runs `sh -c`
            j = n
        if spec.positional and j < n and not argv[j].startswith("-"):
            if spec.positional == "one" or re.match(r"^[0-9]", argv[j]):
                j += 1
        if j < n and argv[j] in spec.script:  # `flock FILE -c 'script'`
            script = argv[j + 1] if j + 1 < n else ""
            j += 2
        i = j
    off = -1 if resplit else i
    return Unwrapped(argv[i:], dyn[i:], bool(argv[i:]) and (dyn[i] or brk[i]), off, env, runtime, script)


class ShellCall(NamedTuple):
    script: Optional[str]   # the -c script
    reads_stdin: bool       # the script comes from stdin
    operand: str = ""       # the script file operand (`bash file.sh`, `bash <(curl ...)`)


def _shell_invocation(argv: list) -> ShellCall:
    """Parse a shell's options: `-o`/`-O`/`+o` take a value (`bash -o pipefail
    -c`), `--rcfile`/`--init-file` take a value, `+e`-style flags are options,
    and a cluster containing `c` (`-lc`, `-ce`) makes the first operand after
    the options (and after `--`) the script."""
    has_c = False
    stdin_flag = False
    i, n = 1, len(argv)
    while i < n:
        tok = argv[i]
        if tok in ("--", "-"):
            i += 1
            break
        if tok.startswith("--"):
            if tok in ("--rcfile", "--init-file", "--init-command"):
                i += 2
                continue
            if tok == "--command":
                has_c = True
            elif tok.startswith("--command="):
                return ShellCall(tok.split("=", 1)[1], False)
            i += 1
            continue
        if len(tok) > 1 and tok[0] in "-+":
            letters = tok[1:]
            if tok[0] == "-":
                has_c = has_c or "c" in letters
                stdin_flag = stdin_flag or "s" in letters
            i += 1 + letters.count("o") + letters.count("O")
            continue
        break
    rest = argv[i:]
    if has_c:
        return ShellCall(rest[0] if rest else "", False)
    if stdin_flag or not rest or rest[0] in ("/dev/stdin", "-", "/dev/fd/0"):
        return ShellCall(None, True)
    return ShellCall(None, False, rest[0])


def _c_runner_script(argv: list) -> Optional[str]:
    """su/runuser -c "<script>": the target user may appear as a positional
    before -c, so every token is scanned for -c/--command."""
    for i in range(1, len(argv)):
        tok = argv[i]
        if tok == "--command" or re.match(r"^-[A-Za-z]*c$", tok):
            return argv[i + 1] if i + 1 < len(argv) else ""
        if tok.startswith("--command="):
            return tok.split("=", 1)[1]
    return None


def _skip_option_flags(tokens: list, i: int, value_flags: set) -> int:
    n = len(tokens)
    while i < n and tokens[i].startswith("-"):
        flag = tokens[i]
        i += 1
        if "=" not in flag and flag in value_flags and i < n:
            i += 1
    return i


# ---------------------------------------------------------------------------
# Install parsing
# ---------------------------------------------------------------------------

def _is_boolean_opt(tok: str, spec: OptSpec) -> bool:
    if tok in spec.boolean or (tok.startswith("--no-") and tok not in _value_opts(spec)):
        return True
    # A cluster of known short booleans (`-qU`, `-DE`).
    if not tok.startswith("--") and len(tok) > 2:
        return all(f"-{c}" in spec.boolean for c in tok[1:])
    return False


def _classify_spec(ecosystem: str, spec: str) -> tuple:
    """Classify a package operand as ("registry", spec), ("local", "") or
    ("source", reason). Source means a URL, git, path or other non-registry
    location: the guard cannot check what it will fetch."""
    if ecosystem == "npm":
        if spec.startswith("npm:"):
            spec = spec[4:]
        elif spec.startswith("jsr:"):
            return "source", f"'{spec}' comes from JSR, which the guard cannot verify"
        if _is_urlish(spec):
            return "source", f"'{spec}' installs from a URL or git source"
        if _is_local(spec):
            return "local", ""
        name, ver = _npm_name_version(spec)
        if not spec.startswith("@") and "/" in name:
            return "source", f"'{spec}' is GitHub shorthand (user/repo), not a registry package"
        if spec.startswith("@") and "/" not in name:
            return "source", f"'{spec}' is not a valid package name"
        if ver.startswith("npm:"):
            # Classify the alias target, but keep `alias@npm:target` so
            # parse_spec validates the target AND checks the alias name.
            kind, detail = _classify_spec("npm", ver[4:])
            return kind, (spec if kind == "registry" else detail)
        if ver.startswith(("workspace:", "file:", "link:", "portal:")):
            return "local", ""
        if _is_urlish(ver) or "/" in ver or ver.startswith(("patch:", "exec:")):
            return "source", f"'{spec}' installs from a URL, git or other non-registry source"
        return "registry", spec
    if _is_urlish(spec):
        return "source", f"'{spec}' installs from a URL or VCS source"
    if ecosystem == "PyPI":
        m = re.match(r"^[A-Za-z0-9._-]+\s*(?:\[[^\]]*\])?\s*@\s*(.+)$", spec.split(";", 1)[0].strip())
        if m and (_is_urlish(m.group(1).strip()) or _is_local(m.group(1).strip())):
            return "source", f"'{spec}' is a PEP 508 direct reference, not a registry release"
        if _is_local(re.sub(r"\[[^\]]*\]$", "", spec)):
            return "local", ""  # `.[dev]`, `./pkg[extra]`: the local project with extras
    if ecosystem == "Pub" and ("{" in spec or spec.startswith(("git:", "path:", "hosted:", "sdk:"))):
        return "source", f"'{spec}' is a non-registry pub source"
    if _is_local(spec) or (ecosystem == "Go" and spec.startswith(".")):
        return "local", ""
    return "registry", spec


def _create_package(spec: str) -> str:
    """The package `npm init <spec>` / `yarn create <spec>` runs: foo ->
    create-foo, @scope -> @scope/create, @scope/foo -> @scope/create-foo. URLs,
    paths and user/repo shorthand are returned unchanged (classified later)."""
    if _is_urlish(spec) or _is_local(spec):
        return spec
    if spec.startswith("@"):
        scope, slash, rest = spec[1:].partition("/")
        if not slash:
            scope, at, ver = scope.partition("@")
            return f"@{scope}/create" + (f"@{ver}" if at else "")
        return f"@{scope}/create-{rest}"
    if "/" in _npm_name_version(spec)[0]:
        return spec
    return "create-" + spec


def _scan_operands(args: list, dyn: list, verb: Verb, runtime: bool) -> Optional[tuple]:
    """Parse the operands after an install verb. Returns (packages, issues,
    scripts), or None when the invocation turns out not to install anything
    (`uv run` without --with, `go run ./cmd`)."""
    spec = verb.opts
    value_opts = _value_opts(spec)
    issues: list = []
    scripts: list = []
    versions: list = []
    pkg_values: list = []
    extra_values: list = []
    positionals: list = []
    prerelease = False
    i, n = 0, len(args)
    end_of_opts = False
    while i < n:
        tok = args[i]
        if tok in (";", "+"):  # find -exec terminator
            break
        if not end_of_opts and tok == "--":
            end_of_opts = True
            i += 1
            continue
        is_opt = not end_of_opts and len(tok) > 1 and tok[0] == "-"
        if not is_opt:
            positionals.append((tok, dyn[i] or tok == "{}"))
            i += 1
            if verb.kind in ("runner", "with", "gorun", "create"):
                break
            continue
        base, eq, value = tok.partition("=")
        i += 1
        prerelease = prerelease or base in spec.prerelease
        if not eq and base not in value_opts and not base.startswith("--") and len(base) > 2 \
                and base[:2] in value_opts:
            base, value, eq = base[:2], base[2:], "="  # -rreq.txt / -Fderive
        if not eq and base in value_opts:
            if i >= n:
                continue
            value = args[i]
            i += 1
        elif not eq:
            if not _is_boolean_opt(base, spec) and i < n and not args[i].startswith("-") \
                    and verb.kind != "unverifiable":
                if "registry" in base or "index" in base:
                    issues.append(f"option '{base}' selects a package registry/index")
                else:
                    issues.append(f"unrecognised option '{base}' before '{args[i]}': the guard cannot "
                                  f"tell whether it takes a value")
            continue
        if base in spec.source or ("registry" in base and base not in value_opts) \
                or _config_selects_registry(base, value):
            issues.append(f"option '{base}' selects a non-registry package source ({value})")
        elif base in spec.local:
            if _is_urlish(value):
                issues.append(f"option '{base}' installs from a URL/VCS source ({value})")
        elif base in spec.version:
            versions.append(value)
        elif base in spec.pkg:
            pkg_values.append(value)
        elif base in spec.extra:
            extra_values.append(value)
        elif base in spec.reqfile:
            if value in ("-", "/dev/stdin") or value.startswith("/dev/fd/") or _SUBST in value \
                    or _is_urlish(value) or "$" in value:
                issues.append(f"requirements for '{base}' come from a URL, stdin or command output "
                              f"({value.replace(_SUBST, '')!r}), not a reviewed file")
        elif base in spec.script:
            scripts.append(value)

    if verb.kind == "with":
        if not extra_values:
            return None
        candidates = [(v, False) for v in extra_values]
    elif verb.kind == "gorun":
        if not positionals or "@" not in positionals[0][0]:
            return None
        candidates = positionals[:1]
    elif verb.kind == "runner":
        candidates = [(v, False) for v in pkg_values] or positionals[:1]
        candidates += [(v, False) for v in extra_values]
    elif verb.kind == "create":
        if not positionals:
            return None  # `npm init -y`: writes package.json, installs nothing
        candidates = [(_create_package(positionals[0][0]), positionals[0][1])]
    elif verb.kind == "inject":
        candidates = positionals[1:] + [(v, False) for v in pkg_values + extra_values]
    else:
        candidates = positionals + [(v, False) for v in pkg_values + extra_values]

    if verb.kind == "unverifiable":
        remote = [t for t, _ in candidates if not _is_local(t, manifests=True)]
        if remote or issues:
            issues.append(f"'{verb.manager}' installs from a registry the package guard cannot "
                          f"verify ({', '.join(remote) or 'custom source'})")
        return [], issues, scripts

    packages: list = []
    for tok, dynamic in candidates:
        if dynamic and _is_local(tok) and "$" not in tok and _SUBST not in tok:
            continue  # a glob over local files (./dist/*.whl)
        if dynamic:
            issues.append(f"package operand '{_show(tok)}' is computed by the shell "
                          f"or supplied at run time")
            continue
        kind, detail = _classify_spec(verb.ecosystem, tok)
        if kind == "source":
            issues.append(detail)
        elif kind == "registry":
            tok = detail
            if versions:
                sep = {"RubyGems": ":", "Packagist": ":", "Pub": ":"}.get(verb.ecosystem, "@")
                tok = f"{tok}{sep}{versions[-1]}"
            elif prerelease and "@" not in tok and "::" not in tok:
                tok = f"{tok}@{NUGET_PRERELEASE_FLOAT}"  # checks the pre-release it would pick
            packages.append(tok)
    if runtime:
        issues.append("package arguments are appended at run time (xargs/parallel), so the guard "
                      "cannot see them")
    return packages, issues, scripts


class Globals(NamedTuple):
    index: int       # index of the subcommand
    unknown: list    # unrecognised options seen
    issues: list     # global options selecting another registry/index
    carried: list    # `opt=value` package options to scan with the verb (pnpm --package=x dlx)


def _config_selects_registry(base: str, value: str) -> bool:
    """`cargo --config 'source.crates-io.replace-with="x"'` and the like."""
    return base == "--config" and bool(re.search(r"(?i)registr|source\.|replace-with|index", value))


def _skip_globals(rest: list, spec: OptSpec, exe: str) -> Globals:
    """Skip a manager's global options before its subcommand. Options that
    select a registry/index become issues, and package-naming options are
    carried over to the verb's operand scan."""
    unknown: list = []
    issues: list = []
    carried: list = []
    value_opts = _value_opts(spec)
    i, n = 0, len(rest)
    while i < n:
        tok = rest[i]
        if exe == "cargo" and tok.startswith("+"):
            i += 1  # cargo +nightly install ...
            continue
        if not tok.startswith("-") or tok in ("-", "--"):
            break
        base, eq, value = tok.partition("=")
        i += 1
        if not eq and base in _TWO_VALUE_OPTS:
            value = " ".join(rest[i:i + 2])
            i += 2
        elif not eq and base in value_opts:
            value = rest[i] if i < n else ""
            i += 1
        elif base not in value_opts and not _is_boolean_opt(base, spec):
            unknown.append(base)
            if not eq:
                continue
        if base in spec.source or "registry" in base or _config_selects_registry(base, value):
            issues.append(f"option '{base}' selects a non-registry package source ({value})")
        elif base in spec.pkg or base in spec.extra:
            carried.append(f"{base}={value}")
    return Globals(i, unknown, issues, carried)


# A persistent config write that points a manager at another registry/index
# (`npm config set registry URL`, `pip config set global.index-url URL`,
# `poetry source add`, `dotnet nuget add source`, `gem sources --add`, `go env -w GOPROXY=...`): later
# installs would be checked against the public registry but fetched from the new one. TLS settings
# (`npm config set strict-ssl false`, a custom CA) are included: they decide who can serve the files.
_REGISTRY_CONFIG_RE = re.compile(r"(?i)registry|index|mirror|repositor|trusted-host|find-links|source|"
                                 r"ssl|cafile|cert|insecure")
_CONFIG_READ_ONLY = frozenset({"get", "list", "ls", "--get", "--list", "-l", "show", "debug", "edit"})


def _registry_config_change(exe: str, sub: list) -> bool:
    if sub[:2] in (["source", "add"], ["sources", "add"]) or sub[:3] == ["nuget", "add", "source"] or (
            sub[:1] == ["sources"] and ("-a" in sub or "--add" in sub)):
        return True
    if exe == "go" and sub[:1] == ["env"] and "-w" in sub:
        return any(_SOURCE_ENV_RE.match(t.partition("=")[0]) for t in sub[1:])
    return (sub[:1] == ["config"] and not _CONFIG_READ_ONLY & set(sub[1:])
            and any(_REGISTRY_CONFIG_RE.search(t) for t in sub[1:]))


def _unverified(reason: str) -> Install:
    return Install("", _UNVERIFIED_MANAGER, [], [reason], 0, [])


def _parse_manager(exe: str, rest: list, dyn: list, runtime: bool) -> Optional[Install]:
    manager = MANAGERS[exe]
    if manager.verbs[0].tokens == ():
        # npx / bunx / uvx ...: every option belongs to the runner itself.
        g = Globals(0, [], [], [])
    else:
        g = _skip_globals(rest, manager.globals, exe)
    i, unknown, global_issues, carried = g.index, list(g.unknown), list(g.issues), list(g.carried)
    while i < len(rest) and rest[i] in manager.prefixes:
        i += 1 + manager.prefixes[rest[i]]
        more = _skip_globals(rest[i:], manager.globals, exe)
        i += more.index
        unknown += more.unknown
        global_issues += more.issues
        carried += more.carried
    if exe == "dotnet" and rest[i:i + 1] == ["add"] and rest[i + 2:i + 3] == ["package"]:
        rest = rest[:i + 1] + rest[i + 2:]  # dotnet add <PROJECT> package X
        dyn = dyn[:i + 1] + dyn[i + 2:]
    for verb in manager.verbs:
        k = len(verb.tokens)
        if tuple(rest[i:i + k]) != verb.tokens:
            continue
        scanned = _scan_operands(carried + rest[i + k:], [False] * len(carried) + dyn[i + k:], verb, runtime)
        if scanned is None:
            return None
        packages, issues, scripts = scanned
        issues = global_issues + issues
        if verb.kind == "unverifiable" and not issues:
            return None
        return Install(verb.ecosystem, verb.manager, packages, issues, i + k, scripts)
    if _registry_config_change(exe, rest[i:]):
        return _unverified(f"`{exe} {' '.join(rest[i:i + 2])} ...` changes which package registry/index "
                           f"(or TLS verification) later installs use; the guard only checks the public registry")
    if i < len(rest) and dyn[i]:
        return _unverified(f"the `{exe}` subcommand '{_show(rest[i])}' is computed "
                           f"by the shell, so the guard cannot tell whether it installs packages")
    if unknown and any(t in _INSTALL_VERB_WORDS for t in rest[i:]):
        return _unverified(f"unrecognised `{exe}` option(s) {', '.join(unknown)} before an install "
                           f"verb: the guard cannot parse the command")
    return None


def _python_module(rest: list) -> Optional[tuple]:
    """For a python interpreter's arguments, return (module, index after the
    module token) for `-m module` (also `-mpip`, `-I -m pip`, `-Im pip`), or
    None when it runs a script / -c code / nothing."""
    i, n = 0, len(rest)
    while i < n:
        tok = rest[i]
        if not tok.startswith("-") or tok == "-":
            return None
        if tok == "--":
            return None
        if tok.startswith("--"):
            i += 1
            continue
        letters = tok[1:]
        for k, c in enumerate(letters):
            if c == "m":
                inline = letters[k + 1:]
                if inline:
                    return inline, i + 1
                return (rest[i + 1], i + 2) if i + 1 < n else None
            if c in _PYTHON_OPT_WITH_VALUE:
                if c == "c":
                    return None
                if not letters[k + 1:]:
                    i += 1  # value is the next token
                break
        i += 1
    return None


def _nix_installable_remote(tok: str) -> bool:
    return not tok.startswith((".", "/", "path:", "~")) and (":" in tok or "#" in tok or "/" in tok)


_NIX_GLOBAL = _opts(
    value="--extra-experimental-features --experimental-features --store --log-format --builders "
          "--max-jobs -j --cores --include -I --profile --eval-store --inputs-from --reference-lock-file "
          "--output-lock-file --expr --file -f --out-link -o --system --update-input --commit-lock-file",
    boolean="--impure -L --print-build-logs --offline --refresh -v --verbose --quiet --debug "
            "--no-write-lock-file -k --keep-going --accept-flake-config --show-trace --json "
            "--recreate-lock-file --no-update-lock-file --no-registries --dry-run --rebuild "
            "--ignore-environment -i --no-link --print-out-paths",
)


def _parse_nix(rest: list, dyn: list) -> Optional[Install]:
    i = _skip_globals(rest, _NIX_GLOBAL, "nix").index
    sub = rest[i:]
    if sub[:1] == ["profile"]:
        j = _skip_globals(sub[1:], _NIX_GLOBAL, "nix").index
        if sub[1 + j:2 + j] and sub[1 + j] in ("install", "add", "upgrade", "remove"):
            return Install("nix", "nix-profile", [], [], 0, [])
        return None
    if sub[:1] and sub[0] in ("develop", "shell", "run", "print-dev-env"):
        args = sub[1:]
        scripts: list = []
        remote: list = []
        k = 0
        while k < len(args):
            tok = args[k]
            if tok in ("-c", "--command") and sub[0] != "run":
                scripts.append(shlex.join(args[k + 1:]))
                break
            if tok == "--":
                break
            if tok in _TWO_VALUE_OPTS:
                k += 3
                continue
            if tok in _value_opts(_NIX_GLOBAL):
                k += 2
                continue
            if not tok.startswith("-") and _nix_installable_remote(tok):
                remote.append(tok)
                if sub[0] == "run":
                    break  # the rest are the program's arguments
            k += 1
        issues = [f"`nix {sub[0]}` fetches and runs code from {', '.join(remote)}, outside the "
                  f"project's pinned flake inputs"] if remote else []
        if not issues and not scripts:
            return None
        return Install("nix", _UNVERIFIED_MANAGER if issues else _SCRIPTS_ONLY, [], issues, 0, scripts)
    return None


def _parse_nix_env(rest: list, dyn: list) -> Optional[Install]:
    for tok in rest:
        if tok in ("--install", "--upgrade", "--set") or (
                tok.startswith("-") and not tok.startswith("--") and set(tok[1:]) & set("iu")):
            return Install("nix", "nix-env", [], [], 0, [])
    return None


def _parse_nix_shell(rest: list, dyn: list) -> Optional[Install]:
    scripts: list = []
    issues: list = []
    for k, tok in enumerate(rest):
        if tok in ("--run", "--command") and k + 1 < len(rest):
            scripts.append(rest[k + 1])
        elif tok in ("-p", "--packages"):
            issues.append("`nix-shell -p` pulls ad-hoc packages that are not pinned by the project")
    if not scripts and not issues:
        return None
    return Install("nix", _UNVERIFIED_MANAGER if issues else _SCRIPTS_ONLY, [], issues, 0, scripts)


def _parse_helm(rest: list, dyn: list) -> Optional[Install]:
    if rest[:2] == ["repo", "add"]:
        return _unverified("`helm repo add` registers a chart repository the guard cannot verify")
    if rest[:1] and rest[0] in ("install", "upgrade", "pull", "template"):
        refs = [t for t in rest[1:] if not t.startswith("-")]
        remote = [t for t in refs if _is_urlish(t) or ("/" in t and not _is_local(t))]
        if remote:
            return _unverified(f"`helm {rest[0]}` pulls chart(s) {', '.join(remote)} the guard cannot verify")
    return None


def _parse_mvn(rest: list, dyn: list) -> Optional[Install]:
    goals = [t for t in rest if not t.startswith("-")]
    if any(re.search(r"(?:^dependency|:maven-dependency-plugin(?::[^:]+)?):(?:get|copy)$", t) for t in goals):
        return _unverified("`mvn dependency:get/copy` downloads a Maven artifact the guard cannot verify")
    return None


# JVM tools that download (and usually run) Maven artifacts by coordinate,
# the way npx does for npm. The guard has no Maven resolver, so they are
# surfaced to the user rather than silently allowed.
_COURSIER_FETCH_VERBS = frozenset({"launch", "install", "fetch", "bootstrap", "get", "setup", "update"})


def _parse_coursier(rest: list, dyn: list) -> Optional[Install]:
    verb = next((t for t in rest if not t.startswith("-")), "")
    if verb in _COURSIER_FETCH_VERBS:
        return _unverified(f"`cs {verb}` downloads Maven artifacts the guard cannot verify")
    return None


def _parse_scala_cli(rest: list, dyn: list) -> Optional[Install]:
    for tok in rest:
        base = tok.partition("=")[0]
        if base in ("--dep", "--dependency", "--compiler-plugin", "--plugin"):
            return _unverified(f"`{base}` adds Maven dependencies the guard cannot verify")
        if not tok.startswith("-") and _is_urlish(tok):
            return _unverified(f"scala-cli runs a remote script ({tok}) the guard cannot verify")
    return None


_JBANG_LOCAL_SUFFIXES = (".java", ".jsh", ".kt", ".groovy", ".md", ".jar")
_JBANG_OTHER_COMMANDS = frozenset({"init", "edit", "cache", "jdk", "config", "info", "export", "catalog",
                                   "alias", "template", "wrapper", "version", "completion", "help", "trust"})


def _parse_jbang(rest: list, dyn: list) -> Optional[Install]:
    """jbang runs a local script, or fetches code by Maven coordinate, URL or
    catalog alias (`jbang g:a:v`, `jbang app install x@catalog`)."""
    operands: list = []
    for tok in rest:
        base = tok.partition("=")[0]
        if base in ("--deps", "--repos", "--catalog"):
            return _unverified(f"jbang `{base}` fetches artifacts the guard cannot verify")
        if not tok.startswith("-"):
            operands.append(tok)
    if operands[:1] and operands[0] in _JBANG_OTHER_COMMANDS:
        if operands[0] == "trust" or operands[0] in ("catalog", "alias") and operands[1:2] == ["add"]:
            return _unverified("jbang is being pointed at a new code source the guard cannot verify")
        return None
    if operands[:2] == ["app", "install"]:
        operands = operands[2:]
    elif operands[:1] in (["run"], ["build"]):
        operands = operands[1:]
    if not operands:
        return None
    ref = operands[0]
    if not _is_urlish(ref) and (_is_local(ref) or ref.lower().endswith(_JBANG_LOCAL_SUFFIXES)):
        return None
    return _unverified(f"jbang fetches and runs '{ref}' (Maven coordinate, URL or catalog alias), which "
                       f"the guard cannot verify")


def _parse_clojure(rest: list, dyn: list) -> Optional[Install]:
    if any(t == "-Sdeps" or t.startswith("-Sdeps") for t in rest):
        return _unverified("`-Sdeps` adds ad-hoc dependencies the guard cannot verify")
    tool = next((t for t in rest if t.startswith("-T")), None)
    if tool is not None and any(t in ("install", "install-latest") for t in rest):
        return _unverified(f"`clojure {tool} install` fetches a tool the guard cannot verify")
    return None


# Installers the guard has no registry validation for (no OSV / publication-age
# lookup exists for their ecosystem), so nothing they fetch can be checked. Like
# imperative Nix installs they are denied outright: a new dependency is declared
# in the project manifest and installed from the lockfile instead.
# Executable -> subcommand token sequences that install, or re-resolve and
# rewrite the lockfile; an empty sequence means every invocation installs
# (bare `cpan` is its interactive shell, which also reads commands from stdin).
# A sequence may appear anywhere after the executable, so global flags placed
# before the verb (`luarocks --tree /x install foo`) cannot hide it. A
# versioned executable (`luarocks-5.4`) is matched by its base name.
UNVALIDATED_INSTALLS: dict = {
    "cpan":     [[]],  # `cpan Foo::Bar` installs; -i is implied
    "cpanm":    [[]],
    "cpm":      [["install"]],
    # carton install re-resolves cpanfile and rewrites cpanfile.snapshot unless
    # --deployment (see UNVALIDATED_FROZEN_FLAGS).
    "carton":   [["install"], ["update"]],
    "mix":      [["archive.install"], ["escript.install"], ["deps.update"],
                 ["deps.unlock"], ["igniter.install"]],
    "zig":      [["fetch"]],
    "dart":     [["pub", "add"], ["pub", "upgrade"], ["pub", "downgrade"],
                 ["pub", "global", "activate"]],
    "flutter":  [["pub", "add"], ["pub", "upgrade"], ["pub", "downgrade"],
                 ["pub", "global", "activate"]],
    "lx":       [["add"], ["install"]],
    "luarocks": [["install"], ["build"]],
}

# (executable, verb) -> flag that turns the verb into a restore of exactly what
# the lockfile pins, which stays allowed.
UNVALIDATED_FROZEN_FLAGS: dict = {
    ("carton", "install"): "--deployment",
}

# Manager-label prefix for installs through an UNVALIDATED_INSTALLS manager (or
# a PowerShell / R / perl -MCPAN installer); these are denied outright.
UNVALIDATED_MANAGER_PREFIX = "unvalidated:"

# A version suffix on an installer executable (`luarocks-5.4`, `luarocks5.1`).
_EXE_VERSION_SUFFIX_RE = re.compile(r"-?[0-9][0-9.]*$")

# PowerShell hosts (names as _exe_name normalises them) and the cmdlets (and
# PSResourceGet's aliases isres/udres) that install or update from PSGallery /
# NuGet. Cmdlet names are case-insensitive.
_POWERSHELLS: frozenset = frozenset({"pwsh", "powershell"})
_PS_INSTALL_RE = re.compile(
    r"(?<![\w-])(install-module|install-psresource|save-module|save-psresource|"
    r"install-script|install-package|update-module|update-script|"
    r"update-psresource|isres|udres)(?![\w-])",
    re.IGNORECASE,
)
# PowerShell parameter names are case-insensitive, accept a `/` prefix on
# Windows PowerShell and may be abbreviated: -c/-com.../-Command take the rest
# of the line as a script, -e/-ec/-enc.../-EncodedCommand a base64 UTF-16LE
# script.
_PS_COMMAND_FLAG_RE = re.compile(
    r"^(?:--?|/)(c|com|comm|comma|comman|command|cwa|commandwithargs)$", re.IGNORECASE)
_PS_ENCODED_FLAG_RE = re.compile(r"^(?:--?|/)(e|ec|en|enc|enco|encod|encode|encoded[a-z]*)$", re.IGNORECASE)

# R interpreters (as _exe_name normalises them) whose `-e <expr>` runs R code;
# any expression that installs (install.packages, remotes::install_github,
# renv::install, pak::pkg_install, BiocManager::install) fetches and runs
# unvalidated package code.
_R_EXES: frozenset = frozenset({"r", "rscript"})
_R_INSTALL_RE = re.compile(
    r"install|update\.packages|\bpak(::pak)?\s*\(|pak::|renv::(update|hydrate)",
    re.IGNORECASE,
)

# `perl -MCPAN -e 'install Foo'`, `perl -M CPAN ...`, `perl -e 'use CPAN; ...'`
# and cpanminus/cpm driven as modules. CPAN::Meta and friends are not
# installers, so only CPAN itself (or CPAN::Shell) matches.
_PERL_CPAN_RE = re.compile(r"\bCPAN(::Shell)?\b(?!::)|\bApp::(cpanminus|cpm)\b")


def _contains_sequence(args: list, seq: list) -> bool:
    """True when seq occurs as consecutive tokens anywhere in args (an empty
    seq always matches)."""
    n = len(seq)
    return any(args[i:i + n] == seq for i in range(len(args) - n + 1))


def _unvalidated_base(exe: str) -> str:
    """The UNVALIDATED_INSTALLS key exe names (`luarocks-5.4` -> `luarocks`),
    or exe unchanged."""
    if exe not in UNVALIDATED_INSTALLS:
        base = _EXE_VERSION_SUFFIX_RE.sub("", exe)
        if base in UNVALIDATED_INSTALLS:
            return base
    return exe


def _pwsh_script(args: list) -> Optional[str]:
    """The script a pwsh/powershell invocation runs: the rest of the line after
    -Command (or its abbreviations), or the decoded -EncodedCommand. Returns
    None when it runs no inline script (-File, interactive). Raises ValueError
    for an -EncodedCommand that cannot be decoded, so the caller fails closed."""
    for i, a in enumerate(args):
        if _PS_COMMAND_FLAG_RE.match(a):
            return " ".join(args[i + 1:])
        if _PS_ENCODED_FLAG_RE.match(a):
            if i + 1 >= len(args):
                return None
            try:
                return base64.b64decode(args[i + 1], validate=True).decode("utf-16-le")
            except (binascii.Error, UnicodeDecodeError) as exc:
                raise ValueError(f"undecodable -EncodedCommand: {exc}") from exc
        if not a.startswith(("-", "/")):
            return None  # first positional: a script file (pwsh) or command
    return None


def _perl_cpan(args: list) -> bool:
    """True when perl loads the CPAN installer: -MCPAN, -M CPAN, or an -e/-E
    program (possibly in a switch cluster such as -wle) that uses it."""
    return any(_PERL_CPAN_RE.search(a[2:] if a[:2] in ("-M", "-m") else a) for a in args)


def _unvalidated_install(exe: str, args: list) -> bool:
    """True when `exe args` installs through a manager in UNVALIDATED_INSTALLS,
    a PowerShell installer cmdlet, an installing R expression, or perl's CPAN
    module. Raises ValueError for a PowerShell script that cannot be decoded."""
    exe = _unvalidated_base(exe)
    if exe in UNVALIDATED_INSTALLS:
        for seq in UNVALIDATED_INSTALLS[exe]:
            if not _contains_sequence(args, seq):
                continue
            frozen = UNVALIDATED_FROZEN_FLAGS.get((exe, seq[0])) if seq else None
            if frozen is None or frozen not in args:
                return True
        return False
    if exe in _POWERSHELLS:
        script = _pwsh_script(args)
        return any(_PS_INSTALL_RE.search(a) for a in args + ([script] if script else []))
    if exe in _R_EXES:
        for i, a in enumerate(args):
            if a in ("-e", "--expr"):
                expr = args[i + 1] if i + 1 < len(args) else ""
            else:
                expr = a[2:] if a.startswith("-e") else ""
            if _R_INSTALL_RE.search(expr):
                return True
        return False
    if exe == "perl":
        return _perl_cpan(args)
    return False


def _is_unvalidated_manager(exe: str) -> bool:
    """True when exe is an installer _unvalidated_install classifies."""
    return (_unvalidated_base(exe) in UNVALIDATED_INSTALLS or exe in _POWERSHELLS
            or exe in _R_EXES or exe == "perl")


def _parse_pwsh(rest: list, dyn: list) -> Optional[Install]:
    """pwsh -Command / -EncodedCommand: the script runs native commands
    (`npm install x`) too, so it is scanned like a shell -c. Installer cmdlets
    are denied by _unvalidated_install before this runs."""
    try:
        script = _pwsh_script(rest)
    except ValueError as exc:
        return _unverified(f"PowerShell {exc}; the guard cannot inspect it")
    if script is None:
        return None
    return Install("", _SCRIPTS_ONLY, [], [], 0, [script])


# Deno subcommands that fetch registry packages. add/install add dependencies
# (a bare `deno install` installs from deno.json, like `npm install`); run,
# serve, x and create download and execute a package, like npx / npm create.
# `deno <module>` (no subcommand) is an implicit `deno run`.
_DENO_INSTALL_VERBS: frozenset = frozenset({"add", "install", "i"})
_DENO_EXEC_VERBS: frozenset = frozenset({"run", "serve", "x", "create"})
# Deno flags that consume the following token as their value.
_DENO_VALUE_FLAGS: frozenset = frozenset({
    "--config", "-c", "--import-map", "--lock", "--cert", "--location",
    "--seed", "--v8-flags", "--env-file", "--ext", "--root", "--name", "-n",
    "--os", "--arch", "--log-level", "-L", "--preload", "--require",
    "--port", "--host",
})
# Specifier prefixes naming a registry package in deno.
_NPM_PREFIX = "npm:"
_JSR_PREFIX = "jsr:"
_DENO_REGISTRY_PREFIXES = (_NPM_PREFIX, _JSR_PREFIX)
# Operands that name a local module or a URL rather than a registry package.
_DENO_LOCAL_PREFIXES = ("./", "../", "/", "~", "file:", "http:", "https:", "data:")
_DENO_SCRIPT_SUFFIXES = (
    ".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs", ".json", ".jsonc",
)

# JSR packages have no OSV feed or age check in this guard, so they cannot be
# validated automatically and are escalated to the user.
_UNVALIDATED_PREFIXES = (_JSR_PREFIX,)


def _deno_package(specifier: str) -> str:
    """Map a deno `npm:` specifier to the npm package (with version, without
    any subpath): `npm:@scope/pkg@1.2/bin` -> `@scope/pkg@1.2`. `jsr:`
    specifiers are returned unchanged."""
    if not specifier.startswith(_NPM_PREFIX):
        return specifier
    spec = specifier[len(_NPM_PREFIX):]
    keep = 2 if spec.startswith("@") else 1
    return "/".join(spec.split("/")[:keep])


def _deno_operands(tokens: list, dyn: list) -> tuple:
    """Split deno arguments into positional operands (token, computed-by-the-
    shell) and the flags seen. Arguments after `--` belong to the executed
    program and are dropped, except a module named right after it (`deno run
    -- npm:cli`). A registry specifier is always kept as an operand, even where
    a value flag would consume it, so a misjudged flag can never hide a
    package."""
    operands: list = []
    flags: set = set()
    i, n = 0, len(tokens)
    while i < n:
        tok = tokens[i]
        i += 1
        if tok == "--":
            if not operands and i < n:
                operands.append((tokens[i], dyn[i]))
            break
        if tok.startswith(_DENO_REGISTRY_PREFIXES) or not tok.startswith("-"):
            operands.append((tok, dyn[i - 1]))
            continue
        flags.add(tok.split("=", 1)[0])
        if "=" not in tok and tok in _DENO_VALUE_FLAGS and i < n:
            if not tokens[i].startswith(_DENO_REGISTRY_PREFIXES):
                i += 1  # consume the flag's value
    return operands, flags


def _deno_registry_package(operand: str, bare: str) -> Optional[str]:
    """Return the registry specifier an operand names, or None for a local
    module or URL. `bare` is the prefix given to an unprefixed name ("" when
    unprefixed names are local modules in this position)."""
    if operand.startswith(_DENO_REGISTRY_PREFIXES):
        return _deno_package(operand)
    if not bare or operand.startswith(_DENO_LOCAL_PREFIXES):
        return None
    return _deno_package(bare + operand)


def _deno_candidates(operands: list, bare: str) -> tuple:
    """(registry packages, issues) for deno operands: a computed operand or a
    remote module URL cannot be verified and is surfaced to the user."""
    packages: list = []
    issues: list = []
    for tok, computed in operands:
        if computed:
            issues.append(f"deno operand '{_show(tok)}' is computed by the shell")
        elif _is_urlish(tok):
            issues.append(f"deno fetches the remote module '{tok}', which the guard cannot verify")
        else:
            pkg = _deno_registry_package(tok, bare)
            if pkg:
                packages.append(pkg)
    return packages, issues


def _parse_deno(rest: list, dyn: list) -> Optional[Install]:
    """Classify a deno invocation (argv after `deno`). Registry packages are:
    every operand of add/install (Deno >= 2.8 treats unprefixed names as npm
    packages; `install -e`/`-g` operands may be local modules), and the module
    run by run/serve/x/create or by an implicit `deno <module>`. `deno run
    main.ts` runs a local script and is not an install."""
    if not rest:
        return None
    verb = rest[0]
    if verb.startswith("-") or verb.startswith(_DENO_REGISTRY_PREFIXES):
        verb, args, adyn = "run", rest, dyn  # `deno -A npm:cli` == `deno run -A npm:cli`
    elif verb in _DENO_INSTALL_VERBS or verb in _DENO_EXEC_VERBS:
        args, adyn = rest[1:], dyn[1:]
    else:
        return None
    operands, flags = _deno_operands(args, adyn)

    if verb in _DENO_INSTALL_VERBS:
        if flags & {"-e", "--entrypoint"}:
            bare = ""  # operands are local entrypoint modules
        elif flags & {"-g", "--global"}:
            # A global install takes a package, URL or local script.
            operands = [o for o in operands if not o[0].endswith(_DENO_SCRIPT_SUFFIXES)]
            bare = _NPM_PREFIX
        else:
            bare = _NPM_PREFIX
        packages, issues = _deno_candidates(operands, bare)
        return Install("npm", "deno", packages, issues, 0, [])

    if not operands:
        return None
    target = operands[0]  # later operands are the program's own arguments
    if verb == "x":
        bare = "" if target[0].endswith(_DENO_SCRIPT_SUFFIXES) else _NPM_PREFIX
    elif verb == "create":
        bare = _JSR_PREFIX if "--jsr" in flags else _NPM_PREFIX if "--npm" in flags else ""
    else:
        bare = ""  # run/serve: an unprefixed module is a local file
    packages, issues = _deno_candidates([target], bare)
    if not packages and not issues:
        return None
    return Install("npm", "deno", packages, issues, 0, [])


def _parse_cmd(rest: list, dyn: list) -> Optional[Install]:
    """cmd.exe /c "<command line>" (`//c` under Git Bash path conversion)."""
    for k, tok in enumerate(rest):
        if re.fullmatch(r"/+[ckCK]", tok):
            return Install("", _SCRIPTS_ONLY, [], [], 0, [" ".join(rest[k + 1:])])
    return None


_SPECIAL_PARSERS: dict = {
    "nix": _parse_nix, "nix-env": _parse_nix_env, "nix-shell": _parse_nix_shell,
    "helm": _parse_helm, "mvn": _parse_mvn, "deno": _parse_deno,
    "pwsh": _parse_pwsh, "powershell": _parse_pwsh, "clojure": _parse_clojure, "clj": _parse_clojure,
    "cmd": _parse_cmd, "mvnw": _parse_mvn, "cs": _parse_coursier, "coursier": _parse_coursier,
    "scala-cli": _parse_scala_cli, "scala": _parse_scala_cli, "jbang": _parse_jbang,
}


def modeled_executables() -> set:
    """Every executable the guard parses (for the catalog coverage test)."""
    return (set(MANAGERS) | set(_SPECIAL_PARSERS) | set(UNVALIDATED_INSTALLS) | _POWERSHELLS | _R_EXES
            | {"python", "perl"})


def parse_install_argv(argv: list, dyn: Optional[list] = None, runtime: bool = False) -> Optional[Install]:
    """Parse an unwrapped argv. Returns an Install when it is a package install
    (or an install the guard must surface as unverifiable), else None.
    Detection is argv-based: the executable and its subcommand verb must
    actually be an install command, so install-like words inside unrelated
    commands are never treated as packages. `python -m pip|uv|...` resolves to
    the underlying installer."""
    if not argv:
        return None
    if dyn is None:
        dyn = [False] * len(argv)
    exe = _exe_name(argv[0])
    rest, rdyn, shift = argv[1:], dyn[1:], 1
    if exe == "python":
        found = _python_module(rest)
        if found is None:
            return None
        module, k = found
        exe = _exe_name(module)
        rest, rdyn, shift = rest[k:], rdyn[k:], 1 + k
    try:
        unvalidated = _unvalidated_install(exe, rest)
    except ValueError as exc:
        return Install("unvalidated", UNVALIDATED_MANAGER_PREFIX + exe, [],
                       [f"PowerShell {exc}; blocked to fail closed because the guard cannot inspect it"], 0, [])
    if unvalidated:
        return Install("unvalidated", UNVALIDATED_MANAGER_PREFIX + _unvalidated_base(exe), [], [], 0, [])
    if exe in _SPECIAL_PARSERS:
        return _SPECIAL_PARSERS[exe](rest, rdyn)
    if exe not in MANAGERS:
        return None
    inst = _parse_manager(exe, rest, rdyn, runtime)
    if inst is None:
        return None
    return inst._replace(verb_end=inst.verb_end + shift)


def _classified_exe(exe: str) -> bool:
    """True when `exe` is an executable the classifier already understands. The
    embedded-install fallback must not second-guess these: a known manager with
    a non-install verb (`go build ./...`) really is not an install."""
    return (exe in MANAGERS or exe in _SPECIAL_PARSERS or exe in _SHELLS or exe == "python"
            or _unvalidated_base(exe) in UNVALIDATED_INSTALLS
            or exe in _NON_EXEC_COMMANDS or exe in _PRIV_C_RUNNERS)


def _embedded_install(argv: list, dyn: list, runtime: bool) -> Optional[tuple]:
    """First genuine install invocation embedded past argv[0] (an exec wrapper
    the guard does not model, `find -exec`, ...): the token must itself BE a
    modeled manager whose tail parses as an install. Returns (Install, index).
    A manager token with nothing after it is a name, not an invocation (`which
    cpan`, `man cpanm`), so the last token is never a candidate."""
    for k in range(1, len(argv) - 1):
        exe = _exe_name(argv[k])
        if exe in MANAGERS or exe in _SPECIAL_PARSERS or exe == "python" or _is_unvalidated_manager(exe):
            parsed = parse_install_argv(argv[k:], dyn[k:], runtime)
            if parsed is not None:
                return parsed, k
    return None


def _show(word: str) -> str:
    """A word for a message, with masked substitution spans shown as $(...)."""
    return re.sub(_SUBST + "+", "$(...)", word)


def _mentions_install(texts: list) -> bool:
    """For a command whose name the shell computes: does any of `texts` (its
    words, the values of variables assigned earlier in the command, the
    substitutions it runs) name a package manager or an install verb? Words are
    split on every non-name character, so `npm${IFS}install` and `{npm,i}`
    still read as `npm` / `install`."""
    pieces = [p for t in texts for p in re.split(r"[^A-Za-z0-9._+-]+", t)]
    return any(_exe_name(p) in MANAGERS or _exe_name(p) in _SPECIAL_PARSERS or p in _INSTALL_VERB_WORDS
               for p in pieces if p)


# ---------------------------------------------------------------------------
# Safety flags
# ---------------------------------------------------------------------------

def _env_value(env: list, name: str) -> Optional[str]:
    for assignment in env:
        key, _, value = assignment.partition("=")
        if key.lower() == name:
            return value
    return None


# Options and environment settings that switch off transport or checksum
# verification. Unlike a registry override (asked about), these are blocked:
# they let a network attacker or mirror substitute the artifact itself.
_INSECURE_OPTS = frozenset({"--trusted-host", "--allow-insecure-host", "-insecure", "--no-strict-ssl"})
# Boolean options that disable TLS verification when set to false
# (`--strict-ssl=false`, or `--strict-ssl false`, which npm's parser accepts).
_TLS_BOOL_OPTS = frozenset({"--strict-ssl"})
_FALSY = frozenset({"false", "0", "no", "off"})
# Environment settings that disable verification whenever they are non-empty,
# and those that do so only with a false-like value.
_INSECURE_ENV_ANY = frozenset({"GONOSUMCHECK", "GOINSECURE", "PIP_TRUSTED_HOST", "UV_INSECURE_HOST",
                               "GIT_SSL_NO_VERIFY"})
_INSECURE_ENV_FALSY = frozenset({"NODE_TLS_REJECT_UNAUTHORIZED", "NPM_CONFIG_STRICT_SSL",
                                 "YARN_ENABLE_STRICT_SSL", "YARN_STRICT_SSL", "GOSUMDB"})


def _integrity_overrides(argv: list, env: list) -> list:
    reasons = [f"option '{tok.partition('=')[0]}' disables TLS/checksum verification of downloads"
               for tok in argv if tok.partition("=")[0] in _INSECURE_OPTS]
    for i, tok in enumerate(argv):
        base, eq, value = tok.partition("=")
        if base in _TLS_BOOL_OPTS and (value if eq else (argv[i + 1] if i + 1 < len(argv) else "")
                                       ).strip("'\"").lower() in _FALSY:
            reasons.append(f"option '{base}' set to false disables TLS verification of downloads")
    for assignment in env:
        name, _, value = assignment.partition("=")
        key, value = name.upper(), value.strip().strip("'\"")
        if (key in _INSECURE_ENV_ANY and value) or (key in _INSECURE_ENV_FALSY and value.lower() in _FALSY) \
                or (key == "GOFLAGS" and "-insecure" in value):
            reasons.append(f"{name}={value} disables TLS/checksum verification of downloaded packages")
    return reasons


def _safety(manager: str, argv: list, env: list) -> tuple:
    """(flag still needed or "", conflict reason or "") for an install."""
    flag = SAFETY_FLAGS.get(manager, "")
    if not flag:
        return "", ""
    if manager in ("npm", "bun"):
        for k, tok in enumerate(argv):
            base, eq, value = tok.partition("=")
            if base == "--no-ignore-scripts" or (base == "--ignore-scripts" and (
                    (eq and value.lower() in ("false", "0", "")) or
                    (not eq and argv[k + 1:k + 2] == ["false"]))):
                return "", f"the command re-enables {manager} lifecycle scripts (--ignore-scripts=false)"
            if base == "--ignore-scripts":
                return "", ""
        value = _env_value(env, "npm_config_ignore_scripts") if manager == "npm" else None
        if value is not None and value.lower() in ("false", "0", ""):
            return "", "npm_config_ignore_scripts=false re-enables npm lifecycle scripts"
        return flag, ""
    if manager in ("pip", "uv-pip"):
        has_all = False
        for k, tok in enumerate(argv):
            base, eq, value = tok.partition("=")
            if base == "--no-binary":
                return "", "the command forces source builds (--no-binary), which run setup code"
            if base == "--only-binary":
                chosen = value if eq else "".join(argv[k + 1:k + 2])
                if ":none:" in chosen:
                    return "", "the command clears --only-binary (:none:), re-enabling source builds"
                has_all = has_all or chosen == ":all:"
        for name in ("pip_no_binary", "uv_no_binary", "uv_no_binary_package"):
            if _env_value(env, name):
                return "", f"{name.upper()} forces source builds, which run setup code"
        return ("" if has_all else flag), ""
    if manager == "cargo":
        return ("" if ("--locked" in argv or "--frozen" in argv) else flag), ""
    return "", ""


# ---------------------------------------------------------------------------
# Command scanning
# ---------------------------------------------------------------------------

def _suspicious_detection(segment: str) -> Detection:
    """A fail-closed detection marker for a segment that could not be safely
    analyzed (unparseable, or nested past the recursion cap). main() denies these
    so the guard never silently allows an install it could not inspect."""
    return Detection("suspicious", _SUSPICIOUS_MANAGER, segment[:200], [])


def _unverified_detection(segment: str, reason: str) -> Detection:
    return Detection("", _UNVERIFIED_MANAGER, segment[:200], [], (reason,))


def _recurse_script(script: str, depth: int, results: list, script_reader: bool = False,
                    env: tuple = ()) -> None:
    """Recursively scan a nested shell script (a shell/su/eval -c body, a
    command/process substitution, a heredoc read by a shell). FAILS CLOSED past
    the recursion cap: a suspicious marker is emitted instead of silently
    dropping the script, so a pathologically nested install still forces
    validation."""
    script = script.strip()
    if not script:
        return
    if depth >= _MAX_SHELL_RECURSION:
        results.append(_suspicious_detection(script))
        return
    results.extend(detect_install_commands(script, depth + 1, script_reader, list(env)))


def _scan_stdin_reader(seg: Segment, heres: list, prev_fed_heredoc: bool, depth: int,
                       results: list, env: list) -> None:
    """A shell (or `source /dev/stdin`) reading its script from stdin: scan a
    literal here-string; anything else fed through a pipe is unknown text."""
    for text, dynamic in heres:
        if dynamic:
            results.append(_unverified_detection(seg.text, "a shell runs a here-string the shell "
                                                           "computes; the guard cannot inspect it"))
        else:
            _recurse_script(text, depth, results, env=env)
    if not heres and seg.piped and not prev_fed_heredoc:
        results.append(_unverified_detection(seg.text, "a shell runs a script piped from another "
                                                       "command; the guard cannot inspect it"))


def _embedded_shell_script(argv: list, depth: int, results: list, env: list) -> None:
    """A shell -c embedded in an unknown command (`find . -exec sh -c '...' ';'`,
    `git rebase --exec` style runners): scan its script."""
    for k in range(1, len(argv)):
        if _exe_name(argv[k]) in _SHELLS:
            call = _shell_invocation(argv[k:])
            if call.script is not None:
                _recurse_script(call.script.replace(_SUBST, " "), depth, results, env=env)
                return


def _scan_segment(seg: Segment, words: list, heres: list, u: Unwrapped, depth: int,
                  results: list, env: list, prev_fed_heredoc: bool, subs: list) -> None:
    """Classify one tokenized segment, appending its detections. `env` holds
    the assignments made earlier in the command (and by enclosing scripts),
    `subs` the command substitutions of this command line."""
    scope_env = env + u.env
    if u.script is not None:
        _recurse_script(u.script, depth, results, env=scope_env)
    argv, dyn = u.argv, u.dyn
    if not argv:
        return
    if u.cmd_dyn:
        context = [a.partition("=")[2] for a in scope_env] + (subs if _SUBST in argv[0] else [])
        if _mentions_install(argv + context):
            results.append(_unverified_detection(
                seg.text, f"the command name '{_show(argv[0])}' is computed by the "
                          f"shell, so the guard cannot tell whether it installs packages"))
        return
    exe = _exe_name(argv[0])

    # `eval "<script>"`: eval concatenates its arguments into a shell script.
    if exe == "eval":
        script = " ".join(argv[1:])
        if _SUBST in script:
            results.append(_unverified_detection(seg.text, "eval runs the output of a command "
                                                           "substitution; the guard cannot inspect it"))
        _recurse_script(script.replace(_SUBST, " "), depth, results, env=scope_env)
        return
    if exe in _SHELLS:
        call = _shell_invocation(argv)
        if call.script is not None:
            if _SUBST in call.script:
                results.append(_unverified_detection(seg.text, "a shell runs the output of a command "
                                                               "substitution; the guard cannot inspect it"))
            _recurse_script(call.script.replace(_SUBST, " "), depth, results, env=scope_env)
        elif call.reads_stdin:
            _scan_stdin_reader(seg, heres, prev_fed_heredoc, depth, results, scope_env)
        elif _SUBST in call.operand or call.operand.startswith("/dev/fd/"):
            results.append(_unverified_detection(seg.text, "a shell runs a script produced by another "
                                                           "command (`<(...)`); the guard cannot inspect it"))
        return
    if exe in ("source", "."):
        target = argv[1] if len(argv) > 1 else ""
        if _SUBST in target:
            results.append(_unverified_detection(seg.text, "the shell sources the output of a "
                                                           "command; the guard cannot inspect it"))
        elif target in ("/dev/stdin", "-") or target.startswith("/dev/fd/"):
            _scan_stdin_reader(seg, heres, prev_fed_heredoc, depth, results, scope_env)
        return

    off = u.off
    if exe in _PRIV_C_RUNNERS:
        script = _c_runner_script(argv)
        if script is not None:
            _recurse_script(script, depth, results, env=scope_env)
            return
        k = _skip_option_flags(argv, 1, _PRIV_VALUE_FLAGS)
        argv, dyn, off = argv[k:], dyn[k:], (off + k if off >= 0 else -1)
        if not argv:
            return
        exe = _exe_name(argv[0])

    inst = parse_install_argv(argv, dyn, u.runtime)
    if inst is None and not _classified_exe(exe):
        # Catalog-driven fallback: an exec wrapper the guard does not model
        # (or a wrapper option it does not know) hides the install behind an
        # unknown argv[0]; re-classify from the embedded manager token.
        found = _embedded_install(argv, dyn, u.runtime)
        if found is not None:
            inst, k = found
            argv, off = argv[k:], (off + k if off >= 0 else -1)
        else:
            _embedded_shell_script(argv, depth, results, scope_env)
    if inst is None:
        return
    for script in inst.scripts:
        _recurse_script(script, depth, results, env=scope_env)

    issues = list(inst.issues)
    all_env = scope_env
    for assignment in all_env:
        if _SOURCE_ENV_RE.match(assignment.partition("=")[0]):
            issues.append(f"environment override {assignment.partition('=')[0]} points the "
                          f"package manager at a different registry/index")
    denials = _integrity_overrides(argv, all_env)
    flag, conflict = _safety(inst.manager, argv, all_env)
    if conflict:
        issues.append(conflict)
    insert_at = -1
    # Rewrite only a top-level argv that is a verbatim slice of the command,
    # so the flag lands right after this install's verb and nowhere else.
    if flag and depth == 0 and off >= 0 and seg.verbatim and 0 < inst.verb_end <= len(argv):
        insert_at = seg.start + words[off + inst.verb_end - 1].end
    if inst.manager.startswith(UNVALIDATED_MANAGER_PREFIX):
        tool = inst.manager[len(UNVALIDATED_MANAGER_PREFIX):]
        reasons = list(inst.issues) or [
            f"`{tool}` installs packages the package guard cannot validate (no vulnerability or "
            f"publication-age check exists for its registry). Declare the dependency in the project "
            f"manifest and install it from the lockfile, or ask the user to run it."]
        results.append(Detection("unvalidated", inst.manager, seg.text, [], (), "", -1,
                                 tuple(reasons) + tuple(denials)))
        return
    if inst.manager == _UNVERIFIED_MANAGER:
        results.append(_unverified_detection(seg.text, " | ".join(issues))._replace(denials=tuple(denials)))
        return
    if inst.manager == _SCRIPTS_ONLY:
        return  # nix develop -c, nix-shell --run, pwsh -Command: only the scripts matter
    results.append(Detection(inst.ecosystem, inst.manager, seg.text, inst.packages, tuple(issues),
                             f" {flag}" if flag else "", insert_at, tuple(denials)))


def detect_install_commands(command: str, _depth: int = 0, _script_reader: bool = False,
                            _env: Optional[list] = None) -> list:
    """
    Find ALL genuine package install invocations in a (possibly compound or
    multi-line) command. Returns a list of Detection tuples -- one per segment
    whose argv is actually an install command, with the raw package specifiers
    extracted from that segment's single parse. Every segment is parsed
    independently so that compound commands like
    ``pip install safe && npm install evil`` cannot sneak an unchecked install
    past the guard, while unrelated commands whose text merely *mentions* an
    install (``git commit -m "add install docs"``, ``grep "npm install" file``)
    are correctly ignored.

    Classification happens at a command POSITION (argv[0] after environment
    assignments, reserved words and exec wrappers are stripped), at every
    recursion depth. Installs hidden behind a shell construct are surfaced by
    recursing into it: command / process substitutions, shell and privilege
    -c runners, eval, here-strings and heredocs read by a shell, and the -c /
    --command scripts of nix develop/shell, nix-shell, npx and flock. A
    catalog-driven fallback re-classifies from an embedded manager token when
    argv[0] is an unknown exec wrapper (``setpriv npm install evil``); builtins
    that never execute their arguments (echo, printf, ...) are exempt.

    Recursion is bounded by _MAX_SHELL_RECURSION; exceeding it, or hitting an
    unparseable segment, FAILS CLOSED via a suspicious marker. An install the
    guard recognises but cannot verify (computed names, unknown options,
    non-registry sources) yields an unverified marker that main() asks about.
    """
    results: list = []

    # Command/process substitutions anywhere in the command are independent
    # scripts; the cleaned command keeps its offsets with the spans masked.
    sub_scripts, command = _extract_substitutions(command)

    # A heredoc body is data, except when something in this command (or in the
    # command a substitution's output feeds) may run it as a script:
    # `bash <<EOF`, `cat <<EOF | sh`, `source /dev/stdin <<EOF`,
    # `bash -c "$(cat <<EOF ...)"`. `_script_reader` carries that from the parent.
    parsed: list = []
    script_reader = _script_reader
    env: list = list(_env or [])
    for seg in _split_segments(command):
        try:
            words, heres = _tokenize(seg.text)
            u = _unwrap(words)
        except ValueError:
            # Unbalanced quotes etc.: we cannot trust any tokenization, so we
            # cannot rule out a hidden install. Fail closed instead of guessing.
            parsed.append((seg, None, [], None))
            continue
        if u.argv and _exe_name(u.argv[0]) in _SCRIPT_READERS:
            script_reader = True
        # `export VAR=x` / a bare `VAR=x` statement applies to later commands.
        if (u.argv[:1] and u.argv[0] in ("export", "declare", "typeset", "readonly")):
            env += [t for t in u.argv[1:] if _ENV_ASSIGN_RE.match(t)]
        elif not u.argv:
            env += u.env
        parsed.append((seg, words, heres, u))

    for script in sub_scripts:
        _recurse_script(script, _depth, results, script_reader, env)

    prev_fed_heredoc = False
    for seg, words, heres, u in parsed:
        if script_reader:
            for body in seg.heredocs:
                _recurse_script(body, _depth, results, env=env)
        if words is None:
            results.append(_suspicious_detection(seg.text))
        else:
            _scan_segment(seg, words, heres, u, _depth, results, env,
                          prev_fed_heredoc and seg.piped, sub_scripts)
        prev_fed_heredoc = bool(seg.heredocs) and script_reader

    return results


def apply_safety_flags(command: str, detections: list) -> tuple:
    """Insert each detection's missing safety flag right after its install
    verb, in that exact argv. Returns (rewritten command, reasons for installs
    whose flag could not be inserted)."""
    inserts: list = []
    failed: list = []
    for d in detections:
        if not d.flag:
            continue
        if d.insert_at < 0:
            failed.append(
                f"`{d.segment[:80]}` needs{d.flag} but runs inside a nested shell, pipeline wrapper or "
                f"multi-line construct the guard cannot rewrite safely. Add{d.flag} to the install "
                f"yourself, or run it as a standalone command.")
        else:
            inserts.append((d.insert_at, d.flag))
    rewritten = command
    for at, flag in sorted(set(inserts), reverse=True):
        rewritten = rewritten[:at] + flag + rewritten[at:]
    return rewritten, failed


# ---------------------------------------------------------------------------
# Core validation logic
# ---------------------------------------------------------------------------

def validate_package(package_name: str, ecosystem: str, kind: str = "none", value: str = "") -> tuple:
    """
    Validate a single package. Returns (decision, reason, resolved_version).
    decision is one of: "allow", "deny", "ask". The version checked is the one
    the manager would install (registry latest for an unpinned install, the
    highest release matching a range), never a string with operators stripped.
    """
    # 0a. Team denylist (checked before everything).
    if TEAM_DENYLIST.matches(package_name, ecosystem):
        return "deny", f"Package '{package_name}' is on the team denylist per consulting policy.", ""

    # 0b. Team allowlist (pre-approved, bypass vuln/age checks).
    if TEAM_ALLOWLIST:
        if TEAM_ALLOWLIST.matches(package_name, ecosystem):
            return "allow", f"Package '{package_name}' is team-approved (pre-approved bypass).", ""
        # If team allowlist is non-empty and package not in it, deny.
        return "deny", f"Package '{package_name}' is not on the team-approved list.", ""

    # 1. Check denylist.
    if DENYLIST.matches(package_name, ecosystem):
        return "deny", f"Package '{package_name}' is on the explicit denylist.", ""

    # 2. Check allowlist.
    if ALLOWLIST.matches(package_name, ecosystem):
        return "allow", f"Package '{package_name}' is on the allowlist.", ""

    # 3. Resolve the version that would be installed, and its publication age.
    # Any failure (network, truncated body, unexpected JSON shape) fails closed.
    try:
        name, version, age_days = resolve_package(package_name, ecosystem, kind, value)
    except UnresolvableSpec as e:
        return "ask", (f"Cannot determine which version of '{package_name}' would be installed ({e}). "
                       f"Pin an exact version so it can be checked."), ""
    except urllib.error.HTTPError as e:
        if e.code in (404, 410):
            return "ask", (f"'{package_name}' is not published on the public {ecosystem} registry, so the "
                           f"guard cannot check it. If it is a private package from a configured registry, "
                           f"confirm that source; otherwise check the name for a typo."), ""
        return "deny", (f"Registry lookup failed for '{package_name}' (HTTPError: {e}). "
                        f"Failing closed."), ""
    except Exception as e:  # noqa: BLE001 -- fail closed on anything unexpected
        return "deny", (f"Registry lookup failed for '{package_name}' ({type(e).__name__}: {e}). "
                        f"Failing closed."), ""

    # 4. Check OSV.dev for known vulnerabilities in THAT version.
    try:
        vulns = query_osv(name, ecosystem, version).get("vulns") or []
        vulns = [v for v in vulns if not v.get("withdrawn")]
    except Exception as e:  # noqa: BLE001 -- fail closed on anything unexpected
        return "deny", (f"OSV.dev API check failed for '{name}' ({type(e).__name__}: {e}). "
                        f"Failing closed."), version
    if vulns:
        vuln_ids = [v.get("id", "unknown") for v in vulns]
        malicious = [i for i in vuln_ids if i.startswith("MAL-")]
        if malicious:
            return "deny", (f"Package '{name}@{version}' is flagged as MALICIOUS ({', '.join(malicious[:5])}). "
                            f"Do not install it."), version
        summary = (
            f"Package '{name}@{version}' has {len(vulns)} known vulnerabilities: "
            f"{', '.join(vuln_ids[:5])}"
            + (f" (and {len(vulns) - 5} more)" if len(vulns) > 5 else "")
            + ". Pin a version without these advisories, or choose an alternative package."
        )
        return "deny", summary, version

    # 5. Publication age of the resolved version.
    if age_days is not None and age_days < MIN_AGE_DAYS:
        return "deny", (
            f"Package '{name}@{version}' was published {age_days:.1f} days ago "
            f"(minimum: {MIN_AGE_DAYS} days). New releases are quarantined to "
            f"block supply chain attacks. Wait for the quarantine period to pass, "
            f"or pin an older, established version."
        ), version

    return "allow", f"Package '{name}@{version}' passed all checks.", version


class Job(NamedTuple):
    name: str
    ecosystem: str
    kind: str
    value: str


def validate_all(jobs: list) -> dict:
    """Validate jobs concurrently within the validation budget. Returns
    {job: (decision, reason, version)}. Stops collecting at the first deny
    (the install is refused anyway), and a job still running when the budget
    is spent is reported as a fail-closed deny. Workers are daemon threads,
    so an in-flight request can never hold the process past its decision."""
    results: dict = {}
    if not jobs:
        return results
    todo: queue.Queue = queue.Queue()
    for job in jobs:
        todo.put(job)
    done: queue.Queue = queue.Queue()

    def worker() -> None:
        while True:
            try:
                job = todo.get_nowait()
            except queue.Empty:
                return
            try:
                outcome = validate_package(job.name, job.ecosystem, job.kind, job.value)
            except Exception as e:  # noqa: BLE001 -- fail closed on anything unexpected
                outcome = ("deny", f"Validation of '{job.name}' failed ({type(e).__name__}: {e}). "
                                   f"Failing closed.", "")
            done.put((job, outcome))

    for _ in range(min(_MAX_WORKERS, len(jobs))):
        threading.Thread(target=worker, daemon=True).start()
    while len(results) < len(jobs):
        left = VALIDATION_BUDGET if _deadline is None else _deadline - time.monotonic()
        try:
            job, outcome = done.get(timeout=max(left, 0))
        except queue.Empty:
            for job in jobs:
                results.setdefault(job, (
                    "deny", f"Validation of '{job.name}' did not finish within {VALIDATION_BUDGET:.0f}s "
                            f"(slow network or too many packages in one command). Failing closed; "
                            f"install fewer packages per command.", ""))
            break
        results[job] = outcome
        if outcome[0] == "deny":
            break
    return results


# ---------------------------------------------------------------------------
# Main hook logic
# ---------------------------------------------------------------------------

def _decision(decision: str, reason: str, updated: Optional[str] = None,
              tool_input: Optional[dict] = None) -> None:
    output: dict = {
        "hookEventName": "PreToolUse",
        "permissionDecision": decision,
        "permissionDecisionReason": reason,
    }
    if updated is not None:
        # updatedInput replaces the whole tool input: keep the other fields
        # (Monitor's required description, Bash's timeout and run_in_background).
        output["updatedInput"] = {**(tool_input or {}), "command": updated}
    print(json.dumps({"hookSpecificOutput": output}))
    sys.exit(0)


def _fixed_denials(d: Detection) -> list:
    """Reasons a detection is refused without any registry lookup."""
    if d.manager == _SUSPICIOUS_MANAGER:
        return ["A command segment could not be safely analyzed (unbalanced "
                "quoting or excessively nested shell constructs) and was blocked "
                "to fail closed. Simplify the command, or run the package install "
                "directly, so the guard can validate it."]
    if d.manager in ("nix-env", "nix-profile"):
        return [f"Imperative Nix installs ({d.manager}) bypass flake pinning and are not "
                f"permitted. Use `qsdev devenv add-package <name>` for system packages "
                f"or `qsdev enable <tool>` for ecosystem tools instead."]
    return list(d.denials)


def main() -> None:
    global _deadline
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
    tool_input = input_data.get("tool_input") or {}
    command = tool_input.get("command") or ""

    # 2. Only process tools that run shell commands (a Monitor watching a
    # WebSocket has no command).
    if tool_name not in SHELL_TOOLS or not isinstance(command, str) or not command:
        sys.exit(0)
    if tool_name == "PowerShell":
        # PowerShell uses backslash as a path separator, not an escape.
        command = command.replace("\\", "/")

    # 3. Detect ALL package install commands (handles compound commands).
    detections = detect_install_commands(command)
    if not detections:
        # Not a package install command — allow silently.
        sys.exit(0)

    if CONFIG_ERRORS:
        reason = ("Package guard configuration is invalid: " + "; ".join(CONFIG_ERRORS)
                  + ". Installs are blocked until it is fixed.")
        audit_log({"event": "deny_config", "command": command, "reason": reason})
        _decision("deny", reason)

    # 4. Collect what needs validating. Most restrictive wins across ALL segments.
    _deadline = start_time + VALIDATION_BUDGET
    deny_reasons: list = []
    ask_reasons: list = []
    checked_packages: list = []
    jobs: list = []
    seen: set = set()

    for d in detections:
        fixed = _fixed_denials(d)
        if fixed:
            deny_reasons.extend(fixed)
            audit_log({"event": "deny_" + ("unanalyzable" if d.manager == _SUSPICIOUS_MANAGER else "segment"),
                       "command": command, "segment": d.segment, "manager": d.manager, "reasons": fixed})
        if d.manager in (_SUSPICIOUS_MANAGER, "nix-env", "nix-profile"):
            continue

        ask_reasons.extend(d.issues)

        for specifier in d.packages:
            if specifier.startswith(_UNVALIDATED_PREFIXES):
                ask_reasons.append(f"Package '{specifier}' comes from a registry this guard cannot "
                                   f"check for vulnerabilities or publication age. Confirm it manually.")
                checked_packages.append(specifier)
                continue
            try:
                pkg_name, kind, value, alias = parse_spec(d.ecosystem, d.manager, specifier)
            except UnresolvableSpec as e:
                ask_reasons.append(f"Cannot parse package specifier '{specifier}' ({e}).")
                continue
            if kind == "remove":
                continue
            checked_packages.append(pkg_name)
            if alias and (DENYLIST.matches(alias, d.ecosystem) or TEAM_DENYLIST.matches(alias, d.ecosystem)):
                deny_reasons.append(f"Package alias '{alias}' is on the denylist.")
                continue
            key = (normalize_name(pkg_name, d.ecosystem), d.ecosystem, kind, value)
            if key not in seen:  # a repeated or differently-spelled package is checked once
                seen.add(key)
                jobs.append(Job(pkg_name, d.ecosystem, kind, value))

    # 5. Validate (skipped once the command is refused anyway).
    results = {} if deny_reasons else validate_all(jobs)
    for job, (decision, reason, version) in results.items():
        is_new = None
        if decision == "allow" and NEW_DEP_GATE != "allow":
            in_lock = is_in_lockfile(job.name, job.ecosystem)
            if in_lock is False:
                is_new = True
                gate_msg = (
                    f"New dependency '{job.name}' not found in project lockfile. "
                    f"Add to team allowlist or lockfile first."
                )
                decision, reason = NEW_DEP_GATE, gate_msg
            elif in_lock is True:
                is_new = False

        if decision == "deny":
            deny_reasons.append(reason)
        elif decision == "ask":
            ask_reasons.append(reason)

        soc2_audit_log({
            "package_name": job.name,
            "ecosystem": job.ecosystem,
            "version": version or "",
            "decision": decision,
            "reason": reason,
            "is_new_dependency": is_new,
        })

    rewritten, unrewritable = apply_safety_flags(command, detections)
    deny_reasons.extend(unrewritable)
    elapsed = round(time.monotonic() - start_time, 2)

    # 6. Make final decision. Most restrictive wins across ALL segments.
    if deny_reasons:
        audit_log({"event": "deny", "command": command, "packages": checked_packages,
                   "reasons": deny_reasons, "elapsed_seconds": elapsed})
        _decision("deny", " | ".join(deny_reasons))

    updated = rewritten if rewritten != command else None
    if ask_reasons:
        audit_log({"event": "ask", "command": command, "rewritten": updated,
                   "packages": checked_packages, "reasons": ask_reasons, "elapsed_seconds": elapsed})
        _decision("ask", " | ".join(ask_reasons), updated, tool_input)

    audit_log({"event": "allow", "command": command, "rewritten": updated,
               "packages": checked_packages, "elapsed_seconds": elapsed})
    if updated is not None:
        # updatedInput needs a decision, and `allow` would skip the permission
        # prompt: it would approve every other command in a compound line, a
        # wrapper (`setsid`, `sudo`), an env assignment or a redirection that
        # no ask rule matches, and would approve installs the user's settings
        # would otherwise prompt for. `ask` shows the user the rewritten
        # command instead, and the hook never widens permissions.
        context = (f"Packages validated ({', '.join(checked_packages) or 'lockfile install'}); "
                   f"safety flags inserted: `{command}` -> `{updated}`")
        _decision("ask", context, updated, tool_input)

    # Validated with nothing to change: no decision, the normal permission flow runs.
    sys.exit(0)


if __name__ == "__main__":
    try:
        main()
    except SystemExit:
        raise
    except BaseException as exc:  # noqa: BLE001 -- any internal error must block, not allow
        audit_log({"event": "internal_error", "error": f"{type(exc).__name__}: {exc}"})
        print(f"package-guard internal error ({type(exc).__name__}: {exc}); blocking to fail closed.",
              file=sys.stderr)
        sys.exit(2)
