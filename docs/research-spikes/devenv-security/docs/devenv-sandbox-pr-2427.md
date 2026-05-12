# PR #2427: feat: run shell in a sandbox (cachix/devenv)
- **Source**: https://github.com/cachix/devenv/pull/2427
- **Retrieved**: 2026-05-12

## Overview
This draft PR introduces sandboxing for the devenv shell using bubblewrap to isolate the shell process. The implementation is early-stage and seeks community feedback.

## Technical Approach

**Technology**: Bubblewrap (bwrap) - containers that isolate the devenv shell execution environment through Linux namespaces.

**Scope of Sandboxing**: The entire shell process runs isolated, not individual executables. This differs from alternative approaches like Landlock.

## Current Limitations

- **Linux-only**: No macOS support
- **NixOS-tested**: Primary testing environment is NixOS
- **Shell features**: zsh with starship configuration unavailable within sandbox
- **Configuration required**: Users must explicitly whitelist filesystem paths

## Configuration Example

The sandbox accepts mount declarations specifying paths and access modes (standard, overlay, device, or temporary):

```
sandbox:
  enable: true
  network:
    enable: true
  mounts:
    - path: /nix/store
    - path: $HOME
      mode: overlay
```

## Key Design Discussion

**Zaytsev's Position**: Prefers comprehensive isolation protecting against supply chain attacks and malicious build scripts, accepting usability tradeoffs.

**LorenzBischer's Alternative** (PR #1783): Proposes wrapping individual Nix-provided executables with Landlock restrictions instead, keeping the user shell unrestricted and maintaining full filesystem access.

## Security Tradeoff Debate

- **Full sandbox**: Protects against compromised dependencies executing arbitrary commands
- **Per-executable sandbox**: Less overhead; requires explicit package declarations; user shell remains fully privileged

The community discussion suggests potential hybrid approaches combining both technologies.
