<!-- Source: https://unit42.paloaltonetworks.com/monitoring-npm-supply-chain-attacks/ -->
<!-- Retrieved: 2026-05-12 -->

# npm Threat Landscape: Attack Surface and Mitigations (Unit 42)

## Executive Summary

The npm ecosystem experienced a critical shift in September 2025 with the Shai-Hulud worm, transitioning from isolated nuisance attacks to "high-consequence threat landscape." Unit 42 has tracked systematic campaigns by threat actors to weaponize developer trust through supply chain compromises.

## Core Threat Evolution

Three major shifts characterize post-Shai-Hulud adversary tactics:

1. **Wormable propagation**: Payloads steal npm tokens and GitHub Personal Access Tokens (PATs) to automatically infect and republish legitimate packages
2. **Infrastructure persistence**: Attackers embed themselves in CI/CD pipelines for long-term, undetectable enterprise access
3. **Multi-stage payloads**: Dormant "sleeper" dependencies activate only under specific conditions to evade scanners

## Attack Vectors and Mechanisms

### Lifecycle Hook Exploitation

The primary infection vector uses npm lifecycle hooks. The malware modifies `package.json` with:
```
"preinstall": "node setup.mjs"
```

This executes automatically during installation before user awareness. Even if lifecycle scripts are blocked, secondary execution paths trigger when users invoke compromised binaries registered in the `bin` field.

### Credential Harvesting Scope

Malicious payloads target:
- npm tokens from `.npmrc` files
- GitHub authentication tokens and Actions secrets
- AWS STS identity, Secrets Manager, and SSM parameters
- Azure Key Vault and GCP Secret Manager credentials
- Kubernetes service account tokens
- Cryptocurrency wallets (Electrum)
- VPN configurations and Claude/MCP settings

### Propagation Mechanisms

#### npm Worm Self-Replication

Once malware obtains valid npm tokens with publish permissions:
- Downloads target package tarballs
- Injects malicious `setup.mjs` and `execution.js` files
- Modifies `package.json` with preinstall hooks
- Increments patch versions
- Republishes to npm registry

This creates exponential infection chains across developer ecosystems.

#### GitHub Dead Drop C2

The malware uses GitHub's public commit search API as a covert command channel:
- Searches for specific keywords in commit messages
- Decodes embedded tokens from Base64-encoded commit bodies
- Establishes new C2 infrastructure without attacker-controlled servers
- Exfiltrates stolen credentials to public repositories with randomized names

#### Obfuscation Techniques

Multi-layered obfuscation protects payloads:
- **String table rotation**: Hex indices resolve from rotated arrays
- **Seeded ASCII shuffle cipher**: Fisher-Yates shuffle with linear congruential PRNG seeded with 0x3039
- **Gzip and Base64 embedding**: Compressed payload blobs hide suspicious code
- **Mangled identifiers**: Variable names replaced with hex patterns

## Notable Attack Campaigns

### April 2026: Bitwarden Compromise

**@bitwarden/cli@2026.4.0** was backdoored by TeamPCP group, affecting Checkmarx distribution across:
- Docker Hub images
- GitHub Actions
- VS Code extensions

### Mini Shai-Hulud Wave (April 29, 2026)

Four SAP Cloud Application Programming (CAP) packages were compromised:
- @cap-js/sqlite@2.2.2
- @cap-js/postgres@2.2.2
- @cap-js/db-service@2.10.1
- mbt@1.2.48

Combined, these packages had ~570,000 weekly downloads.

## Defense and Mitigation Strategies

### Preventive Controls

**Cooldown Periods**: Block package versions published within 24-72 hours (most malicious packages are identified and removed within this window)

**Disable Lifecycle Scripts**: Configure `.npmrc` with `ignore-scripts=true` to prevent automatic preinstall/postinstall execution

**Version Pinning**: Use `package-lock.json` and `npm ci` instead of `npm install` in CI/CD pipelines

**Private Registry Proxying**: Route all npm traffic through internal registries rather than direct `registry.npmjs.org` access

**Namespace Shadowing Prevention**: Use scoped packages (@myorg/lib) and configure private registries to resolve internal scopes exclusively

**Provenance Verification**: Verify OpenID Connect attestations and use `slsa-verifier` to validate package provenance during builds

**Egress Filtering**: Restrict CI/CD runner network access to only private registries and approved deployment targets

**SBOM Generation**: Automatically create Software Bill of Materials for production releases enabling rapid impact analysis during zero-day announcements
