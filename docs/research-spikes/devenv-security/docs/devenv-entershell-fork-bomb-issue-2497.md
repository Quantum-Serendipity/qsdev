<!-- Source: https://github.com/cachix/devenv/issues/2497 -->
<!-- Retrieved: 2026-05-12 -->

# GitHub Issue #2497: enterShell Tasks Fork Bomb via Re-evaluation Loop

## Issue Summary

This issue describes a critical problem in the devenv project where enterShell tasks that modify git-tracked files trigger exponential process growth.

## Root Cause Chain

The problem occurs through this sequence:

1. Running `devenv shell` executes enterShell tasks that alter tracked files
2. File modifications change the Nix flake's dirty-tree hash
3. direnv's hook fires `use devenv`, re-evaluating the flake with the new hash
4. Re-evaluation triggers enterShell tasks again, creating an infinite loop
5. If tasks use subprocess patterns calling `devenv tasks run`, each subprocess re-evaluates, causing exponential growth (6^n processes)

## Implemented Workarounds

The issue reporter documents three pragmatic solutions they had to implement:

- **`.envrc` guard**: Prevents recursive `use devenv` by checking if `DEVENV_ROOT` matches current directory
- **Idempotent writes**: All tasks compare before writing, skipping unchanged files
- **Recursion guard variable**: Tasks check for `_DEVENV_TASK_RUNNING` environment variable to prevent nested re-evaluation

## Proposed Solutions

The issue suggests several approaches:

- Track when enterShell completes for a given flake hash to prevent redundant runs
- Implement built-in recursion detection for nested `devenv tasks run` calls
- Allow tasks to declare file mutations and suppress direnv re-evaluation afterward
- Improve documentation regarding this pitfall and available workarounds
