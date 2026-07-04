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
import shlex
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
# argv-based install detection
# ---------------------------------------------------------------------------
#
# Package names are extracted from the *argv* of a genuine install invocation,
# never from arbitrary substrings of the command line. Earlier revisions matched
# an install pattern anywhere in the raw string and then tokenized the WHOLE
# command, so words belonging to unrelated commands were looked up as if they
# were packages — e.g. `git commit -m "...install..."` or `grep "npm install"
# file` had message/pattern words registry-checked (and a stray word could match
# a real advisory, blocking the command). We now shell-tokenize each command
# segment and require the executable + subcommand verb to actually BE an install
# before extracting any operands. shlex tokenization is the key defence: a quoted
# argument such as "npm install foo" collapses to a single token, so it can never
# be read as an `npm` executable followed by an `install` verb.

# Shell wrappers that may precede the real executable; skipped so the executable
# that follows is treated as argv[0] (e.g. `sudo npm install`, `env FOO=1 pip …`).
COMMAND_PREFIXES: set[str] = {
    "sudo", "doas", "env", "command", "builtin", "exec",
    "time", "nice", "nohup", "stdbuf", "setsid", "ionice", "timeout",
    # Debuggers / tracers / launchers / sandboxes / schedulers that run the
    # command that FOLLOWS them. Without these, `strace npm install evil` hides
    # the install behind an argv[0] the detector never recognises.
    "strace", "ltrace", "catchsegv", "proot", "firejail",
    "flock", "unshare", "chrt", "taskset", "xargs",
}

# Wrapper option flags that consume the following token as their value, so both
# the flag and its value are skipped when locating the real executable
# (e.g. `sudo -u deploy npm install`).
_WRAPPER_VALUE_FLAGS: set[str] = {
    "-u", "--user", "-g", "--group", "-n", "-C", "-h", "--host",
    "-p", "--prompt", "-r", "--role", "-t", "--type", "-U", "-D",
}

# `timeout`'s own value-taking flags, consulted ONLY for the `timeout` wrapper.
# They must NOT be folded into the shared set above: `-s` is a boolean for sudo,
# so treating it as value-consuming for every wrapper would make
# `sudo -s npm install` skip `npm` as if it were -s's value — hiding the install
# behind an argv[0] of `install`.
_TIMEOUT_VALUE_FLAGS: set[str] = {"-k", "--kill-after", "-s", "--signal"}

# Wrappers that take a mandatory positional argument before the command they run
# (e.g. `timeout 10 npm install`): the leading numeric token must be skipped so
# argv[0] resolves to the real executable rather than the duration.
_DURATION_RE = re.compile(r"^[0-9]")

# Wrappers whose FIRST non-flag positional is a resource argument (a lock file /
# fd for `flock`, a CPU mask for `taskset`, a priority for `chrt`) that precedes
# the real command; that one positional is skipped so argv[0] is the executable.
_WRAPPER_LEADING_POSITIONAL: set[str] = {"flock", "taskset", "chrt"}

# A python interpreter (python, python3, python3.12, ...) whose `-m <module>`
# invocation must be resolved to the underlying installer (pip/uv), so that
# `python -m pip install <pkg>` is validated like a bare `pip install <pkg>`.
_PYTHON_RE = re.compile(r"^python[0-9.]*$")

# Shells whose `-c "<script>"` argument is itself a command line that must be
# recursively scanned, so `bash -c "npm install evil"` cannot hide the install
# behind the shell executable. Combined short options (e.g. `bash -lc`) count.
_SHELLS: set[str] = {"sh", "bash", "zsh", "dash", "ash", "ksh", "mksh"}
_SHELL_C_FLAG_RE = re.compile(r"^-[A-Za-z]*c$")

# Privilege launchers whose `-c "<script>"` (or `--command`) argument is a shell
# script, exactly like a shell's -c (e.g. `su -c "npm install evil"`). Unlike a
# shell, the target user may appear as a positional BEFORE -c, so every token is
# scanned for the flag. These are deliberately NOT in COMMAND_PREFIXES: stripping
# them as plain wrappers would swallow the -c script and hide the install.
_PRIV_C_RUNNERS: set[str] = {"su", "runuser"}
# Value-taking option flags for su/runuser exec form (`runuser -u user <cmd>`).
_PRIV_VALUE_FLAGS: set[str] = {
    "-u", "--user", "-g", "--group", "-G", "--supp-group",
    "-s", "--shell", "-w", "--whitelist-environment",
}

# Bound on recursive shell-script scanning (shell -c, eval, su -c, command and
# process substitutions). Exceeding it FAILS CLOSED (a suspicious marker is
# emitted so main() blocks for validation) rather than silently dropping the
# deeper script — a dropped script is a fail-open bypass.
_MAX_SHELL_RECURSION = 6

# Manager label for a segment that could not be safely analyzed (unparseable, or
# nested past the recursion cap). main() denies these to fail closed.
_SUSPICIOUS_MANAGER = "__suspicious__"

# A leading VAR=value environment assignment (e.g. `FOO=bar npm install ...`).
_ENV_ASSIGN_RE = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*=")

# Executable -> list of (verb_tokens, ecosystem, manager_label). verb_tokens are
# the subcommand tokens that must immediately follow the executable to count as
# an install; an empty list means the executable itself is the install (npx).
INSTALL_COMMANDS: dict[str, list[tuple[list[str], str, str]]] = {
    "npm":      [(["install"], "npm", "npm"), (["i"], "npm", "npm"), (["add"], "npm", "npm")],
    "npx":      [([], "npm", "npx")],
    "yarn":     [(["add"], "npm", "yarn"), (["install"], "npm", "yarn")],
    "pnpm":     [(["add"], "npm", "pnpm"), (["install"], "npm", "pnpm"), (["i"], "npm", "pnpm")],
    "bun":      [(["add"], "npm", "bun"), (["install"], "npm", "bun"), (["i"], "npm", "bun")],
    "pip":      [(["install"], "PyPI", "pip")],
    "pip3":     [(["install"], "PyPI", "pip")],
    "uv":       [(["pip", "install"], "PyPI", "uv-pip"), (["add"], "PyPI", "uv-add")],
    "cargo":    [(["add"], "crates.io", "cargo"), (["install"], "crates.io", "cargo")],
    "go":       [(["get"], "Go", "go"), (["install"], "Go", "go")],
    "gem":      [(["install"], "RubyGems", "gem")],
    "composer": [(["require"], "Packagist", "composer")],
}

# Shell operators (and newlines) that separate independent commands. Each
# resulting segment is validated on its own so a compound OR multi-line command
# (including pipe stages and backgrounded commands) cannot smuggle an unchecked
# install past the guard. Two-character operators are listed before their
# one-character prefixes (`&&` before `&`, `||` before `|`, `\r\n` before `\n`)
# so each is consumed whole. Newline and `&` matter: `echo hi\nnpm install evil`
# and `foo & npm install evil` would otherwise be a single unparsed segment.
_SEGMENT_SPLIT_RE = re.compile(r"\s*(?:&&|\|\||;|\||&|\r\n|\n|\r)\s*")

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

def _split_segments(command: str) -> list[str]:
    """Split a (possibly compound or multi-line) command into independent command
    segments on shell operators (&&, ||, ;, |, &) and newlines. Each stage is a
    separate command validated on its own — `echo x | npm install evil` and a
    multi-line `echo hi\\nnpm install evil` both still check the install."""
    return [s for s in _SEGMENT_SPLIT_RE.split(command) if s.strip()]


def _skip_option_flags(tokens: list[str], i: int, value_flags: set[str]) -> int:
    """Advance past the option flags starting at tokens[i]: every token that
    begins with "-" is consumed, and a flag in `value_flags` that does not carry
    its value inline via "=" additionally consumes the following token as its
    value. Returns the index of the first token past the option block."""
    n = len(tokens)
    while i < n and tokens[i].startswith("-"):
        flag = tokens[i]
        i += 1
        if "=" not in flag and flag in value_flags and i < n:
            i += 1  # consume the flag's value too
    return i


def _strip_wrappers(tokens: list[str]) -> list[str]:
    """Return argv from already-tokenized `tokens` with leading environment
    assignments (VAR=val) and exec wrappers (sudo/env/strace/xargs/...) removed,
    so argv[0] is the real executable. Returns [] when nothing remains.

    Only wrappers in COMMAND_PREFIXES are stripped; shells and su/runuser are
    intentionally left in place so their `-c <script>` can be recursed into
    rather than swallowed.
    """
    i, n = 0, len(tokens)
    while i < n:
        tok = tokens[i]
        if _ENV_ASSIGN_RE.match(tok):
            i += 1
            continue
        if tok in COMMAND_PREFIXES:
            # Consult this specific wrapper's value-flag grammar; timeout's value
            # flags apply only to timeout, never to sudo/env/etc.
            value_flags = _TIMEOUT_VALUE_FLAGS if tok == "timeout" else _WRAPPER_VALUE_FLAGS
            # Skip this wrapper's option flags (and their values).
            i = _skip_option_flags(tokens, i + 1, value_flags)
            # Wrappers that take a leading positional before the command:
            # `timeout 10 npm …` (numeric duration), `flock /tmp/l npm …`
            # (lock file/fd), `taskset 0x1 npm …`, `chrt 50 npm …`.
            if i < n:
                if tok == "timeout":
                    if _DURATION_RE.match(tokens[i]):
                        i += 1
                elif tok in _WRAPPER_LEADING_POSITIONAL and not tokens[i].startswith("-"):
                    i += 1
            continue
        break
    return tokens[i:]


def _extract_substitutions(text: str) -> tuple[list[str], str]:
    """Pull command/process substitutions out of `text`, returning
    (inner_scripts, cleaned_text). Each `$(...)`, backtick `` `...` ``, `<(...)`
    and `>(...)` is a shell command in its own right and must be scanned; its span
    is replaced by a space in cleaned_text so the remainder tokenizes normally and
    the substitution's inner operators do not fragment surrounding segments.

    An unbalanced construct is treated as running to end-of-string (fail closed:
    the remainder is still scanned rather than dropped).
    """
    # Fast path: this runs on EVERY hook invocation, and the overwhelmingly
    # common command contains no substitution at all. Without any of the four
    # markers the character scan below cannot extract anything, so skip it.
    if "`" not in text and "$(" not in text and "<(" not in text and ">(" not in text:
        return [], text
    scripts: list[str] = []
    out: list[str] = []
    i, n = 0, len(text)
    while i < n:
        c = text[i]
        if c == "`":
            j = text.find("`", i + 1)
            if j == -1:
                scripts.append(text[i + 1:])
                out.append(" ")
                i = n
            else:
                scripts.append(text[i + 1:j])
                out.append(" ")
                i = j + 1
            continue
        if (c == "$" and i + 1 < n and text[i + 1] == "(") or (
            c in "<>" and i + 1 < n and text[i + 1] == "("
        ):
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
            if depth == 0:
                scripts.append(text[start:j - 1])
                i = j
            else:
                scripts.append(text[start:])
                i = n
            out.append(" ")
            continue
        out.append(c)
        i += 1
    return scripts, "".join(out)


def _match_verb(args: list[str], verb_tokens: list[str]) -> Optional[list[str]]:
    """If args begins with verb_tokens, return the remaining operands; else None."""
    if len(args) < len(verb_tokens):
        return None
    for idx, verb in enumerate(verb_tokens):
        if args[idx] != verb:
            return None
    return args[len(verb_tokens):]


def _extract_package_args(operands: list[str], manager: str) -> list[str]:
    """From the operands that follow an install verb, return the package
    specifiers, skipping flags (and their values) and local path arguments."""
    packages: list[str] = []
    skip_next = False
    for tok in operands:
        if skip_next:
            skip_next = False
            continue
        if tok.startswith("-"):
            base = tok.split("=", 1)[0]
            if "=" not in tok and (tok in FLAGS_WITH_ARGS or base in FLAGS_WITH_ARGS):
                skip_next = True
            continue
        if tok in (".", "./", ".."):
            continue
        packages.append(tok)
    return packages


def _c_runner_script(argv: list[str]) -> Optional[str]:
    """If argv invokes a `-c` runner — a shell (`bash -c "<script>"`) or a
    privilege launcher (`su -c "<script>"`, `runuser -c "<script>"`) — return the
    script argument so it can be recursively scanned; else None.

    Shells: honour combined short options (`bash -lc "<script>"`) and bail at the
    first positional before any -c (a script FILE such as `bash x.sh` is not -c).
    su/runuser: the target user may appear as a positional before -c
    (`su deploy -c "<script>"`), so every token is scanned for -c/--command.
    """
    if not argv:
        return None
    exe = os.path.basename(argv[0])
    if exe in _SHELLS:
        for i in range(1, len(argv)):
            tok = argv[i]
            if _SHELL_C_FLAG_RE.match(tok):
                return argv[i + 1] if i + 1 < len(argv) else None
            if not tok.startswith("-"):
                return None  # first positional before any -c: not a -c invocation
        return None
    if exe in _PRIV_C_RUNNERS:
        for i in range(1, len(argv)):
            tok = argv[i]
            # A bare `-c` already matches _SHELL_C_FLAG_RE; only the long form
            # `--command` needs listing separately.
            if tok == "--command" or _SHELL_C_FLAG_RE.match(tok):
                return argv[i + 1] if i + 1 < len(argv) else None
        return None
    return None


def _strip_priv_runner(argv: list[str]) -> list[str]:
    """For a su/runuser invocation WITHOUT -c (exec form, e.g.
    `runuser -u deploy npm install evil`): drop the launcher and its option flags
    (consuming values for flags like -u/-g/-s) so the wrapped command surfaces as
    argv[0]. Returns [] when nothing remains."""
    return argv[_skip_option_flags(argv, 1, _PRIV_VALUE_FLAGS):]


def _suspicious_detection(segment: str) -> tuple[str, str, str, list[str]]:
    """A fail-closed detection marker for a segment that could not be safely
    analyzed (unparseable, or nested past the recursion cap). main() denies these
    so the guard never silently allows an install it could not inspect."""
    return ("suspicious", _SUSPICIOUS_MANAGER, segment[:200], [])


def parse_install_argv(argv: list[str]) -> Optional[tuple[str, str, list[str]]]:
    """Parse an already-tokenized argv (wrappers/env-assignments stripped by
    _strip_wrappers). Return (ecosystem, manager_label, packages) when it is a
    genuine package-install invocation, else None. Detection is argv-based: the
    executable and its subcommand verb must actually be an install command, so
    install-like words inside unrelated commands are never treated as packages.
    Resolves `python -m pip|uv` module invocations to the underlying installer
    before matching."""
    if not argv:
        return None
    exe = os.path.basename(argv[0])
    rest = argv[1:]

    # `python -m pip install …` / `python3 -m uv pip install …`: the real
    # installer is the module named after -m, not the interpreter. Resolve it so
    # the invocation is validated like a bare `pip`/`uv` install.
    if _PYTHON_RE.match(exe) and rest and rest[0] == "-m":
        if len(rest) < 2:
            return None
        exe = os.path.basename(rest[1])
        rest = rest[2:]

    # Imperative Nix installs are matched specially (denied downstream) and carry
    # no registry package operands.
    if exe == "nix-env":
        if any(a == "--install" or a.startswith("-i") for a in rest):
            return ("nix", "nix-env", [])
        return None
    if exe == "nix":
        if _match_verb(rest, ["profile", "install"]) is not None:
            return ("nix", "nix-profile", [])
        return None

    for verb_tokens, ecosystem, manager in INSTALL_COMMANDS.get(exe, []):
        operands = _match_verb(rest, verb_tokens)
        if operands is not None:
            return (ecosystem, manager, _extract_package_args(operands, manager))
    return None


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

def _classified_exe(exe: str) -> bool:
    """True when `exe` is an executable the classifier already understands — a
    catalog package manager, a nix special case, a shell, or a python
    interpreter. The catalog wrapper fallback must not second-guess these: a
    known manager with a non-install verb (`go build ./...`) really is not an
    install, and shells/pythons have their own dedicated handling."""
    return (
        exe in INSTALL_COMMANDS
        or exe in ("nix", "nix-env")
        or exe in _SHELLS
        or bool(_PYTHON_RE.match(exe))
    )


def _embedded_install(argv: list[str]) -> Optional[tuple[str, str, list[str]]]:
    """Parse of the first genuine install invocation embedded past argv[0]: the
    token must itself BE an INSTALL_COMMANDS manager whose tail parse_install_argv
    classifies as an install — one classifier for the primary path and this
    fallback, so the two can never disagree. Purely catalog-driven — no wrapper
    names are consulted, so wrappers unknown to COMMAND_PREFIXES (setpriv,
    nsenter, systemd-run, ...) are covered by construction. shlex has already
    collapsed quoted text into single tokens, so `grep -rn "npm install" .`
    carries no bare `npm` token and can never match. Returns None when argv
    embeds no install."""
    for k in range(1, len(argv)):
        if argv[k] in INSTALL_COMMANDS:
            parsed = parse_install_argv(argv[k:])
            if parsed is not None:
                return parsed
    return None


def _recurse_script(
    script: str, depth: int, results: list[tuple[str, str, str, list[str]]]
) -> None:
    """Recursively scan a nested shell script (a shell/su/eval -c body or a
    command/process substitution), appending its detections. FAILS CLOSED past
    the recursion cap: a suspicious marker is emitted instead of silently dropping
    the script, so a pathologically nested install still forces validation."""
    script = script.strip()
    if not script:
        return
    if depth >= _MAX_SHELL_RECURSION:
        results.append(_suspicious_detection(script))
        return
    results.extend(detect_install_commands(script, depth + 1))


def detect_install_commands(command: str, _depth: int = 0) -> list[tuple[str, str, str, list[str]]]:
    """
    Find ALL genuine package install invocations in a (possibly compound or
    multi-line) command. Returns a list of (ecosystem, manager_label, segment,
    packages) tuples — one per segment whose argv is actually an install command,
    with the raw package specifiers extracted from that segment's single parse.
    Every segment is parsed independently so that compound commands like
    ``pip install safe && npm install evil`` cannot sneak an unchecked install
    past the guard, while unrelated commands whose text merely *mentions* an
    install (``git commit -m "add install docs"``, ``grep "npm install" file``)
    are correctly ignored.

    Classification only ever happens at a command POSITION (argv[0] after wrappers
    are stripped and after each shell construct is entered recursively), never by
    matching install words inside plain string arguments — that is what keeps the
    false-positive suite safe. Installs hidden behind a shell construct are
    surfaced by recursing into it:
      - command / process substitutions: ``echo $(npm install evil)``,
        ``x=`npm install evil` ``, ``diff <(pip install evil) x``
      - shell / privilege -c runners: ``bash -c``, ``sh -c``, ``su -c``,
        ``runuser -c``
      - ``eval "<script>"`` (its concatenated arguments are a shell script)
    Recursion is bounded by _MAX_SHELL_RECURSION; exceeding it, or hitting an
    unparseable segment, FAILS CLOSED via a suspicious marker rather than dropping
    the script.

    One catalog-driven fallback relaxes the argv[0] rule for TOP-LEVEL segments
    only: when argv[0] matches nothing known, an INSTALL_COMMANDS manager token
    immediately followed by one of its install verbs later in argv is
    re-classified from that token, so exec wrappers absent from
    COMMAND_PREFIXES (``setpriv npm install evil``) cannot fail open. Quoted
    text is immune — shlex keeps it a single token.
    """
    results: list[tuple[str, str, str, list[str]]] = []

    # Command/process substitutions anywhere in the command are independent
    # scripts. Extract and recurse into them first, then continue with a cleaned
    # command whose substitution spans are blanked out (so their inner operators
    # can't fragment the surrounding segments).
    sub_scripts, command = _extract_substitutions(command)
    for script in sub_scripts:
        _recurse_script(script, _depth, results)

    for segment in _split_segments(command):
        segment = segment.strip()
        if not segment:
            continue

        try:
            tokens = shlex.split(segment)
        except ValueError:
            # Unbalanced quotes etc.: we cannot trust any tokenization, so we
            # cannot rule out a hidden install. Fail closed instead of guessing.
            results.append(_suspicious_detection(segment))
            continue

        argv = _strip_wrappers(tokens)
        if not argv:
            continue
        exe = os.path.basename(argv[0])

        # `eval "<script>"` / `eval pip install evil`: eval concatenates its
        # arguments into a shell script and runs it — recurse into that script.
        if exe == "eval":
            _recurse_script(" ".join(argv[1:]), _depth, results)
            continue

        # `bash -c "<script>"`, `su -c "<script>"`, `runuser -c "<script>"`: the
        # install lives inside the -c script, not behind the runner executable.
        script = _c_runner_script(argv)
        if script is not None:
            _recurse_script(script, _depth, results)
            continue

        # su/runuser exec form without -c (`runuser -u deploy npm install evil`):
        # drop the launcher so the wrapped command can be classified.
        if exe in _PRIV_C_RUNNERS:
            argv = _strip_priv_runner(argv)
            if not argv:
                continue
            exe = os.path.basename(argv[0])

        parsed = parse_install_argv(argv)
        if parsed is None and _depth == 0 and not _classified_exe(exe):
            # Catalog-driven wrapper fallback. argv[0] failed every
            # classification above, so this segment was about to be dropped —
            # exactly how an exec-style wrapper missing from COMMAND_PREFIXES
            # (setpriv, nsenter, systemd-run, ...) used to smuggle an install
            # through fail-open. If a catalog manager token whose tail
            # classifies as one of its install verbs appears later in argv,
            # re-classify from that token so the real package specifiers are
            # extracted and validated. Top level only: inside recursed
            # -c/eval/substitution scripts a manager-verb pair is routinely
            # inert data (`bash -c "echo pip install docs"` is pinned as
            # must-allow by the false-positive suite), while the explicit
            # classifications above still run at every depth.
            parsed = _embedded_install(argv)
        if parsed is not None:
            ecosystem, manager, packages = parsed
            results.append((ecosystem, manager, segment, packages))

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

    for ecosystem, manager, segment, packages in detections:
        # Fail-closed marker: a segment that could not be safely analyzed
        # (unparseable, or nested past the recursion cap). Deny so an install we
        # could not inspect is never silently allowed.
        if manager == _SUSPICIOUS_MANAGER:
            reason = (
                "A command segment could not be safely analyzed (unbalanced "
                "quoting or excessively nested shell constructs) and was blocked "
                "to fail closed. Simplify the command, or run the package install "
                "directly, so the guard can validate it."
            )
            deny_reasons.append(reason)
            audit_log({
                "event": "deny_unanalyzable",
                "command": command,
                "segment": segment,
                "reason": reason,
            })
            continue

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

        # `packages` came from this segment's parse (not the full compound command).
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
