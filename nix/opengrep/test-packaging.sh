#!/usr/bin/env bash
# Single entry point for checking the OpenGrep derivation (nix/opengrep),
# used by CI (.github/workflows/ci.yml, job opengrep-nix) and by humans when
# bumping OpenGrep. Works from any working directory and uses the
# flake-pinned nixpkgs, never <nixpkgs>.
#
#   test-packaging.sh                Build .#opengrep for the current system,
#                                    smoke-test --version, and run the rule
#                                    library against the built binary.
#   test-packaging.sh --hashes-only  Verify the pinned release hash of every
#                                    platform in the derivation (no build).
set -euo pipefail

# Checked explicitly: a failed substitution inside cd "$(...)" leaves cd "",
# which bash 3.2 (macOS /bin/bash) treats as a successful no-op.
top="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)" || {
	echo "test-packaging.sh: not inside the qsdev git checkout" >&2
	exit 1
}
cd "$top"

# Every platform the derivation advertises (meta.platforms, sources in
# nix/opengrep/default.nix).
systems=(x86_64-linux aarch64-linux x86_64-darwin aarch64-darwin)

usage() {
	echo "usage: $0 [--hashes-only]" >&2
}

build_and_test() {
	echo "Building OpenGrep from the flake..."
	local out version
	out="$(nix build --no-link --print-out-paths .#opengrep)"
	version="$(nix eval --raw .#opengrep.version)"

	echo "Testing binary reports version ${version}..."
	local got
	got="$("$out/bin/opengrep" --version)"
	if [[ "$got" != *"$version"* ]]; then
		echo "test-packaging.sh: $out/bin/opengrep --version: expected ${version}, got: ${got}" >&2
		exit 1
	fi
	echo "$got"

	echo "Validating the rule library and fixtures..."
	PATH="$out/bin:$PATH" QSDEV_REQUIRE_RULE_SCANNER=1 \
		go test -count=1 -run 'TestCoreRules_' ./rules/

	echo "All tests passed."
}

verify_hashes() {
	local sys url want
	for sys in "${systems[@]}"; do
		url="$(nix eval --raw ".#packages.${sys}.opengrep.src.url")"
		want="$(nix eval --raw ".#packages.${sys}.opengrep.src.outputHash")"
		echo "Verifying ${sys}: ${url}"
		nix store prefetch-file --expected-hash "$want" "$url"
	done
	echo "All platform hashes verified."
}

case "$#:${1:-}" in
0:) build_and_test ;;
1:--hashes-only) verify_hashes ;;
*)
	usage
	exit 2
	;;
esac
