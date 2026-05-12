# PR #1783: Implement Transparent Sandboxing (Landlock)
- **Source**: https://github.com/cachix/devenv/pull/1783
- **Retrieved**: 2026-05-12

## Basic Information
- **Title:** "implement transparent sandboxing"
- **Author:** LorenzBischof
- **Status:** Draft (still in progress)
- **Repository:** cachix/devenv
- **Branch:** LorenzBischof:push-uvwrmtustnww -> cachix:main

## Timeline
- **Created:** March 19, 2025
- **Last significant activity:** February 9, 2026
- **Current status:** Incomplete, author lacks capacity for development

## Core Concept

The proposal aims to transparently sandbox all processes, packages, tasks, and scripts to the current directory without requiring container-based development. LorenzBischof noted: "My goal was to enable transparent sandboxing. One main advantage of devenv is that development is not within a container, meaning everything else on the host system stays available."

## Technical Implementation

### Approach
The PR implements basic filesystem sandboxing using **Landlock LSM** (Linux Security Module). The implementation is "best-effort" and gracefully disables itself on unsupported systems.

### How It Works
- Restricts filesystem access to specified directories
- Confines processes to the development environment's closure (Nix dependencies)
- Sandboxing is transparent to the user during normal development

### Limitations
1. **Linux-only:** Currently only works on Linux with supported kernels
2. **macOS incompatible:** Historical sandboxing mechanisms like `sandbox_init` are deprecated
3. **Landlock constraints:** Restrictions can only increase, never decrease -- incompatible with dynamic enable/disable requirements
4. **direnv integration:** Challenges exist with directory environment manager compatibility

## Evolution and Alternative Approaches

LorenzBischer created a separate proof-of-concept called **peninsula** (inspired by island) designed to support direnv through seccomp-unotify mechanisms, though this adds complexity.

## Why It Remains Draft

The author stated: "This definitely needs more work and was just an experiment. I wont have any time to develop or think about this until next year."

## Community Reception

Contributors expressed interest. Gabyx noted the advantage of avoiding container nesting problems in CI/CD environments, preferring the lightweight sandboxing alternative.
