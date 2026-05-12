# Lockfile Poisoning: Attack Vector Analysis

- **Source**: https://safedep.substack.com/p/lockfile-poisoning-an-attack-vector
- **Retrieved**: 2026-05-12

## Overview

Lockfile poisoning is a supply chain attack technique where malicious developers exploit the cognitive burden of reviewing auto-generated dependency files to introduce malware. The attack is particularly effective because platforms like GitHub and GitLab hide changes in auto-generated files by default.

## Technical Mechanism

**How Lockfiles Work:**
Package managers resolve semantic version constraints into specific versions and create lockfiles to ensure reproducible builds. For the npm ecosystem, `package-lock.json` stores the complete dependency resolution graph, including resolved URLs and integrity checksums.

**The Vulnerability:**
Unlike other package managers, npm stores "the resolved URL of a package artefact in the `package-lock.json` itself." This creates an attack opportunity because:

1. Reviewers face high cognitive load examining voluminous auto-generated files
2. Git hosting platforms obscure lockfile changes from default views
3. Both the artifact source URL and integrity information travel through the same channel

## Attack Vectors

The article identifies two critical abuse cases:

1. **Tampering with artifact URLs and checksums** - Redirects downloads to attacker-controlled malware, compromising production environments
2. **Introducing new dependency entries** - Executes malicious code during installation, potentially affecting CI/CD pipelines with access to sensitive data

## Detection and Mitigation

The `vet` tool implements framework-level detection by:
- Verifying package source URLs originate from trusted registries
- Validating URL paths match expected `node_modules` structures
- Currently supports npm; future ecosystems are planned

## Affected Ecosystem

The npm ecosystem faces heightened risk due to its unique architecture, though the vulnerability pattern is applicable to other package managers.
