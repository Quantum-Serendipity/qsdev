# nix-mineral: NixOS Security Hardening Module
- **Source**: https://github.com/cynicsketch/nix-mineral
- **Retrieved**: 2026-05-12

## Overview
nix-mineral is a NixOS module designed for "convenient system hardening" that operates as a drop-in addition to existing NixOS systems. The project emphasizes that it serves defense-in-depth purposes only and is not a substitute for core security practices like proper OPSEC or sandboxing.

## Hardening Categories

The module covers several security domains:

**Filesystem Protection**: Implementation of systemd-tmpfiles configuration and restrictive mount options to secure file storage and access.

**Kernel Hardening**: Extensive use of sysctl parameters and boot configuration to strengthen kernel-level protections.

**Network Security**: Sysctl-based hardening and configuration of network-related services to reduce exposure.

**Attack Surface Reduction**: A comprehensive blacklist of kernel modules to minimize potential entry points.

**System Entropy**: Hardening measures focused on random number generation quality.

## Configuration Approach

Users configure the module through three methods:

1. **Fetchgit method**: For non-flake systems using automatic retrieval
2. **Flakes integration**: Preferred approach allowing version pinning and updates
3. **Options customization**: Setting specific parameters via `nix-mineral.settings`

## Key Limitations

- **Alpha Status**: The project explicitly warns that data loss or functionality issues may occur
- **nixos-unstable dependency**: Target systems should run unstable releases; compatibility issues may arise with renamed options
- **Threat Model**: Assumes non-state adversaries; anonymity is not considered
- **No UX/Architecture Changes**: Cannot perform complete system overhauls compared to dedicated security-focused operating systems

## Compatibility Notes

Users likely need to adjust options for their specific hardware and software. The module requires manual override of incompatible options when system configurations change across releases.
