# PR #2427: feat: run shell in a sandbox (Detailed)
- **Source**: https://github.com/cachix/devenv/pull/2427
- **Retrieved**: 2026-05-12

## Basic Information
- **Title:** feat: run shell in a sandbox
- **Author:** zaytsev
- **Status:** Draft (not merged)
- **Repository:** cachix/devenv
- **Creation Date:** January 23, 2026
- **Last Update:** February 21, 2026

## Core Description
The PR implements sandboxing for devenv shell processes using bubblewrap. Author states: "Use `bubblewrap` to spawn devenv shell process in an isolated sandbox. Early PoC just to get some feedback."

## Stated Limitations
- Tested on NixOS only
- Linux only, no macOS support

## Configuration Example
The implementation supports YAML configuration with mount options:
```yaml
sandbox:
  enable: true
  network:
    enable: true
  mounts:
    - path: /nix/store
    - path: /dev
      mode: dev
    - path: $HOME
      mode: overlay
```

## Key Technical Discussion

### Mount/Namespace Strategy
The sandbox isolates filesystem access through selective mounting:
- Mounts /nix/store for package access
- Mounts /dev with device mode
- Mounts /etc/passwd and /etc/group
- Mounts HOME with tmpfs or overlay modes
- Mounts /run/current-system/sw for system packages
- Enables optional network isolation

### Main Debate: Two Competing Approaches

**Bubblewrap Approach (This PR):**
- Sandboxes the entire shell process
- Isolates networking, IPC, and user namespaces
- Trades convenience (no zsh/starship) for security

**Landlock Approach (Alternative PR #1783):**
- Wraps individual executables from devenv packages
- Allows unsandboxed shell with access to user's full filesystem
- Targets supply chain attacks on specific tools

## Significant Comments & Discussion

**LorenzBischer** raised concerns about the approach:
- Shell loses zsh/starship customization
- Whole home directory must be whitelisted, reducing isolation benefit
- Questions whether sandboxing everything provides meaningful protection beyond the Nix packages themselves

**Zaytsev's Counter-Arguments:**
- Acknowledges trade-offs but values "peace of mind" against "malicious cargo build scripts"
- Notes wrapping every executable doesn't prevent bad actors from finding unwrapped versions
- Worries Landlock approach becomes fragile if PATH gets modified

**Key Quote from Zaytsev:** "I'd rather go with bubblewrap" despite acknowledging both solutions have merit.

## Technical Implementation Details

**Bubblewrap Capabilities Used:**
- Filesystem namespace isolation
- Device node mounting
- Overlay mounts for HOME
- Network namespace configuration
- User/IPC namespace support

**What Gets Isolated:**
- Filesystem write access outside mounts
- Network (optional)
- Process isolation

**What Remains Accessible:**
- Mounted /nix/store (all packages)
- User's home directory (with overlay mode)
- System binaries in /run/current-system/sw
- Dev nodes for I/O

## Why It Hasn't Merged

1. **Draft Status:** Author explicitly marked as WiP (Work in Progress)
2. **Unresolved Design Debate:** Competing with alternative Landlock approach
3. **Platform Limitations:** Linux/NixOS only - broader support needed
4. **UX Concerns:** Missing shell customization (starship) impacts developer experience
5. **No Reviewers:** PR shows "No reviews" section

## Recent Activity
Multiple force-pushes between January 30 - February 21 suggest ongoing refinement without consensus resolution.
