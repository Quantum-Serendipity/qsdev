#!/usr/bin/env bash
# Builds the prebuilt OpenGrep package and runs the rule library through it.
# Run from the repository root.
set -euo pipefail

echo "Building OpenGrep prebuilt package..."
out="$(nix-build --no-out-link "$(dirname "$0")")"

echo "Testing binary..."
"$out/bin/opengrep" --version

echo "Validating the rule library and fixtures..."
PATH="$out/bin:$PATH" QSDEV_REQUIRE_RULE_SCANNER=1 go test -count=1 -run 'TestCoreRules_' ./rules/

echo "All tests passed."
