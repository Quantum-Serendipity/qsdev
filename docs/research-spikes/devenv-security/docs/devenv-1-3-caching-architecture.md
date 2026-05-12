<!-- Source: https://devenv.sh/blog/2024/10/03/devenv-13-instant-developer-environments-with-nix-caching/ -->
<!-- Retrieved: 2026-05-12 -->

# devenv 1.3: Instant Developer Environments with Nix Caching

## Overview

The devenv 1.3 release introduces "precise caching to Nix evaluation, significantly speeding up developer environments." Once cached, "results of a Nix eval or build can be recalled in single-digit milliseconds."

## How the Caching Architecture Works

The system operates through a multi-step process:

**Parsing Phase:**
Devenv parses Nix's internal logs during evaluation to identify which files and directories are accessed. For each path, the system records:
- The complete file path
- A hash of file contents
- The last modification timestamp

This metadata gets stored in a SQLite database.

**Retrieval and Validation:**
When running devenv commands, the system:
1. Queries the database for previously accessed paths
2. Compares current file hashes and timestamps against stored values
3. Invalidates the cache if differences are detected, triggering full re-evaluation
4. Uses cached results when no changes are found

**Change Detection:**
This approach enables detection of modifications including direct changes to Nix files, modifications to imported files or directories, and updates to files read via Nix built-ins like `readFile` or `readDir`.

## Design Inspiration

The caching approach draws inspiration from lorri, a tool that pioneered parsing Nix logs for caching purposes. However, devenv integrates caching as an automated built-in feature "that works automatically without additional setup," distinguishing it from lorri's background daemon requirement.

## Comparison with Alternative Solutions

**Nix's Built-in Cache:**
Nix's native flake evaluation cache bases invalidation on input locks, overlooking changes during typical development workflows.

**direnv and nix-direnv:**
These tools struggle with limitations like requiring "manual file watching" and inability to detect changes in deeply nested imports.

## Future Developments

The team plans to bring `nix develop` functionality in-house to reduce cached shell launch overhead to under 100 milliseconds, particularly addressing macOS performance concerns.
