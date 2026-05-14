# Don't Rebuild Yourself: An Intro to Nix Package Caches — Jetify (Devbox)

- **Source**: https://www.jetify.com/blog/dont-rebuild-yourself-an-intro-to-nix-package-caches
- **Retrieved**: 2026-03-20

## Build Time Examples
- **MongoDB**: Building from source can take up to 30 minutes for a single package
- **Full closures**: Building a full closure from source can take hours

## 4-Layer Cache System (Devbox)
1. The Nix Store (`/nix/store`) — local cache
2. Jetify Private Cache — organization-level
3. Official Nix Cache (cache.nixos.org) — community
4. Jetify Prebuilt Cache — commercial prebuilts

## Cache Gap Problem
Public cache gaps exist for:
- Custom packages and flakes
- Licensed packages (MongoDB, Terraform, Vault)
- Older or less popular platform packages

These gaps mean some organizations will experience "hours" of build time on first build with no cache.

## Performance Claims
With Jetify Cache enabled, developers can "start reducing your installation times in seconds" — qualitative only, no specific metrics.

## Relevance
Demonstrates the importance of binary caches for Nix CI performance — without cache, Nix builds from source and can be extremely slow. The cache is the mechanism that enables speed, not Nix itself.
