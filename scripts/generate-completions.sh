#!/bin/sh
set -eu

# Generate shell completion scripts for packaging (e.g. nFPM).
# Requires ./bin/qsdev to be built first.

mkdir -p dist_share/completions
./bin/qsdev completion bash > dist_share/completions/qsdev.bash
./bin/qsdev completion zsh > dist_share/completions/qsdev.zsh
./bin/qsdev completion fish > dist_share/completions/qsdev.fish
