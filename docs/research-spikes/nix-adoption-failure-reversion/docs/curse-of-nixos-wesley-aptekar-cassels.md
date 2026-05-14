<!-- Source: https://blog.wesleyac.com/posts/the-curse-of-nixos -->
<!-- Retrieved: 2026-03-20 -->

# The Curse of NixOS

**Author:** Wesley Aptekar-Cassels
**Date:** January 2022
**NixOS Duration:** Approximately 3 years as sole OS on laptop

## The Core "Curse" Metaphor
The author describes NixOS as curse-like because while it demonstrates superior package management philosophy that makes alternatives unusable, it simultaneously burdens users with "extremely complicated constantly changing software" requiring configuration in a poorly-designed homegrown language.

## What NixOS Gets Right
The fundamental innovation is that software exists in a content-addressable store rather than installing globally. This enables:
- Multiple versions of the same package coexisting simultaneously
- Trivial rollbacks through symlink manipulation
- Easy zero-downtime deploys
- Running patched software versions without system corruption
- Prevention of bricking during updates (mirroring mobile A/B partitioning schemes)

## Major Criticisms

### 1. The Nix Language Problem
The homegrown configuration language is poorly designed and difficult to learn. Most users survive through copy-pasting examples until they need custom configurations, then become "completely high and dry." Documentation disconnects language learning from practical NixOS usage.

### 2. Lack of True Isolation
Software must be recompiled for NixOS. The system identifies dependencies through crude methods like "grepping a package for /nix/store/" rather than static analysis. Standard shebangs like #!/bin/bash fail without patching.

### 3. Secondary Issues
Patching packages is theoretically simple but practically annoying, and configurations exhibit "spooky action-at-a-distance" effects.

## Alternative Approaches Considered
The author prototyped combining Nix-store repositories with overlayfs filesystem layering, but overlayfs fails under heavy path overlaying. Silverblue's approach -- building container images for package compositions -- feels fundamentally unsatisfying despite enabling access to multiple distribution repositories.

## Continuing Despite Frustrations
Despite all criticisms, Aptekar-Cassels explicitly states: "I'm going to keep using it, since I can't stand anything else after having a taste of NixOS." However, he actively hopes for a successor that adopts NixOS's dependency philosophy while avoiding its implementation mistakes and improving user-friendliness.
