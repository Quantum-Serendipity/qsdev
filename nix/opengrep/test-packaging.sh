#!/usr/bin/env bash
set -euo pipefail

echo "Building OpenGrep prebuilt package..."
nix-build "$(dirname "$0")"

echo "Testing binary..."
./result/bin/opengrep --version

echo "Testing rule validation..."
./result/bin/opengrep scan --validate --config rules/core/injection/go-sql-injection.yaml

echo "All tests passed."
