# nix-sandbox: Bubblewrap + Nix CLI Sandboxing Tool
- **Source**: https://github.com/fabian-thomas/nix-sandbox
- **Retrieved**: 2026-05-12

## Purpose & Workflow

This tool creates isolated runtime environments for CLI applications using Bubblewrap and Nix. The primary use case involves running coding agents like OpenCode and Codex in constrained sandboxes, preventing unintended system access while maintaining project access.

## Core Architecture

**How it works:** The launcher combines two technologies -- Nix provides dependency management while Bubblewrap handles process isolation through Linux user namespaces and bind mounts.

## Filesystem Configuration

The sandbox mounts:
- Your current working directory at its original absolute path
- Nested `.git` directories as read-only (toggle with `--no-git-ro`)
- A minimal set of explicitly specified directories
- Optional custom shell environments via `NIX_SHELL_FILE`

The tool "mounts the entire parent directory of `NIX_SHELL_FILE` read-only into the sandbox so Nix can load that file."

## Isolation Mechanisms

**Isolated components:**
- Network (disable with `--no-net` for offline mode)
- Environment variables (minimal explicit configuration)
- Filesystem view (restrictive bind mount strategy)

**Not explicitly isolated:** PID namespace details aren't specified in the documentation.

## Key Limitations

The primary constraint involves unprivileged user namespace restrictions. On systems where `kernel.apparmor_restrict_unprivileged_userns=1`, Bubblewrap fails with "Permission denied" errors. Users must set this kernel parameter to `0` to enable the tool.

## Usage Examples

**Basic execution:**
```
./nix-sandbox python3 -c "import yaml; print('ok')"
```

**Ad-hoc packages:**
```
./nix-sandbox --pkgs "git ripgrep fd" bash -lc "rg --version"
```

**Custom environment:**
```
NIX_SHELL_FILE="path/to/environment.nix" ./nix-sandbox -- python3
```

The codebase uses 87.6% Shell and 12.4% Nix -- reflecting its scripted launcher design with Nix-based configuration.
