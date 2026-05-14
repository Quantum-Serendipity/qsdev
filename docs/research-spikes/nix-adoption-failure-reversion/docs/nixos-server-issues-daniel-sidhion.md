<!-- Source: https://sidhion.com/blog/nixos_server_issues/ -->
<!-- Retrieved: 2026-03-20 -->

# NixOS is a good server OS, except when it isn't

**Author:** Daniel Sidhion
**Date:** March 27, 2024 (updated January 1, 2025)

## Core Context
Sidhion investigated reducing NixOS system sizes for server deployments, particularly for microVMs and worker machines. His default minimal headless configuration consumed approximately 900MB despite using minimal and headless profiles -- substantially larger than Alpine Linux's ~210MB footprint.

## Key Pain Points & Solutions

### 1. Nixpkgs Source Inclusion (~179MB)
The system bundled a complete Nixpkgs copy for flake registry management. Setting nix.enable = false eliminated this overhead, as deployment systems don't require runtime Nix availability. Result: 733MB.

### 2. Perl & Python Dependencies (~242MB)
"Python only comes in because of install-systemd-boot.sh" and Perl enabled activation scripts. Adopting the perlless profile removed both. Sidhion noted this prevented runtime configuration switching but accepted this tradeoff for immutable deployments. Result: 491MB.

### 3. Duplicate systemd Packages (~14MB)
Both systemd and systemd-minimal existed simultaneously. An overlay redirected dbus to use full systemd instead. Result: 477MB.

### 4. Unnecessary Subsystems (~30MB)
Disabling udev, LVM, sudo, and security wrapper modules removed unneeded infrastructure. However, this required disabling security wrappers entirely via disabledModules and providing dummy options.

## What Didn't Work
Sidhion encountered substantial obstacles attempting further reductions:
- Infinite recursion errors when overlaying fuse packages
- Systemic circular dependencies in Nixpkgs
- Hardcoded binaries throughout default modules
- Multiple "-minimal" package variants with conflicting dependencies

He documented unresolved issues: kernel optimization, locale removal, coreutils/util-linux filtering, and redundant utility binaries.

## Final Assessment
Abandoned NixOS for ultra-minimal servers. Sidhion concluded: "trying to mold NixOS into the shape I wanted just isn't the way to go." He achieved ~300MB reductions on existing servers but determined creating a server-focused NixOS fork or pursuing containers would be more practical than continued core library modifications.

## Recommendation
NixOS excels at server configuration management but assumes interactive OS defaults unsuitable for minimal deployments. The effort required to achieve container-equivalent leanness exceeds practical value without architectural redesign.
