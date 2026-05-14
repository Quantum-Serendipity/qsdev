<!-- Source: https://dev.to/sleeyax/why-i-stopped-using-nixos-and-went-back-to-arch-4070 -->
<!-- Retrieved: 2026-03-20 -->

# Why I Stopped Using NixOS and Went Back to Arch Linux

**Author:** Sleeyax
**Date:** March 7, 2025
**NixOS Duration:** Nearly 1 year (installed May 17, 2024)

## Setup Details
- NixOS on laptop; Arch Linux on desktop
- Using nixos-unstable branch with flakes, home-manager, and cachix

## What They Liked About NixOS
The author appreciated the conceptual promise: "define your whole system configuration in a special config file syntax (Nix) and as output it gives you a reproducable build of your whole operating system."

## Major Pain Points Identified

### 1. Frequent System Breakage
Systems broke constantly before updates. The author experienced an endless "rebuild > fix > rebuild" cycle, with random component failures (audio, Bluetooth, Electron apps) after reboots with no clear cause.

### 2. Cryptic Error Messages
Error outputs were verbose but unhelpful, burying critical information like "logseq has been removed" under layers of stack traces incomprehensible to non-core developers.

### 3. Massive Update Sizes
NixOS keeps multiple package versions alongside old ones for rollback capability. Example: "glibc-locales" had six different versions stored simultaneously, causing severe disk bloat.

### 4. Compilation Overhead
Regular maintenance updates took 4-5+ hours due to compiling from source. Binary caches (cachix) frequently missed packages, forcing unnecessary local rebuilds.

### 5. Poor Documentation
"Better answers on the Arch Wiki than on the NixOS Wiki itself." Documentation was vague, outdated, abstract, with scarce practical examples and missing crucial details.

## Why They Left
Compilation times became the breaking point -- the constant 4-5 hour rebuilds made the system impractical for daily use despite appreciating the rollback concept.

## What They Switched To
Arch Linux on their laptop, matching their desktop setup.

## Suggested Alternatives for NixOS Features
- **Generations:** BTRFS snapshot volumes
- **Declarative packages:** aconfmgr
- **Configuration management:** Git-synced dotfiles
- **Development environments:** Docker/Podman instead of Nix flakes
