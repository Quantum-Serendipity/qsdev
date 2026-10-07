#!/usr/bin/env bash
# planted-markers.sh: the hermetic test job (XA-WS12). Runs the whole test
# suite with project markers planted above every temp directory a test can
# create, so a test that resolves a project from t.TempDir() without its own
# .git ceiling walks into a planted ancestor and fails.
#
# The marker set (.qsdev.yaml, .qsdev/, .devinit/, .claude/settings.json,
# CLAUDE.md, devenv.nix; never a .git) is planted twice:
#
#   - into $PLANT_ROOT (default /tmp), the shared world-writable directory.
#     qsdev ignores markers under a 1777 directory, so this alone is vacuous;
#     it covers a test that reads markers without the trust check.
#   - into a fresh 0755 directory owned by this user, created under
#     $RUNNER_TEMP (or $TMPDIR, or /tmp) and exported as TMPDIR/TMP/TEMP.
#     Those markers are trusted, which is what a developer machine with a
#     project in an ancestor of its temp directory looks like.
#
# The suite runs in vendor mode with GOWORK=off and -shuffle=on. Extra
# arguments are passed to go test and replace the default ./... package list,
# e.g. to reproduce a shuffle order:
#
#   scripts/ci/planted-markers.sh -shuffle=1759700000000000000 ./addons/...
#
# On a developer machine set PLANT_ROOT to a scratch directory instead of
# planting into the shared /tmp. A marker already in $PLANT_ROOT is refused,
# never overwritten; everything planted is removed on exit.
set -euo pipefail

plant_root="${PLANT_ROOT:-/tmp}"
markers=(.qsdev.yaml .qsdev .devinit .claude CLAUDE.md devenv.nix)

for m in "${markers[@]}"; do
	if [[ -e $plant_root/$m || -L $plant_root/$m ]]; then
		echo "planted-markers: $plant_root/$m already exists; refusing to plant over it" >&2
		exit 1
	fi
done

trusted=""
# shellcheck disable=SC2329 # invoked by the EXIT trap below
cleanup() {
	for m in "${markers[@]}"; do
		rm -rf "${plant_root:?}/$m"
	done
	if [[ -n $trusted ]]; then
		rm -rf "$trusted"
	fi
}
trap cleanup EXIT

# plant writes the marker set into $1.
plant() {
	mkdir -p "$1/.qsdev" "$1/.devinit" "$1/.claude"
	printf 'version: 2\nsecurity:\n  level: standard\n' >"$1/.qsdev.yaml"
	printf 'tier: baseline\n' >"$1/.qsdev/defaults.yaml"
	printf '{"permissions":{"allow":["Bash(*)"]}}\n' >"$1/.claude/settings.json"
	printf '# Planted marker project\n' >"$1/CLAUDE.md"
	printf '{ ... }: { }\n' >"$1/devenv.nix"
}

plant "$plant_root"
trusted="$(mktemp -d "${RUNNER_TEMP:-${TMPDIR:-/tmp}}/planted.XXXXXX")"
chmod 0755 "$trusted"
plant "$trusted"

export TMPDIR="$trusted" TMP="$trusted" TEMP="$trusted" GOWORK=off

args=("$@")
if ((${#args[@]} == 0)); then
	args=(./...)
fi

cd "$(cd "${BASH_SOURCE[0]%/*}/../.." && pwd)"
status=0
go test -mod=vendor -count=1 -shuffle=on "${args[@]}" || status=$?
exit "$status"
