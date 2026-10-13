#!/usr/bin/env bash
# dogfood.sh - check that this repository is what current qsdev generates.
#
# Builds qsdev from source, copies the committed tree (`git archive HEAD`)
# into a fresh git repository under a temporary directory, and runs
#   1. `qsdev check`, which must pass every check,
#   2. `qsdev init --mode join`, which must regenerate every file unchanged,
#   3. `qsdev repair --reset` of the three-way-merged JSON files, which must
#      leave them unchanged too (join keeps an entry qsdev no longer generates
#      as user content, and `qsdev check` accepts it).
# Any failed check or any drift exits non-zero and prints the difference.
# A failure means the repository needs regenerating; CONTRIBUTING.md
# ("Dogfooding") has the procedure.
#
# Only committed content is checked: commit (locally) before running it.
#
# The copy gets the canonical repository as its origin remote, because
# generation derives files from it (.cosign/policy.yaml names the image
# repository and signing identity): DOGFOOD_ORIGIN if set, else this
# checkout's upstream remote, else its origin. Working from a fork, add an
# upstream remote or set DOGFOOD_ORIGIN to the canonical URL; https and ssh
# URLs of the same repository generate the same files.
#
# HOME, TMPDIR and XDG_* point into the temporary directory, which isolates
# almost all user state. The one exception is the org overlay: qsdev anchors
# it to the account's home from the user database (not $HOME), so a developer
# whose ~/.config/qsdev/defaults.yaml has settings can get a different result
# from CI. The script warns when that file has any uncommented line.
#
# The temporary directory is removed on exit; DOGFOOD_KEEP=1 keeps it (its
# path is printed) for inspecting a failure.
#
# Usage: scripts/dogfood.sh   (or `make dogfood`), from any directory.

set -euo pipefail

repo=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

# The canonical repository URL: DOGFOOD_ORIGIN, else the upstream remote
# (a fork's checkout), else origin.
origin=${DOGFOOD_ORIGIN:-}
if [ -z "$origin" ]; then
	origin=$(git -C "$repo" remote get-url upstream 2>/dev/null ||
		git -C "$repo" remote get-url origin 2>/dev/null || true)
fi
if [ -z "$origin" ]; then
	echo "dogfood: this checkout has no upstream or origin remote, and generated files" >&2
	echo "dogfood: derive from it; set DOGFOOD_ORIGIN to the canonical repository URL." >&2
	exit 1
fi

# jq normalises the .mcp.json release cutoffs before the drift check.
if ! command -v jq >/dev/null 2>&1; then
	echo "dogfood: jq is required (it is on GitHub's ubuntu runners and in the devenv shell)" >&2
	exit 1
fi

account_home=$(getent passwd "$(id -u)" 2>/dev/null | cut -d: -f6 || true)
overlay="${account_home:-$HOME}/.config/qsdev/defaults.yaml"
overlay_warning=""
if [ -f "$overlay" ] && grep -Eq '^[[:space:]]*[^#[:space:]]' "$overlay"; then
	overlay_warning="dogfood: WARNING: $overlay has settings and qsdev reads it whatever HOME is; a result here can differ from CI"
	echo "$overlay_warning" >&2
fi

tmp=$(mktemp -d)
# shellcheck disable=SC2329 # invoked by the EXIT trap below
cleanup() {
	if [ "${DOGFOOD_KEEP:-}" = 1 ]; then
		echo "dogfood: kept $tmp" >&2
	else
		rm -rf -- "$tmp"
	fi
}
trap cleanup EXIT

# A plain source build, as used to regenerate: devenv.nix embeds the
# generating version, so it must be "dev". A release or ldflags build would
# report drift, and so would VCS stamping, which gives a clean checkout a
# pseudo-version; -buildvcs=false keeps the version independent of git state.
# `qsdev check` resolves the `qsdev selfprotect` hook through PATH, so this
# binary must come first.
mkdir -p "$tmp/bin"
(cd "$repo" && GOWORK=off go build -mod=vendor -buildvcs=false -o "$tmp/bin/qsdev" ./cmd/qsdev)
export PATH="$tmp/bin:$PATH"

export HOME="$tmp/home"
export TMPDIR="$tmp/tmp"
export XDG_CONFIG_HOME="$tmp/home/.config"
export XDG_CACHE_HOME="$tmp/home/.cache"
export XDG_DATA_HOME="$tmp/home/.local/share"
export XDG_STATE_HOME="$tmp/home/.local/state"
export GIT_CONFIG_NOSYSTEM=1
mkdir -p "$HOME" "$TMPDIR" "$XDG_CONFIG_HOME" "$XDG_CACHE_HOME" "$XDG_DATA_HOME" "$XDG_STATE_HOME"

project="$tmp/qsdev"
mkdir -p "$project"
git -C "$repo" archive HEAD | tar -x -C "$project"
cd "$project"
git init -q
git remote add origin "$origin"
git add -A
git -c user.name=dogfood -c user.email=dogfood@localhost -c commit.gpgsign=false \
	commit -q --no-verify -m "dogfood snapshot"

status=0
echo "==> qsdev check"
if ! qsdev check </dev/null; then
	echo "dogfood: qsdev check failed" >&2
	status=1
fi

echo "==> qsdev init --mode join"
if ! qsdev init --mode join --quiet </dev/null; then
	echo "dogfood: qsdev init --mode join failed" >&2
	status=1
fi

# Reset the three-way-merged JSON files to exactly what qsdev generates, so
# the drift check below reports a stale hook or allow rule that join kept as
# user content.
for merged in .claude/settings.json .mcp.json; do
	echo "==> qsdev repair --reset --file $merged"
	if ! qsdev repair --reset --file "$merged" </dev/null >/dev/null; then
		echo "dogfood: qsdev repair --reset --file $merged failed" >&2
		status=1
	fi
done

# The .mcp.json release cutoffs depend on the day qsdev runs:
# mcpregistry.HardenedLauncherSpec age-gates an npx launcher with the env
# npm_config_before and a uvx launcher with a leading --exclude-newer=<cutoff>
# arg, both relative to the current UTC day, so a join on a later day
# rewrites them. `qsdev check` above validates the committed cutoffs. A
# change in the cutoffs alone is therefore not drift: when .mcp.json and
# HEAD's agree with every cutoff replaced by a placeholder, restore the
# committed file. Any other change is left for the drift check below.
# shellcheck disable=SC2016 # a jq program: $p is a jq variable
mcp_cutoff_filter='
def cutoffs($p):
	if type == "object" then
		(if (.env | type) == "object" and (.env | has("npm_config_before"))
			then .env.npm_config_before = $p else . end)
		| (if (.args | type) == "array"
			then .args |= map(if type == "string" and startswith("--exclude-newer=")
				then "--exclude-newer=" + $p else . end)
			else . end)
	else . end;
if (.mcpServers | type) == "object" then .mcpServers |= map_values(cutoffs($p)) else . end
'
mcp_cutoffs() { jq -S --arg p '<release-cutoff>' "$mcp_cutoff_filter"; }
if ! git diff --quiet -- .mcp.json; then
	if now=$(mcp_cutoffs <.mcp.json) && committed=$(git show HEAD:.mcp.json | mcp_cutoffs) &&
		[ -n "$now" ] && [ "$now" = "$committed" ]; then
		echo "==> .mcp.json: only the release cutoffs changed; restoring the committed file"
		git checkout -- .mcp.json
	fi
fi

drift=$(git status --porcelain --untracked-files=all)
if [ -n "$drift" ]; then
	echo "dogfood: regeneration changed the repository:" >&2
	echo "$drift" >&2
	git --no-pager diff >&2
	git ls-files --others --exclude-standard -z | xargs -0 -r -I{} \
		git --no-pager diff --no-index -- /dev/null {} >&2 || true
	status=1
fi

if [ "$status" -eq 0 ]; then
	echo "dogfood: ok"
elif [ -n "$overlay_warning" ]; then
	echo "$overlay_warning" >&2
fi
exit "$status"
