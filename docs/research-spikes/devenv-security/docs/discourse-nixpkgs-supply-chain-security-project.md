# NixOS Discourse: Nixpkgs Supply Chain Security Project
- **Source**: https://discourse.nixos.org/t/nixpkgs-supply-chain-security-project/34345?page=2
- **Retrieved**: 2026-05-12

## Project Goals

The initiative aims to create a centralized vulnerability tracking system for Nixpkgs. The core objective is to help security teams and package maintainers identify vulnerable packages in distribution channels and prioritize remediation efforts, enabling users to access secure system upgrades.

## Proposed Measures

**Three-Phase Vulnerability Workflow:**

1. **Initial Triaging**: Security team members review pre-computed CVE-to-derivation matches, filtering suggestions for further inspection with the goal of reaching "inbox zero."

2. **Draft**: Teams adjust automatic matches, correct duplicates, and verify which derivations are affected before publication.

3. **Mitigation**: Approved records publish as GitHub issues, pinging affected maintainers, with automatic archival upon closure.

## Implementation Status

**As of June 2025**, the project has progressed significantly:

- Basic triaging workflow completed with optimization improvements
- 17 user stories and 30+ additional issues addressed
- Publishing functionality implemented to create GitHub issues automatically
- Deployment to production infrastructure underway (expected late June/early July 2025)
- CPE metadata work underway to improve vulnerability discoverability

## Additional Security Initiatives

Related work includes:
- Automating commit-bit lifetime management for contributor access
- Security reviews of core packages
- Development of local scanning tools like the nix-local-security-scanner

## Community Reactions

Testers have reported the tracker already identifies vulnerable packages missed by other methods. Feedback emphasizes needing intermediate workflow stages, noise reduction in suggestions, and RSS feed support for users.
