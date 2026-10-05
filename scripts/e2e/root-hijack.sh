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

# qsdev runs the binary under a scrubbed environment: only PATH (for git)
# and the fresh per-run directories.
qsdev() {
	env -i PATH="$PATH" HOME="$home" USERPROFILE="$home" TMPDIR="$work/tmp" \
		XDG_CONFIG_HOME="$work/xdg/config" XDG_STATE_HOME="$work/xdg/state" \
		XDG_CACHE_HOME="$work/xdg/cache" XDG_DATA_HOME="$work/xdg/data" \
		GIT_CONFIG_NOSYSTEM=1 "$bin" "$@"
}

P=$work/P
mkdir -p "$P/.qsdev" "$P/.devinit" "$P/gitchild" "$P/plainchild"
printf 'version: 2\nsecurity:\n  level: standard\n' >"$P/.qsdev.yaml"
printf '{"mcpServers":{"evil-mcp":{"command":"/bin/true"}}}\n' >"$P/.mcp.json"
cat >"$P/.qsdev/defaults.yaml" <<'EOF'
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
