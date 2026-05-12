# npm Supply Chain Security Defenses in 2026

- **Source**: https://mondoo.com/blog/npm-supply-chain-security-package-manager-defenses-2026
- **Retrieved**: 2026-05-12

## Overview

The article examines how package managers protect JavaScript ecosystems from supply chain attacks like the Shai-Hulud worm, which "compromised 796 packages with 132 million monthly downloads." Security operates on two fronts: publisher-side (registry protections) and consumer-side (installation protections).

## Publisher-Side Defenses (npm Registry)

- **Trusted Publishing & Provenance**: Packages built via GitHub Actions OIDC tokens with Sigstore-signed attestations achieve "SLSA Build Level 2" status
- **Granular Access Tokens**: Time-limited, scoped credentials replace all-or-nothing tokens
- **Mandatory 2FA**: Required for high-impact package maintainers

## Consumer-Side Defenses

### pnpm v11 Three-Layer Approach

**1. Lifecycle Script Blocking**
`strictDepBuilds: true` (enabled by default since late 2025) blocks preinstall and postinstall scripts. An allowlist approach requires explicit configuration:

```yaml
allowBuilds:
  esbuild: true
  sharp: true
```

Packages not listed cannot execute code during installation.

**2. Release Cooldown**
`minimumReleaseAge` defaults to 1440 minutes (one day), preventing installation of recently published versions. This "would have blocked both" the Shai-Hulud and debug/chalk attacks, which were detected and removed within hours.

**3. Trust Policy**
`trustPolicy: no-downgrade` detects when packages are published with weaker authentication than previous versions, blocking installations where trust levels decrease.

## Comparison Across Package Managers

| Feature | pnpm v11 | npm CLI | Yarn Berry | Bun |
|---------|----------|---------|-----------|-----|
| Script blocking (default) | Yes | No | Yes | Yes |
| Per-package allowlist | Yes | No | Yes | Yes |
| Release cooldown (default) | Yes (1 day) | No | Yes (3 days) | Opt-in |
| Trust/provenance enforcement | Opt-in | Manual only | No | No |

**npm CLI limitations**: npm offers only a blunt `--ignore-scripts` flag with "no per-package allowlist." Users cannot selectively enable scripts for trusted dependencies.

**Yarn Berry advantages**: Implements `enableHardenedMode` (enabled by default on GitHub PRs) for lockfile validation against the registry, protecting against poisoning attacks.

## Defense-in-Depth Strategy

"Neither layer alone is sufficient." When bypassing one control (e.g., adding React to release-cooldown exceptions for a critical patch), other layers remain active:
- Lifecycle script blocking prevents injected build scripts
- Trust policy verifies publication through legitimate CI/CD pipelines

## Practical Implementation Recommendations

- **pnpm v11 users**: Review `allowBuilds` mappings and opt into `trustPolicy: no-downgrade`
- **Yarn Berry users**: Verify `enableHardenedMode` is active and audit `dependenciesMeta` exceptions
- **npm CLI users**: "the fewest consumer-side protections available" — should "evaluate whether switching to a package manager with stronger defaults is practical"
- **All teams**: Document every security control exception with rationale for an audit trail
