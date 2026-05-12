# Supply-Chain Guardrails for npm, pnpm, and Yarn

- **Source**: https://www.coinspect.com/blog/supply-chain-guardrails/
- **Retrieved**: 2026-05-12

## Overview

This article from Coinspect Security provides practical mitigation strategies against supply chain attacks affecting JavaScript package managers. The guidance addresses both package maintainers and consumers of dependencies.

## For Package Publishers

Key protective measures include:

- **Authentication hardening**: "Use hardware security keys (FIDO2/WebAuthn) instead of TOTP" rather than time-based codes, which are more vulnerable to phishing
- **Trusted publishing**: Adopt OIDC-based trusted publishing to replace long-lived API tokens with identity-verified build credentials
- **Token management**: Migrate from legacy tokens to granular, short-lived alternatives with minimal required permissions
- **Enforcement policies**: Make two-factor authentication mandatory across maintainers and contributors
- **Workflow security**: Avoid risky GitHub Actions triggers like `pull_request_target` on untrusted contributions; restrict permissions and limit publishing to protected branches

## For Dependency Consumers

The article emphasizes several defensive practices:

**Version pinning and lockfiles:**
- Avoid semantic versioning ranges (`^` or `~`) in package.json
- Commit lockfiles (`package-lock.json`, `pnpm-lock.yaml`, `yarn.lock`) to freeze entire dependency trees
- Use strict installation modes that fail if lockfiles drift from manifests

**Script execution controls:**
- Treat lifecycle hooks (postinstall, prepare, etc.) as untrusted code
- Disable scripts by default in CI environments and enable selectively
- Environment variable configuration: `NPM_CONFIG_IGNORE_SCRIPTS=true` for npm

**Timing and review:**
- Delay adoption of newly published packages by several days, allowing community detection of malicious versions
- Review lockfile diffs before merging to catch unexpected transitive dependencies

**Token security in CI/CD:**
- Use identity-based access (OIDC) when available instead of injecting long-lived tokens
- Restrict tokens to read-only access and specific registries when identity methods unavailable

**Typosquatting prevention:**
- Verify package names against official documentation before installation

## Package Manager-Specific Guidance

**npm:**
- Use `npm ci` instead of `npm install` in CI/CD
- Configure `.npmrc` with `save-exact=true`
- Set `NPM_CONFIG_IGNORE_SCRIPTS=true` or add `ignore-scripts=true` to `.npmrc`

**pnpm:**
- Run with `pnpm install --frozen-lockfile`
- Lifecycle scripts disabled by default in CI
- Version 10.16+ supports `minimumReleaseAge` setting for 60-day cooldown periods on new releases

**Yarn:**
- Yarn v1: `yarn install --frozen-lockfile`
- Yarn v2+ (Berry): `yarn install --immutable`
- Configure `.yarnrc` with `save-exact=true` (v1) or `.yarnrc.yml` with immutability settings (v2+)
- Disable scripts with `yarn install --ignore-scripts` in CI

## Forward-Looking Approach

The article notes that current mitigations address common threats but acknowledges limitations. Future resilience requires moving toward systematic verification and isolation through sandboxing and the Principle of Least Privilege, which would restrict each dependency's file system, network, and module access.
