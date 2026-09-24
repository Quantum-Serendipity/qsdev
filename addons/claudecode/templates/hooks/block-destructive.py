#!/usr/bin/env python3
"""
Claude Code PreToolUse Hook: Destructive Operation Prevention

Blocks dangerous shell commands across six categories: filesystem destruction,
git force operations, database destruction, remote code execution, consulting
cross-environment protection, and infrastructure destruction.

Inspects every tool that runs a shell command (SHELL_TOOLS: Bash, PowerShell
and Monitor), matching the hook's settings.json matcher.

Complements static deny rules in settings.json with dynamic, context-aware
checks. Commands are tokenized (quotes removed, wrappers such as sudo/env and
git's global options skipped) and each check inspects the argv of the command
it is about, so a quoted argument (a commit message, a grep pattern) is never
mistaken for the command itself, while `bash -c`, `eval`, `$(...)`, backticks
and process substitution are parsed as the nested scripts they are.

Exit codes:
  0 — allow or deny (with JSON on stdout for deny)
  2 — hook error (fail-closed)

Configuration via environment variables:
  DESTRUCTIVE_PREVENTION_PROTECTED_BRANCHES — comma-separated (default: main,master,production,release)
  DESTRUCTIVE_PREVENTION_PRODUCTION_HOSTS   — comma-separated host name parts (default: prod,production,live,staging)
"""

# Annotations stay unevaluated so the PEP 604 / PEP 585 forms below also load
# on Python 3.9 (e.g. macOS's /usr/bin/python3); evaluating them there raised
# TypeError at import, which exited 1 and failed open.
from __future__ import annotations

import sys


def _fail_closed(exc_type, exc, _tb) -> None:
    """Uncaught exceptions anywhere, including at import time, exit 2 so
    Claude Code blocks the tool call instead of treating exit 1 as a
    non-blocking error."""
    print(f"destructive prevention error: {exc_type.__name__}: {exc}", file=sys.stderr)
    sys.stderr.flush()
    import os as _os
    _os._exit(2)


sys.excepthook = _fail_closed

import hashlib  # noqa: E402
import json  # noqa: E402
import os  # noqa: E402
import posixpath  # noqa: E402
import re  # noqa: E402
import shlex  # noqa: E402
import subprocess  # noqa: E402
from datetime import datetime, timezone  # noqa: E402
from pathlib import Path  # noqa: E402

# Tools whose tool_input.command runs in a shell. The hook's settings.json
# matcher must list exactly these tools (hook_registry.go shellToolMatcher;
# kept in sync by TestHookMatchersCoverScriptTools).
SHELL_TOOLS: tuple[str, ...] = ("Bash", "PowerShell", "Monitor")

PROTECTED_BRANCHES: list[str] = [
    b.strip()
    for b in os.environ.get(
        "DESTRUCTIVE_PREVENTION_PROTECTED_BRANCHES",
        "main,master,production,release",
    ).split(",")
    if b.strip()
]

PRODUCTION_HOST_PATTERNS: list[str] = [
    p.strip().lower()
    for p in os.environ.get(
        "DESTRUCTIVE_PREVENTION_PRODUCTION_HOSTS",
        "prod,production,live,staging",
    ).split(",")
    if p.strip()
]

# Environment names that mark a deployment target as production.
DEPLOY_ENVIRONMENTS: frozenset[str] = frozenset({"prod", "production", "live"})

AUDIT_LOG: Path = Path(
    os.environ.get("CLAUDE_PROJECT_DIR", ".")
) / ".claude" / "logs" / "hook-audit.jsonl"
AUDIT_LOG_MAX_BYTES = 10 * 1024 * 1024


def audit_log(entry: dict) -> None:
    """Append a JSON entry to the audit log. Never raises. The file is created
    0600 and rotated to <name>.1 once it exceeds AUDIT_LOG_MAX_BYTES; entries
    carry command fingerprints, never command text (which can hold secrets)."""
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


def command_fingerprint(command: str) -> dict:
    """Identify a command in the audit log without recording its text."""
    return {
        "command_sha256": hashlib.sha256(command.encode("utf-8", "surrogateescape")).hexdigest(),
        "command_len": len(command),
    }


def deny(category: str, reason: str, remediation: str, command: str) -> None:
    """Output structured deny JSON and exit."""
    audit_log({
        "event": "deny",
        "hook": "destructive-prevention",
        "category": category,
        "reason": reason,
        **command_fingerprint(command),
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
# Command parsing
# ---------------------------------------------------------------------------

_PUNCT = ";&|()`\n<>"
_TWO_CHAR_OPS = ("&&", "||", ";;", "|&", "<<", ">>", ">&", "<&", "&>", ">|", "<>")

# How deep nested scripts (bash -c, eval, $(...), backticks) are re-parsed.
_MAX_NESTING = 4


class Cmd:
    """One simple command: its argv (quotes removed), its redirections, and
    how it is connected to its neighbours."""

    __slots__ = ("argv", "redirects", "pipe_in", "pipeline", "parent")

    def __init__(self, pipeline: int, pipe_in: bool = False, parent: Cmd | None = None):
        self.argv: list[str] = []
        self.redirects: list[tuple[str, str]] = []
        self.pipe_in = pipe_in          # stdin comes from the previous command
        self.pipeline = pipeline        # commands joined by `|` share an id
        self.parent = parent            # command whose argument this one is


class _Parser:
    def __init__(self) -> None:
        self.cmds: list[Cmd] = []
        self.next_pipeline = 0

    def _new_pipeline(self) -> int:
        self.next_pipeline += 1
        return self.next_pipeline

    def parse(self, text: str, depth: int, parent: Cmd | None) -> None:
        tokens = _tokenize(text)
        start = len(self.cmds)
        cur = Cmd(self._new_pipeline(), parent=parent)
        # Open substitutions/groups: (kind, command to resume after closing).
        stack: list[tuple[str, Cmd | None]] = []
        pending_redirect: str | None = None

        def finish(c: Cmd) -> None:
            if c.argv or c.redirects:
                self.cmds.append(c)

        for tok, is_op in tokens:
            if not is_op:
                if pending_redirect is not None:
                    cur.redirects.append((pending_redirect, tok))
                    pending_redirect = None
                else:
                    cur.argv.append(tok)
                continue
            pending_redirect = None
            i = 0
            while i < len(tok):
                two = tok[i:i + 2]
                c = tok[i]
                if tok[i:i + 3] == "<<<":
                    pending_redirect = "<<<"
                    i += 3
                    continue
                if two in ("<(", ">("):
                    # Process substitution: an argument of the current command.
                    stack.append(("subst", cur))
                    cur = Cmd(self._new_pipeline(), parent=cur)
                    i += 2
                    continue
                if two in _TWO_CHAR_OPS:
                    if two == "|&":
                        finish(cur)
                        cur = Cmd(cur.pipeline, pipe_in=True, parent=cur.parent)
                    elif two in ("&&", "||", ";;"):
                        finish(cur)
                        cur = Cmd(self._new_pipeline(), parent=cur.parent)
                    else:
                        pending_redirect = two
                    i += 2
                    continue
                if c == "|":
                    finish(cur)
                    cur = Cmd(cur.pipeline, pipe_in=True, parent=cur.parent)
                elif c in ";&\n":
                    finish(cur)
                    cur = Cmd(self._new_pipeline(), parent=cur.parent)
                elif c == "(":
                    if cur.argv and cur.argv[-1].endswith("$"):
                        # `$(`: command substitution, an argument of cur.
                        cur.argv[-1] = cur.argv[-1][:-1]
                        if not cur.argv[-1]:
                            cur.argv.pop()
                        stack.append(("subst", cur))
                        cur = Cmd(self._new_pipeline(), parent=cur)
                    elif cur.argv:
                        # `(` in argument position: a PowerShell subexpression
                        # (a syntax error in POSIX shells), an argument of cur.
                        stack.append(("subst", cur))
                        cur = Cmd(self._new_pipeline(), parent=cur)
                    else:
                        stack.append(("group", None))
                        finish(cur)
                        cur = Cmd(self._new_pipeline(), parent=cur.parent)
                elif c == ")":
                    finish(cur)
                    kind, resume = stack.pop() if stack else ("group", None)
                    if kind == "subst" and resume is not None:
                        cur = resume
                    else:
                        cur = Cmd(self._new_pipeline(), parent=cur.parent)
                elif c == "`":
                    if stack and stack[-1][0] == "backtick":
                        finish(cur)
                        _, resume = stack.pop()
                        cur = resume if resume is not None else Cmd(self._new_pipeline())
                    else:
                        stack.append(("backtick", cur))
                        cur = Cmd(self._new_pipeline(), parent=cur)
                elif c in "<>":
                    pending_redirect = c
                i += 1
        finish(cur)
        for _, resume in reversed(stack):
            if resume is not None and resume is not cur:
                finish(resume)

        if depth >= _MAX_NESTING:
            return
        # Nested scripts: quoted command substitutions and the script
        # arguments of shells, eval and su are parsed as commands too.
        for cmd in list(self.cmds[start:]):
            for inner in _nested_scripts(cmd):
                self.parse(inner, depth + 1, cmd)


def _tokenize(text: str) -> list[tuple[str, bool]]:
    """Split text into (token, is_operator) pairs with quotes removed. Input
    that shlex cannot parse (unbalanced quotes) is retried with the quote
    characters blanked, so a dangerous command is still recognised."""
    for candidate in (text, re.sub(r"[\"']", " ", text)):
        try:
            lex = shlex.shlex(candidate, posix=True, punctuation_chars=_PUNCT)
            lex.whitespace = " \t\r"
            lex.whitespace_split = True
            return [(t, bool(t) and all(ch in _PUNCT for ch in t)) for t in lex]
        except ValueError:
            continue
    return [(t, False) for t in re.split(r"[\s\"'\\]+", text) if t]


def _command_substitutions(token: str) -> list[str]:
    """Return the scripts inside `$(...)` and backticks within one token (a
    quoted argument keeps them intact after tokenizing)."""
    out: list[str] = []
    i = token.find("$(")
    while i >= 0:
        depth, j = 1, i + 2
        while j < len(token) and depth:
            depth += {"(": 1, ")": -1}.get(token[j], 0)
            j += 1
        out.append(token[i + 2:j - 1] if depth == 0 else token[i + 2:])
        i = token.find("$(", j)
    parts = token.split("`")
    out.extend(parts[k] for k in range(1, len(parts), 2))
    return [s for s in out if s.strip()]


SHELLS: frozenset[str] = frozenset({
    "sh", "bash", "zsh", "dash", "ksh", "mksh", "ash", "fish", "csh", "tcsh",
})
POWERSHELLS: frozenset[str] = frozenset({"pwsh", "powershell", "powershell.exe", "pwsh.exe"})


def _nested_scripts(cmd: Cmd) -> list[str]:
    """Scripts a command runs from its arguments."""
    scripts: list[str] = []
    for tok in cmd.argv:
        scripts.extend(_command_substitutions(tok))
    argv = effective_argv(cmd.argv)
    if not argv:
        return scripts
    name = prog(argv[0])
    args = argv[1:]
    if name in SHELLS or name in POWERSHELLS:
        for k, a in enumerate(args):
            is_c = (
                a.startswith("-") and not a.startswith("--") and "c" in a[1:]
                if name in SHELLS
                else a.lower() in ("-c", "-command", "-com")
            )
            if is_c and k + 1 < len(args):
                scripts.append(args[k + 1])
                break
    elif name == "eval" and args:
        scripts.append(" ".join(args))
    elif name in ("su", "runuser"):
        for k, a in enumerate(args):
            if a in ("-c", "--command") and k + 1 < len(args):
                scripts.append(args[k + 1])
            elif a.startswith("--command="):
                scripts.append(a.split("=", 1)[1])
    elif name in ("watch",):
        positional = [a for a in args if not a.startswith("-")]
        if positional:
            scripts.append(" ".join(positional))
    return scripts


def parse_commands(command: str) -> list[Cmd]:
    parser = _Parser()
    parser.parse(command, 0, None)
    return parser.cmds


def prog(token: str) -> str:
    """The program name a command word runs: its basename, lower-cased."""
    return posixpath.basename(token.replace("\\", "/")).lower()


_ASSIGNMENT = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*=")

# Commands that run their arguments as a new command, with the options of
# each that consume a separate value.
_WRAPPERS: dict[str, frozenset[str]] = {
    "sudo": frozenset({"-u", "-g", "-C", "-D", "-h", "-p", "-r", "-t", "-U", "-T", "--user", "--group"}),
    "doas": frozenset({"-u", "-C"}),
    "env": frozenset({"-u", "-C", "--unset", "--chdir"}),
    "nohup": frozenset(),
    "time": frozenset({"-f", "-o", "--format", "--output"}),
    "command": frozenset(),
    "builtin": frozenset(),
    "exec": frozenset({"-a"}),
    "nice": frozenset({"-n", "--adjustment"}),
    "ionice": frozenset({"-c", "-n", "-p", "--class", "--classdata"}),
    "timeout": frozenset({"-s", "-k", "--signal", "--kill-after"}),
    "stdbuf": frozenset({"-i", "-o", "-e"}),
    "setsid": frozenset(),
    "unbuffer": frozenset(),
    "busybox": frozenset(),
    "xargs": frozenset({"-I", "-n", "-L", "-P", "-d", "-E", "-s", "-a", "--max-args",
                        "--max-lines", "--max-procs", "--delimiter", "--arg-file"}),
}


def effective_argv(argv: list[str]) -> list[str]:
    """Strip leading variable assignments and exec-style wrappers (sudo, env,
    nohup, timeout, xargs, ...) so argv[0] is the program that really runs."""
    i = 0
    while i < len(argv):
        if _ASSIGNMENT.match(argv[i]):
            i += 1
            continue
        name = prog(argv[i])
        if name not in _WRAPPERS:
            break
        value_opts = _WRAPPERS[name]
        i += 1
        while i < len(argv) and argv[i].startswith("-") and argv[i] != "-":
            opt = argv[i]
            i += 1
            if opt == "--":
                break
            if opt in value_opts:
                i += 1
        if name == "timeout" and i < len(argv):
            i += 1  # the duration
    return argv[i:]


# ---------------------------------------------------------------------------
# Path resolution
# ---------------------------------------------------------------------------

def _slash(p: str) -> str:
    return p.replace("\\", "/")


# The shell's home: $HOME, which is what ~ and $HOME expand to in Bash, Git
# Bash on Windows included. Python's expanduser ignores HOME on Windows, so it
# is only the fallback. PowerShell's $env:USERPROFILE is read separately, and
# every spelling of home counts as a critical path.
HOME: str = _slash(os.environ.get("HOME") or os.path.expanduser("~"))
_PROFILE: str = _slash(os.environ.get("USERPROFILE") or HOME)
_HOME_SPELLINGS: tuple[tuple[str, str], ...] = (
    ("${HOME}", HOME), ("$HOME", HOME), ("$env:USERPROFILE", _PROFILE),
    ("$env:HOME", HOME), ("~", HOME),
)
_DRIVE = re.compile(r"^[A-Za-z]:(/|$)")


def _normpath(p: str) -> str:
    norm = posixpath.normpath(p)
    if norm.startswith("//"):
        norm = "/" + norm.lstrip("/")  # normpath keeps a POSIX leading "//"
    return norm


def _anchor(p: str) -> str:
    """A normalised absolute path with any drive letter dropped, the form
    resolve_path returns, so C:/Users/me compares equal to a resolved ~."""
    p = _slash(p)
    if _DRIVE.match(p):
        p = p[2:] or "/"
    return _normpath(p)


def _absolute(path: str, cwd: str) -> str | None:
    """A shell word naming a path as an absolute, slash-separated path that
    keeps any drive letter: ~, $HOME and ${HOME} are expanded, trailing `/*`
    globs dropped (`dir/*` reaches everything `dir` holds), and relative paths
    joined to cwd. None when the word still holds an unexpanded variable."""
    p = _slash(path)
    for spelling, home in _HOME_SPELLINGS:
        if p == spelling or p.startswith(spelling + "/"):
            p = home + p[len(spelling):]
            break
    while p.endswith("/*"):
        p = p[:-2] or "/"
    if p == "*":
        p = "."
    if not p or "$" in p or p.startswith("~"):
        return None
    if not _DRIVE.match(p) and not p.startswith("/"):
        p = posixpath.join(_slash(cwd), p)
    return p


def resolve_path(path: str, cwd: str) -> str | None:
    """Resolve a shell word naming a path (see _absolute) to the normalised,
    drive-less form paths are compared in. None when it cannot be resolved."""
    p = _absolute(path, cwd)
    return None if p is None else _anchor(p)  # a drive root compares like "/"


def resolve_dir(path: str, cwd: str) -> str | None:
    """Like resolve_path, but keeps the drive letter: for a directory the hook
    itself opens (the repository `git -C` or `cd` moves to), where a drive-less
    path would resolve against this process's current drive on Windows."""
    p = _absolute(path, cwd)
    if p is None:
        return None
    drive = p[:2] if _DRIVE.match(p) else ""
    return drive + _anchor(p)


def _covers(ancestor: str, path: str) -> bool:
    """Whether ancestor is path itself or one of its parent directories."""
    if os.name == "nt":
        ancestor, path = ancestor.lower(), path.lower()
    return path == ancestor or path.startswith(ancestor.rstrip("/") + "/")


# Every spelling of the home directory: the shell's $HOME, Windows'
# USERPROFILE and the platform default, which differ on Windows.
_HOMES: frozenset[str] = frozenset(
    _anchor(h) for h in (HOME, _PROFILE, _slash(os.path.expanduser("~")))
)


def is_critical_path(resolved: str, project: str, include_project: bool) -> bool:
    """A recursive delete of resolved would take out the filesystem root, the
    home directory (or a directory above it), or, when include_project, the
    project root (or a directory above it)."""
    if resolved == "/" or any(_covers(resolved, h) for h in _HOMES):
        return True
    return include_project and bool(project) and _covers(resolved, _anchor(project))


def _is_temp(resolved: str) -> bool:
    roots = {"/tmp", _slash(os.path.realpath("/tmp"))}
    tmpdir = os.environ.get("TMPDIR", "")
    if tmpdir:
        roots.add(_normpath(_slash(tmpdir)))
    return any(_covers(r, resolved) for r in roots)


_CHDIR = frozenset({"cd", "pushd", "chdir", "set-location", "sl"})


def working_dirs(cmds: list[Cmd], cwd: str) -> list[str]:
    """The directory each command runs in: cwd, as changed by any earlier
    top-level `cd` (so `cd .. && rm -rf project` resolves against the parent).
    A cd whose target cannot be resolved (`cd "$DIR"`) keeps the last known
    directory."""
    dirs: list[str] = []
    cur = cwd
    for cmd in cmds:
        dirs.append(cur)
        argv = effective_argv(cmd.argv)
        if cmd.parent is not None or not argv or prog(argv[0]) not in _CHDIR:
            continue
        operands = [a for a in argv[1:] if not a.startswith("-")]
        if argv[1:2] == ["-"]:
            continue  # `cd -`: the previous directory, unknown here
        target = resolve_dir(operands[0] if operands else "~", cur)
        if target is not None:
            cur = target
    return dirs


# ---------------------------------------------------------------------------
# Category 1: Filesystem destruction
# ---------------------------------------------------------------------------

_FORK_BOMB = re.compile(r':\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:')
_BLOCK_DEVICE = re.compile(
    r"^/dev/(sd[a-z]|hd[a-z]|vd[a-z]|xvd[a-z]|nvme\d|mmcblk\d|disk\d|rdisk\d|md\d|dm-\d|loop\d|mapper/)"
)


def _short_flags(args: list[str]) -> set[str]:
    """The letters of every short-option cluster (`-rf` -> {r, f})."""
    flags: set[str] = set()
    for a in args:
        if a == "--":
            break
        if a.startswith("-") and not a.startswith("--") and len(a) > 1:
            flags.update(a[1:])
    return flags


def _rm_targets(args: list[str]) -> tuple[bool, list[str]]:
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
    return recursive, targets


_PS_REMOVE = frozenset({"remove-item", "ri", "del", "erase", "rd", "rmdir"})


def _ps_remove_targets(args: list[str]) -> tuple[bool, list[str]]:
    """Remove-Item: -Recurse may be abbreviated (-r, -rec); targets are the
    positional arguments and the values of -Path/-LiteralPath."""
    recursive = False
    targets: list[str] = []
    k = 0
    while k < len(args):
        a = args[k]
        low = a.lower()
        if low.startswith("-"):
            name = low[1:].split(":", 1)[0]
            if name and "recurse".startswith(name):
                recursive = True
            elif name in ("path", "literalpath", "lp", "pspath"):
                if k + 1 < len(args):
                    targets.append(args[k + 1])
                    k += 1
        else:
            targets.extend(t for t in a.split(",") if t)
        k += 1
    return recursive, targets


def _find_deletes(args: list[str]) -> tuple[bool, list[str]]:
    """Whether find deletes what it matches, and its starting points."""
    starts: list[str] = []
    k = 0
    while k < len(args) and args[k] in ("-H", "-L", "-P", "-O0", "-O1", "-O2", "-O3"):
        k += 1
    while k < len(args) and not args[k].startswith(("-", "(", "!")):
        starts.append(args[k])
        k += 1
    expr = args[k:]
    deletes = "-delete" in expr
    for j, a in enumerate(expr):
        if a in ("-exec", "-execdir", "-ok", "-okdir") and j + 1 < len(expr):
            inner = effective_argv(expr[j + 1:])
            if inner and prog(inner[0]) in ("rm", "shred", "unlink", "rmdir"):
                deletes = True
    return deletes, starts or ["."]


def _rsync_delete_target(args: list[str]) -> str | None:
    """The destination of an rsync that deletes extraneous files, else None."""
    if not any(a == "--del" or a.startswith("--delete") for a in args):
        return None
    positional = [a for a in args if not a.startswith("-")]
    return positional[-1] if len(positional) >= 2 else None


def check_filesystem(cmds: list[Cmd], cwd: str, project: str) -> tuple[str, str] | None:
    whole_tree = (
        "Recursive deletion of root/home directory or the project tree detected.",
        "Use targeted rm on specific files or directories within the project.",
    )
    for cmd, cwd in zip(cmds, working_dirs(cmds, cwd)):
        for op, target in cmd.redirects:
            if ">" in op and _BLOCK_DEVICE.match(target):
                return (
                    "Direct device write via redirection detected.",
                    "Do not write directly to block devices.",
                )
        argv = effective_argv(cmd.argv)
        if not argv:
            continue
        name, args = prog(argv[0]), argv[1:]
        if name == "dd" and any(
            a.startswith("of=") and _BLOCK_DEVICE.match(a[3:]) for a in args
        ):
            return (
                "Direct device write detected.",
                "Do not write directly to block devices.",
            )
        if name.startswith("mkfs") or name in ("mke2fs", "wipefs"):
            return (
                "Filesystem creation on device detected.",
                "Do not format devices from within a project.",
            )
        if name == "rm" or name in _PS_REMOVE:
            recursive, targets = (
                _rm_targets(args) if name == "rm" else _ps_remove_targets(args)
            )
            if recursive and any(
                (r := resolve_path(t, cwd)) is not None and is_critical_path(r, project, True)
                for t in targets
            ):
                return whole_tree
        if name == "find":
            deletes, starts = _find_deletes(args)
            if deletes and any(
                (r := resolve_path(s, cwd)) is not None and is_critical_path(r, project, False)
                for s in starts
            ):
                return (
                    "find deleting from the filesystem root or home directory detected.",
                    "Run find with -delete only on specific directories within the project.",
                )
        if name == "rsync":
            dest = _rsync_delete_target(args)
            r = resolve_path(dest, cwd) if dest and ":" not in dest.split("/", 1)[0] else None
            if r is not None and is_critical_path(r, project, True):
                return (
                    "rsync --delete into the root, home or project directory detected.",
                    "Sync into a dedicated subdirectory instead.",
                )
    return None


# ---------------------------------------------------------------------------
# Category 2: Git force operations
# ---------------------------------------------------------------------------

# git global options that take a separate value (`git -C dir push ...`).
_GIT_VALUE_OPTS = frozenset({
    "-C", "-c", "--git-dir", "--work-tree", "--namespace", "--super-prefix",
    "--config-env", "--exec-path", "--attr-source",
})
_PUSH_VALUE_OPTS = frozenset({"-o", "--push-option", "--repo", "--receive-pack", "--exec"})


def _git_subcommand(args: list[str], cwd: str) -> tuple[str, list[str], str]:
    """git's subcommand, its arguments, and the directory it runs in (cwd as
    changed by -C)."""
    k = 0
    while k < len(args) and args[k].startswith("-"):
        if args[k] == "-C" and k + 1 < len(args):
            cwd = resolve_dir(args[k + 1], cwd) or cwd
        k += 2 if args[k] in _GIT_VALUE_OPTS else 1
    if k >= len(args):
        return "", [], cwd
    return args[k], args[k + 1:], cwd


def current_branch(repo_dir: str) -> str | None:
    """The branch checked out in repo_dir, or None (detached, not a repo, or
    git unavailable)."""
    try:
        out = subprocess.run(
            ["git", "-C", repo_dir, "symbolic-ref", "--short", "-q", "HEAD"],
            capture_output=True, text=True, timeout=2,
        )
    except (OSError, subprocess.SubprocessError):
        return None
    if out.returncode != 0:
        return None
    return out.stdout.strip() or None


def _branch_of(ref: str) -> str:
    for prefix in ("refs/heads/", "heads/"):
        if ref.startswith(prefix):
            return ref[len(prefix):]
    return ref


def _push_rewrites_protected(args: list[str], repo_dir: str) -> bool:
    force = delete = False
    positional: list[str] = []
    k = 0
    while k < len(args):
        a = args[k]
        if a in ("--force", "--force-if-includes") or a.startswith("--force-with-lease"):
            force = True
        elif a in ("--delete", "--mirror", "--prune"):
            delete = True
        elif a in _PUSH_VALUE_OPTS:
            k += 1
        elif a.startswith("-") and not a.startswith("--") and len(a) > 1:
            force = force or "f" in a[1:]
            delete = delete or "d" in a[1:]
        elif not a.startswith("-"):
            positional.append(a)
        k += 1
    if "--mirror" in args:
        return True  # force-updates and deletes every remote ref
    if force and ("--all" in args or "--branches" in args):
        return True  # force-pushes every local branch, protected ones included
    protected = set(PROTECTED_BRANCHES)
    # No refspec pushes the current branch, as does a HEAD (or @) refspec.
    specs = positional[1:] or ["HEAD"]  # positional[0] is the repository
    for spec in specs:
        plus = spec.startswith("+")
        spec = spec.lstrip("+")
        src, sep, dst = spec.partition(":")
        target = _branch_of(dst if sep else src)
        if not (force or delete or plus or (sep and not src)):
            continue
        if target in ("HEAD", "@"):
            target = current_branch(repo_dir) or ""
        if target in protected:
            return True
    return False


def check_git(cmds: list[Cmd], cwd: str) -> tuple[str, str] | None:
    """Check for destructive git operations: force pushes and deletes of
    protected branches, hard resets, forced cleans and forced branch deletes."""
    for cmd, cmd_cwd in zip(cmds, working_dirs(cmds, cwd)):
        argv = effective_argv(cmd.argv)
        if not argv or prog(argv[0]) != "git":
            continue
        sub, args, repo_dir = _git_subcommand(argv[1:], cmd_cwd)
        flags = _short_flags(args)
        if sub == "push" and _push_rewrites_protected(args, repo_dir):
            return (
                "Force push, delete or mirror of a protected branch detected.",
                "Use a feature branch and PR workflow instead of force-pushing.",
            )
        if sub == "reset" and "--hard" in args:
            return (
                "Hard reset detected — this discards uncommitted changes.",
                "Use git stash or create a backup branch before resetting.",
            )
        if sub == "clean" and ("f" in flags or "--force" in args) and not (
            "n" in flags or "--dry-run" in args
        ):
            return (
                "Git clean with force detected — removes untracked files permanently.",
                "Review untracked files with git clean -n (dry run) first.",
            )
        if sub == "branch" and (
            "D" in flags
            or (("d" in flags or "--delete" in args) and ("f" in flags or "--force" in args))
        ):
            return (
                "Force branch deletion detected.",
                "Use git branch -d (lowercase) for safe deletion that checks merge status.",
            )
    return None


# ---------------------------------------------------------------------------
# Category 3: Database destruction
# ---------------------------------------------------------------------------

SQL_CLIENTS: frozenset[str] = frozenset({
    "psql", "pgcli", "mysql", "mariadb", "mycli", "sqlite3", "sqlite", "litecli",
    "duckdb", "sqlcmd", "usql", "clickhouse-client", "clickhouse", "cockroach",
    "snowsql", "trino", "presto", "isql",
})

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


def _sql_texts(cmds: list[Cmd], command: str) -> list[str]:
    """SQL a command sends to a database client: the client's own arguments,
    or the whole command line when the client reads a pipe, heredoc or file
    from stdin. Commands without a SQL client (grep, git commit) send none."""
    texts: list[str] = []
    for cmd in cmds:
        if not any(prog(t) in SQL_CLIENTS for t in cmd.argv):
            continue
        texts.append(" ".join(cmd.argv))
        if cmd.pipe_in or any(op.startswith("<") for op, _ in cmd.redirects):
            texts.append(command)
    return texts


def check_database(cmds: list[Cmd], command: str) -> tuple[str, str] | None:
    for text in _sql_texts(cmds, command):
        for pattern, reason, remediation in DB_PATTERNS:
            if pattern.search(text):
                return reason, remediation
        for stmt in text.split(";"):
            if re.search(r'\bDELETE\s+FROM\s+\S+', stmt, re.IGNORECASE) and not re.search(
                r'\bWHERE\b', stmt, re.IGNORECASE
            ):
                return (
                    "DELETE FROM without WHERE clause — deletes all rows.",
                    "Add a WHERE clause to target specific rows, or use TRUNCATE if intended.",
                )
    return None


# ---------------------------------------------------------------------------
# Category 4: Remote code execution
# ---------------------------------------------------------------------------

FETCHERS: frozenset[str] = frozenset({
    "curl", "wget", "fetch", "aria2c", "http", "https", "xh", "lwp-request",
    "iwr", "irm", "invoke-webrequest", "invoke-restmethod",
})
# Interpreters that run code they read from stdin when given no script.
CODE_INTERPRETERS = re.compile(
    r"^(python[0-9.]*|pypy[0-9.]*|node|nodejs|deno|bun|perl|ruby|php|lua|tclsh|osascript)$"
)
_EVAL_FLAGS = frozenset({"-c", "-m", "-e", "-E", "-r", "-p", "--eval", "--print", "run"})
EXPR_EVALUATORS: frozenset[str] = frozenset({"iex", "invoke-expression"})
SOURCE_COMMANDS: frozenset[str] = frozenset({"source", ".", "eval"})


def _runs_stdin_as_code(argv: list[str]) -> bool:
    """Whether a command executes what it reads on stdin."""
    name = prog(argv[0])
    if name in SHELLS or name in POWERSHELLS or name in EXPR_EVALUATORS:
        return True
    if not CODE_INTERPRETERS.match(name):
        return False
    args = argv[1:]
    if any(a in _EVAL_FLAGS for a in args):
        return False
    positional = [a for a in args if not a.startswith("-")]
    return not positional or positional[0] == "-" or "-" in args


def _runs_argument_as_code(argv: list[str]) -> bool:
    """Whether a command executes a script given as one of its arguments."""
    name = prog(argv[0])
    return (
        name in SHELLS or name in POWERSHELLS or name in EXPR_EVALUATORS
        or name in SOURCE_COMMANDS or bool(CODE_INTERPRETERS.match(name))
    )


def _is_fetch(cmd: Cmd) -> bool:
    argv = effective_argv(cmd.argv)
    return bool(argv) and prog(argv[0]) in FETCHERS


_PS_DOWNLOAD_EXEC = re.compile(
    r"(?i)(\b(iex|invoke-expression)\b|\[scriptblock\]::create).*\b(downloadstring|downloadfile|iwr|irm|invoke-webrequest"
    r"|invoke-restmethod|start-bitstransfer|curl|wget)\b"
)


def check_remote_code(cmds: list[Cmd], command: str, powershell: bool) -> tuple[str, str] | None:
    fetch_pipelines = {c.pipeline for c in cmds if _is_fetch(c)}
    for idx, cmd in enumerate(cmds):
        # Downloaded content piped into anything that executes stdin.
        if cmd.pipe_in and cmd.pipeline in fetch_pipelines and any(
            c.pipeline == cmd.pipeline and _is_fetch(c) for c in cmds[:idx]
        ):
            raw = cmd.argv[0] if cmd.argv else ""
            if prog(raw) in ("sudo", "doas"):
                return (
                    "Pipe-to-sudo pattern detected.",
                    "Download the script first, review it, then execute with appropriate permissions.",
                )
            argv = effective_argv(cmd.argv)
            if argv and _runs_stdin_as_code(argv):
                return (
                    "Pipe-to-interpreter pattern detected (downloaded content piped to a shell or interpreter).",
                    "Download the script first, review it, then execute.",
                )
        # A download used as the script of a shell/interpreter/eval:
        # sh -c "$(curl ...)", bash <(curl ...), eval "$(wget -O- ...)".
        if _is_fetch(cmd) and cmd.parent is not None:
            parent = effective_argv(cmd.parent.argv)
            if parent and _runs_argument_as_code(parent):
                return (
                    "Downloaded script executed via command or process substitution.",
                    "Download the script first, review it, then execute.",
                )
    if powershell and _PS_DOWNLOAD_EXEC.search(command):
        return (
            "Invoke-Expression of downloaded content detected.",
            "Download the script first, review it, then execute.",
        )
    return None


# ---------------------------------------------------------------------------
# Category 5: Consulting cross-environment
# ---------------------------------------------------------------------------

_SSH_VALUE_OPTS = frozenset("BbcDEeFIiJLlmOoPpQRSWw")
_NAME_PARTS = re.compile(r"[^a-z0-9]+")


def _name_parts(text: str) -> set[str]:
    return {p for p in _NAME_PARTS.split(text.lower()) if p}


def _host_of(dest: str) -> str:
    """Host of an ssh/scp destination (user@host, host:path, ssh://user@host:port/)."""
    if "://" in dest:
        dest = dest.split("://", 1)[1].split("/", 1)[0]
    dest = dest.rsplit("@", 1)[-1]
    return dest.split(":", 1)[0]


def _ssh_destinations(name: str, args: list[str]) -> list[str]:
    positional: list[str] = []
    k = 0
    while k < len(args):
        a = args[k]
        if a == "--":
            positional.extend(args[k + 1:])
            break
        if a.startswith("-") and len(a) > 1:
            if len(a) == 2 and a[1] in _SSH_VALUE_OPTS:
                k += 1
        else:
            positional.append(a)
        k += 1
    if name == "ssh":
        return positional[:1]
    return [p for p in positional if ":" in p.split("/", 1)[0] or "://" in p]


def _option_values(args: list[str], names: tuple[str, ...]) -> list[str]:
    values: list[str] = []
    for k, a in enumerate(args):
        for n in names:
            if a == n and k + 1 < len(args):
                values.append(args[k + 1])
            elif a.startswith(n + "="):
                values.append(a.split("=", 1)[1])
    return values


def _first_positional(args: list[str], value_opts: frozenset[str]) -> tuple[str, list[str]]:
    k = 0
    while k < len(args) and args[k].startswith("-"):
        k += 2 if args[k] in value_opts else 1
    return (args[k], args[k + 1:]) if k < len(args) else ("", [])


_KUBE_VALUE_OPTS = frozenset({
    "-n", "--namespace", "--context", "--kube-context", "--cluster", "--kubeconfig",
    "-s", "--server", "--user", "--token", "-l", "--selector",
})
_DOCKER_VALUE_OPTS = frozenset({
    "-H", "--host", "-c", "--context", "--config", "-l", "--log-level",
    "--tlscacert", "--tlscert", "--tlskey",
})


def _file_op_operands(name: str, args: list[str]) -> list[str]:
    """The cp/mv/rsync operands that the command writes or removes: the
    destination (-t DIR, else the last operand) for cp and rsync, whose
    sources are only read, and every operand for mv, which also removes its
    sources."""
    value_opts = {"-S", "--suffix"}
    if name == "rsync":
        value_opts |= {"-e", "--rsh", "--rsync-path", "--exclude", "--include", "--filter", "-f"}
    targets: list[str] = []
    operands: list[str] = []
    k = 0
    while k < len(args):
        a = args[k]
        if a in ("-t", "--target-directory") and k + 1 < len(args):
            targets.append(args[k + 1])
            k += 1
        elif a.startswith("--target-directory="):
            targets.append(a.split("=", 1)[1])
        elif a in value_opts:
            k += 1
        elif not a.startswith("-"):
            operands.append(a)
        k += 1
    if name == "mv":
        return targets + operands
    if targets:
        return targets
    return operands[-1:] if len(operands) > 1 else []


def _outside_project_operand(name: str, args: list[str], cwd: str, project: str) -> str | None:
    """The first operand cp/mv/rsync writes that resolves outside the project
    tree and outside the temp directories, else None."""
    project_real = _slash(os.path.realpath(project))
    for operand in _file_op_operands(name, args):
        if name == "rsync" and ":" in operand.split("/", 1)[0]:
            continue  # remote destination: covered by the host check
        resolved = resolve_path(operand, cwd)
        if resolved is None:
            continue
        real = _slash(os.path.realpath(resolved))
        if _covers(project_real, real) or _covers(_normpath(_slash(project)), resolved):
            continue
        if _is_temp(resolved) or _is_temp(real):
            continue
        return operand
    return None


def check_cross_environment(cmds: list[Cmd], cwd: str, project: str) -> tuple[str, str] | None:
    """Check for operations targeting production or outside project scope."""
    for cmd, cwd in zip(cmds, working_dirs(cmds, cwd)):
        argv = effective_argv(cmd.argv)
        if not argv:
            continue
        name, args = prog(argv[0]), argv[1:]

        if name in ("ssh", "scp", "sftp", "mosh", "rsync"):
            for dest in _ssh_destinations("ssh" if name in ("ssh", "mosh") else name, args):
                hit = _name_parts(_host_of(dest)) & set(PRODUCTION_HOST_PATTERNS)
                if hit:
                    return (
                        f"SSH/SCP to production-like host detected (matched: {sorted(hit)[0]}).",
                        "Use deployment pipelines instead of direct production access.",
                    )

        if name in ("kubectl", "helm", "docker", "podman"):
            value_opts = _KUBE_VALUE_OPTS if name in ("kubectl", "helm") else _DOCKER_VALUE_OPTS
            sub, rest = _first_positional(args, value_opts)
            if sub in ("apply", "deploy", "push"):
                # Contexts, namespaces and daemon hosts match on any name
                # part; an image tag must equal a production name exactly,
                # so app:production-candidate is not a production deploy.
                hit = any(
                    _name_parts(v) & DEPLOY_ENVIRONMENTS
                    for v in _option_values(
                        args, ("-n", "--namespace", "--context", "--kube-context", "--cluster", "-H", "--host")
                    )
                )
                if name in ("docker", "podman") and sub == "push":
                    for ref in (a for a in rest if not a.startswith("-")):
                        last = ref.rsplit("/", 1)[-1]
                        tag = last.split(":", 1)[1] if ":" in last else ""
                        registry = ref.split("/", 1)[0] if "/" in ref else ""
                        hit = hit or tag.lower() in DEPLOY_ENVIRONMENTS or (
                            ("." in registry or ":" in registry)
                            and bool(_name_parts(registry) & DEPLOY_ENVIRONMENTS)
                        )
                if hit:
                    return (
                        "Deployment command targeting production detected.",
                        "Use CI/CD pipelines for production deployments.",
                    )

        if name == "ansible-playbook" or (
            name in ("terraform", "tofu", "terragrunt") and _terraform_subcommand(args)[0] == "apply"
        ):
            if any(_name_parts(a) & DEPLOY_ENVIRONMENTS for a in args):
                return (
                    "Infrastructure command targeting production detected.",
                    "Use CI/CD pipelines for production deployments.",
                )

        if name in ("cp", "mv", "rsync") and project:
            outside = _outside_project_operand(name, args, cwd, project)
            if outside is not None:
                return (
                    f"File operation targets path outside project directory: {outside}",
                    "Restrict file operations to the current project tree.",
                )
    return None


# ---------------------------------------------------------------------------
# Category 6: Infrastructure destruction
# ---------------------------------------------------------------------------

def _terraform_subcommand(args: list[str]) -> tuple[str, list[str]]:
    for k, a in enumerate(args):
        if not a.startswith("-"):
            return a, args[k + 1:]
    return "", []


def check_infrastructure(cmds: list[Cmd]) -> tuple[str, str] | None:
    for cmd in cmds:
        argv = effective_argv(cmd.argv)
        if not argv:
            continue
        name, args = prog(argv[0]), argv[1:]
        if name in ("terraform", "tofu", "terragrunt"):
            sub, rest = _terraform_subcommand(args)
            auto = any(
                a.lstrip("-") == "auto-approve" or a.lstrip("-").startswith("auto-approve=t")
                for a in rest
            )
            destroy = sub == "destroy" or (sub == "apply" and any(a.lstrip("-") == "destroy" for a in rest))
            if destroy and auto:
                return (
                    "Terraform/OpenTofu auto-approved destroy detected.",
                    "Review the destroy plan manually: terraform plan -destroy.",
                )
        if name in ("docker", "podman"):
            sub, rest = _first_positional(args, _DOCKER_VALUE_OPTS)
            action, opts = _first_positional(rest, frozenset())
            flags = _short_flags(opts)
            if sub == "system" and action == "prune" and ("a" in flags or "--all" in opts):
                return (
                    "Docker system prune --all detected — removes all unused data.",
                    "Use targeted docker prune commands (image, container, volume).",
                )
            if sub == "volume" and action == "prune" and (
                flags & {"a", "f"} or "--all" in opts or "--force" in opts
            ):
                return (
                    "Docker volume force prune detected — removes all unused volumes.",
                    "List volumes first with docker volume ls and remove specific ones.",
                )
        if name == "kubectl":
            sub, rest = _first_positional(args, _KUBE_VALUE_OPTS)
            if sub != "delete":
                continue
            resource, _ = _first_positional(rest, _KUBE_VALUE_OPTS)
            kind = resource.split("/", 1)[0].lower()
            if kind in ("namespace", "namespaces", "ns") or any(
                a in ("--all", "-A", "--all-namespaces") for a in rest
            ):
                return (
                    "Kubernetes namespace or bulk deletion detected.",
                    "Verify the target and use kubectl delete with --dry-run=client first.",
                )
    return None


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
    tool_input = input_data.get("tool_input") or {}
    command = tool_input.get("command") or ""

    # A Monitor call watching a WebSocket (`ws`) has no command.
    if tool_name not in SHELL_TOOLS or not command:
        sys.exit(0)

    project = os.environ.get("CLAUDE_PROJECT_DIR", "")
    cwd = input_data.get("cwd") or project or os.getcwd()
    powershell = tool_name == "PowerShell"
    # PowerShell uses backslash as a path separator, not an escape.
    cmds = parse_commands(command.replace("\\", "/") if powershell else command)

    if _FORK_BOMB.search(command):
        deny("Filesystem Destruction", "Fork bomb detected.", "Do not run fork bombs.", command)

    checks = (
        ("Filesystem Destruction", lambda: check_filesystem(cmds, cwd, project)),
        ("Git Force Operation", lambda: check_git(cmds, cwd)),
        ("Database Destruction", lambda: check_database(cmds, command)),
        ("Remote Code Execution", lambda: check_remote_code(cmds, command, powershell)),
        ("Cross-Environment", lambda: check_cross_environment(cmds, cwd, project)),
        ("Infrastructure Destruction", lambda: check_infrastructure(cmds)),
    )
    for category, check in checks:
        result = check()
        if result:
            deny(category, result[0], result[1], command)

    # All checks passed — allow.
    audit_log({
        "event": "allow",
        "hook": "destructive-prevention",
        **command_fingerprint(command),
    })
    sys.exit(0)


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        print(f"destructive prevention error: {e}", file=sys.stderr)
        sys.exit(2)
