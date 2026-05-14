<!-- Source: https://dennybritz.com/posts/adopting-nix/ -->
<!-- Retrieved: 2026-03-20 -->

# Adopting Nix

**Author:** Denny Britz
**Date:** January 7, 2023

## Context
The author refactored build processes and CI pipelines for multiple projects that previously used Makefiles, Dockerfiles, and Github artifacts, hoping Nix would consolidate these disparate tools.

## What Worked

### Reproducible CI
Running identical commands locally and in GitHub Actions eliminated trial-and-error deployments. The author notes that "if it builds locally, it should build in CI."

### Docker Image Generation
Nix builds Docker images without Docker itself, leveraging cached dependencies from /nix/store for speed. After building with nix build, generating images took only seconds rather than 10-15 minutes.

### Reduced Code Duplication
A single .nix file replaced overlapping Makefiles, Dockerfiles, and CI scripts. Cross-project dependencies became manageable through flakes without requiring GitHub artifact authentication.

### Implicit Dependency Resolution
Nix eliminated failures from mismatched glibc versions between Ubuntu versions -- a recurring production problem.

## Major Challenges

### Language Design
The Nix language feels "a bit like taking a purely functional language Haskell, removing its beautiful type system, and then mixing in some Javascript." Poor tooling and editor integration compound this issue.

### Documentation Fragmentation
Official manuals are outdated and scattered. The author recommends nix-pills as the best starting resource, yet notes that "the best resources for adopting Nix are actively maintained examples" found outside official channels.

### Flakes Uncertainty
While flakes simplify dependency declaration, the community remains divided on adoption, causing documentation to split between flake and non-flake approaches. Most official documentation omits flakes entirely.

### Package Manager Integration
Tools like crane and pip2nix remain early-stage with unresolved bugs, forcing workarounds.

### macOS Limitations
Sandboxing differences make macOS support significantly weaker than Linux, with remote builders requiring painful manual setup.

## Outcome
The article doesn't explicitly state whether the author continued using Nix long-term, but the detailed troubleshooting suggests pragmatic adoption with reservations about ecosystem maturity.
