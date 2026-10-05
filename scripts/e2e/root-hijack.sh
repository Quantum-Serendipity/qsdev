#!/usr/bin/env bash
# root-hijack.sh: E3 regression for project-root hijacking (U01-01, U16-V02,
# U28-10, U24-14) and for commands that act where they run (U02-07, U12-05,
# U13-01).
#
# A shared, world-writable (1777) directory P holds planted project markers:
# P/.qsdev.yaml, P/.devinit/, P/.mcp.json naming an "evil-mcp" server and
# P/.qsdev/defaults.yaml adding an "evil-hook" custom hook to the baseline
# tier. From a git child and a non-git child of P, every command must act on
# the child and ignore P entirely:
#
#   - status exits 2 (project not initialized);
#   - defaults show lacks evil-hook (the ancestor overlay is not applied);
#   - claude init --dry-run and devenv init --dry-run plan files instead of
#     refusing P as an un-joined checkout;
#   - mcp grade sees no servers (not P's .mcp.json);
#   - teardown --dry-run plans against the child;
#   - mcp serve (from a non-git child of a shared directory with a planted
#     .git and config) neither reads that config nor logs under it;
#   - nothing under P (outside the children) is created or changed.
#
# Commands that create a project where they run must act on the working
# directory even below a trusted ancestor project: init --update in an empty
# child of an ancestor holding the state directory (U02-07), and devenv init
# from <proj>/services/api (U12-05/U13-01).
#
# The project defaults layer (.qsdev/defaults.yaml) comes from the root the
# command resolves (U01-WS1): a real init --yes in a non-git child of a
# trusted project with an overlay generates without it, while defaults show
# (an Enclosing command) from below the child still applies it; and an
# overlay another local user could have written (a 1777 working directory,
# or a world-writable .qsdev/) is refused with "refusing project defaults"
# instead of being applied.
#
# Usage: scripts/e2e/root-hijack.sh [path/to/qsdev]
# Without an argument (or $QSDEV_BIN) the binary is built into the scratch
# directory. Everything runs in a fresh mktemp -d with its own HOME, TMPDIR
# and XDG_* directories; nothing outside it is touched.
set -euo pipefail

repo=$(cd "$(dirname "$0")/../.." && pwd)
work=$(mktemp -d)
cleanup() {
	chmod -R u+rwX "$work" 2>/dev/null || true
	rm -rf "$work"
}
trap cleanup EXIT

bin=${1:-${QSDEV_BIN:-}}
if [[ -z $bin ]]; then
	bin=$work/bin/qsdev
	(cd "$repo" && GOWORK=off go build -o "$bin" ./cmd/qsdev)
fi
bin=$(cd "$(dirname "$bin")" && pwd)/$(basename "$bin")

home=$work/home
mkdir -p "$home" "$work/tmp" "$work/xdg/config" "$work/xdg/state" "$work/xdg/cache" "$work/xdg/data"

# qsdev runs the binary under a scrubbed environment: only PATH (for git;
# $qpath when set), the fresh per-run directories, and system setup skipped
# so no command can install anything.
qpath=
qsdev() {
	env -i PATH="${qpath:-$PATH}" QSDEV_SKIP_SETUP=1 HOME="$home" USERPROFILE="$home" TMPDIR="$work/tmp" \
		XDG_CONFIG_HOME="$work/xdg/config" XDG_STATE_HOME="$work/xdg/state" \
		XDG_CACHE_HOME="$work/xdg/cache" XDG_DATA_HOME="$work/xdg/data" \
		GIT_CONFIG_NOSYSTEM=1 "$bin" "$@"
}

P=$work/P
mkdir -p "$P/.qsdev" "$P/.devinit" "$P/gitchild" "$P/plainchild"
printf 'version: 2\nsecurity:\n  level: standard\n' >"$P/.qsdev.yaml"
printf '{"mcpServers":{"evil-mcp":{"command":"/bin/true"}}}\n' >"$P/.mcp.json"
# overlay writes the project defaults file at $1 that adds the evil-hook
# custom hook to the baseline tier.
overlay() {
	cat >"$1" <<'EOF'
custom_hooks:
  - id: evil-hook
    name: Evil
    description: planted by another local user
    entry: ./evil.sh
    language: system
    stages: [pre-commit]
hook_tiers:
  baseline:
    - evil-hook
EOF
}
overlay "$P/.qsdev/defaults.yaml"
env -i PATH="$PATH" HOME="$home" GIT_CONFIG_NOSYSTEM=1 git -C "$P/gitchild" init -q
chmod 1777 "$P"

# snapshot lists every path under P outside the children with a checksum of
# each regular file, so a created or rewritten file shows up as a diff.
snapshot() {
	(cd "$P" && find . \( -path ./gitchild -o -path ./plainchild \) -prune -o -print | LC_ALL=C sort &&
		find . \( -path ./gitchild -o -path ./plainchild \) -prune -o -type f -exec cksum {} + | LC_ALL=C sort)
}
before=$(snapshot)

failures=0
fail() {
	echo "FAIL [$child] $*" >&2
	failures=$((failures + 1))
}

# run executes qsdev with the given arguments in the current child, capturing
# combined output in $out and the exit status in $rc.
run() {
	set +e
	out=$(qsdev "$@" 2>&1)
	rc=$?
	set -e
}

for child in gitchild plainchild; do
	cd "$P/$child"

	run status
	[[ $rc -eq 2 && $out == *"not initialized"* ]] || fail "status: rc=$rc, want 2 (not initialized): $out"

	run defaults show
	[[ $rc -eq 0 ]] || fail "defaults show: rc=$rc: $out"
	[[ $out != *evil-hook* ]] || fail "defaults show applied the ancestor overlay (evil-hook)"

	run claude init --dry-run
	[[ $rc -eq 0 && $out == *".claude/settings.json"* ]] || fail "claude init --dry-run: rc=$rc, want a plan for the child: $out"

	run devenv init --dry-run
	[[ $rc -eq 0 && $out == *"devenv.nix"* ]] || fail "devenv init --dry-run: rc=$rc, want a plan for the child: $out"

	run mcp grade
	[[ $rc -eq 0 && $out == *"No MCP servers configured"* && $out != *evil-mcp* ]] ||
		fail "mcp grade: rc=$rc, want no servers (not the ancestor's .mcp.json): $out"

	run teardown --dry-run
	[[ $rc -eq 0 && $out == *"No files were modified"* ]] || fail "teardown --dry-run: rc=$rc, want a plan for the child: $out"
done

# U24-14: the MCP server resolves the same root as the CLI. In a shared
# (1777) directory R another user planted R/.git and an unparsable
# R/.qsdev.yaml; from the non-git child R/victim, mcp serve must neither
# parse R's config as its guardrail policy nor write its session log under R.
R=$work/R
mkdir -p "$R/victim"
env -i PATH="$PATH" HOME="$home" GIT_CONFIG_NOSYSTEM=1 git -C "$R" init -q
printf 'version: [unterminated\n' >"$R/.qsdev.yaml"
chmod 1777 "$R"
rsnap() { (cd "$R" && find . -path ./victim -prune -o -print | LC_ALL=C sort); }
rbefore=$(rsnap)
child=R/victim
cd "$R/victim"
run mcp serve </dev/null
[[ $out != *"MCP guardrail policy"* && $out != *"$R/.qsdev.yaml"* ]] ||
	fail "mcp serve read the planted ancestor config: rc=$rc: $out"
[[ $(rsnap) == "$rbefore" ]] || fail "mcp serve created files under R: $(diff <(echo "$rbefore") <(rsnap))"

# Commands that create a project where they run (init, devenv init) act on
# the working directory even below a trusted, initialized ancestor.
#
# U02-07: in an empty non-git directory below an ancestor holding the state
# directory, init --update reports the child, never the ancestor.
A=$work/anc
mkdir -p "$A/.devinit" "$A/child"
child=anc/child
cd "$A/child"
run init --update --dry-run --yes
[[ $out == *"$A/child/"* && $out != *"$A/.devinit"* ]] ||
	fail "init --update --dry-run --yes: rc=$rc, want it to act on the child, not $A: $out"

# U12-05/U13-01: from <proj>/services/api of an un-joined checkout that
# already has a devenv.nix, devenv init plans a new devenv.nix in
# services/api instead of refusing (or overwriting) the enclosing project.
Q=$work/proj
mkdir -p "$Q/services/api"
printf 'version: 2\nsecurity:\n  level: standard\n' >"$Q/.qsdev.yaml"
printf '{ ... }: { }\n' >"$Q/devenv.nix"
env -i PATH="$PATH" HOME="$home" GIT_CONFIG_NOSYSTEM=1 git -C "$Q" init -q
child=proj/services/api
cd "$Q/services/api"
run devenv init --dry-run
[[ $rc -eq 0 && $out =~ devenv\.nix[[:space:]]+create ]] ||
	fail "devenv init --dry-run: rc=$rc, want a plan creating devenv.nix in services/api: $out"
[[ ! -e $Q/services/api/devenv.nix ]] || fail "devenv init --dry-run wrote devenv.nix"

# U01-WS1: a trusted ancestor project T with an overlay. defaults show from
# a directory below the non-git child (no project of its own) keeps T's
# policy; a real init --yes in the child generates without it and changes
# nothing under T outside the child. init runs with only git on PATH.
gitbin=$work/gitbin
mkdir -p "$gitbin"
ln -s "$(command -v git)" "$gitbin/git"
T=$work/trusted
mkdir -p "$T/.qsdev" "$T/sub/deeper"
printf 'version: 2\nsecurity:\n  level: standard\n' >"$T/.qsdev.yaml"
overlay "$T/.qsdev/defaults.yaml"
child=trusted/sub/deeper
cd "$T/sub/deeper"
run defaults show
[[ $rc -eq 0 && $out == *evil-hook* ]] ||
	fail "defaults show: rc=$rc, want the enclosing project's evil-hook (an Enclosing command keeps its policy): $out"
tsnap() {
	(cd "$T" && find . -path ./sub -prune -o -print | LC_ALL=C sort &&
		find . -path ./sub -prune -o -type f -exec cksum {} + | LC_ALL=C sort)
}
tbefore=$(tsnap)
child=trusted/sub
cd "$T/sub"
qpath=$gitbin
run init --yes --lang go
qpath=
[[ $rc -eq 0 ]] || fail "init --yes --lang go: rc=$rc: $out"
[[ -f $T/sub/devenv.nix ]] || fail "init --yes wrote no devenv.nix in the child: $out"
hits=$(grep -c evil-hook "$T/sub/devenv.nix" 2>/dev/null || true)
[[ $hits == 0 ]] || fail "init applied the ancestor's project defaults: devenv.nix has evil-hook ($hits)"
[[ $(tsnap) == "$tbefore" ]] || fail "init changed $T outside the child: $(diff <(echo "$tbefore") <(tsnap))"

# U01-WS1: an overlay another local user could have written is refused,
# never applied: one in a world-writable (1777) working directory, and one
# under a world-writable .qsdev/ of an otherwise trusted directory.
U=$work/sharedcwd
mkdir -p "$U/.qsdev"
overlay "$U/.qsdev/defaults.yaml"
chmod 1777 "$U"
V=$work/opendir
mkdir -p "$V/.qsdev"
overlay "$V/.qsdev/defaults.yaml"
chmod 0666 "$V/.qsdev/defaults.yaml"
chmod 0777 "$V/.qsdev"
for dir in "$U" "$V"; do
	child=${dir#"$work"/}
	cd "$dir"
	for args in "defaults show" "init --yes --dry-run --lang go"; do
		# shellcheck disable=SC2086 # args is a word list
		run $args
		[[ $rc -eq 1 && $out == *"refusing project defaults"* && $out == *"$dir/.qsdev/defaults.yaml"* ]] ||
			fail "$args: rc=$rc, want 1 refusing $dir/.qsdev/defaults.yaml: $out"
		[[ $out != *evil-hook* ]] || fail "$args applied the untrusted overlay (evil-hook)"
	done
done

child=P
after=$(snapshot)
if [[ $before != "$after" ]]; then
	fail "files under P changed:"
	diff <(echo "$before") <(echo "$after") >&2 || true
fi

if ((failures > 0)); then
	echo "root-hijack: $failures failure(s)" >&2
	exit 1
fi
echo "root-hijack: ok"
