# Bubblewrap Claude Code Sandbox (Reference Implementation)
- **Source**: https://github.com/matgawin/bubblewrap-claude
- **Retrieved**: 2026-05-12

## Project Overview

This Nix flake creates isolated execution environments for Claude Code using bubblewrap (bwrap) sandboxing. It provides a profile-based system for language-specific development while maintaining security boundaries.

## Isolation Mechanisms

**Process & Namespace Isolation:**
The project uses `--unshare-all` to achieve "complete namespace isolation," separating the sandbox from the host system's network, IPC, and process namespaces.

**Filesystem Restrictions:**
Only two directories remain writable: the project directory itself and `/tmp`. All other filesystem access is read-only or blocked, preventing unintended modifications to the host system.

**Network Control:**
HTTP proxy filtering restricts outbound connections to an allowlist of approved domains. This prevents Claude Code from accessing arbitrary internet resources while permitting necessary API communication.

## Mounted Filesystem Paths

The flake mounts several categories of paths:

- **Configuration:** Host's `~/.claude.json` automatically mounts if present
- **Language-specific caches** (read-only): Python pip/poetry caches, Rust cargo registries, Go module caches, Node package manager caches
- **System directories:** `/nix` for Nix operations (profile-dependent)
- **Custom binds:** Additional paths via profile-specific `args` parameter

## Nix Flake Architecture

The `mkSandbox` function creates sandbox packages accepting:
- `name`: Required identifier
- `packages`: Tools to include
- `env`: Environment variables
- `preStartHooks`: Runtime initialization commands
- `args`: Additional bwrap arguments
- `allowList`: Approved network domains
- `customPrompt`: System prompt for Claude Code behavior

Built-in profiles include base, Go, Python, Rust, C++, JavaScript, and DevOps configurations, each pre-configured with appropriate toolchains and cache bindings.

## Relevance to devenv Sandboxing

This project demonstrates a working pattern for wrapping Nix-provided tools in bubblewrap with:
- `--unshare-all` for full namespace isolation
- Selective bind mounts for project directory (rw) and system dirs (ro)
- Network filtering via proxy
- Profile-based configuration in Nix

The same pattern could be adapted to wrap `devenv shell` or individual devenv-provided binaries.
