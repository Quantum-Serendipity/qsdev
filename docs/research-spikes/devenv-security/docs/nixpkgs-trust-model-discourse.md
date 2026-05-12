<!-- Source: https://discourse.nixos.org/t/trust-model-for-nixpkgs/9450 -->
<!-- Retrieved: 2026-05-12 -->

# Trust Model for Nixpkgs: Discussion Summary

## Core Trust Concerns

The original poster's organization questioned whether nixpkgs could be trusted compared to Debian packages backed by Canonical. The concern centered on who vets packages and what mechanisms ensure security.

## Committer Access and Governance

**Current Structure:**
- Approximately 139 people have commit access to the master branch
- Access is granted by GitHub organization owners (including @edolstra and @domenkozar)
- The committer list is not public, though one committer suggested it should be

**Key Limitations:**
The master branch lacks protection requirements. As one contributor noted, "We don't have the CI set up well enough to protect the branch from all use cases currently."

## Verification Mechanisms

Several technical advantages distinguish nixpkgs from traditional package management:

1. **Reproducibility:** Build processes are transparent and auditable through source code reconstruction
2. **Pinning:** Users can freeze specific nixpkgs versions and audit only approved changes
3. **Isolation:** Nix builds operate in sandboxed environments without network access
4. **Source Verification:** Complete dependency chains and tools used in production are documented and hashable

One participant noted: "you can turn off the nixos.org cache, which will build it from source. That really is the holy grail of a distribution."

## Identified Weaknesses

- Not all packages in nixpkgs achieve reproducible builds
- Committers can merge changes directly without PR review
- No formal security policy or certification process exists
- Bootstrap tool verification remains incomplete

## Organizational Comparison

Participants noted that trusting Canonical differs less than assumed from trusting volunteer-led Debian—both rely on committer integrity rather than institutional vetting.
