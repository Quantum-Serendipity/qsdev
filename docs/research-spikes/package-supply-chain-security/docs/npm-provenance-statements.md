# npm Provenance Statements
- **Source**: https://docs.npmjs.com/generating-provenance-statements/
- **Retrieved**: 2026-05-12

## What Is npm Provenance?

npm provenance establishes supply-chain security by creating verifiable records of where and how packages were built. The system generates two types of attestations:

1. **Provenance attestation**: Links to source code and build instructions
2. **Publish attestation**: Registry-generated signature when packages are published

Sigstore is used for short-lived, ephemeral certificates to sign software.

## Supported CI/CD Systems

Only two cloud-hosted CI/CD platforms currently support provenance:
- GitHub Actions
- GitLab CI/CD

Self-hosted runners and other platforms are not supported.

## Requirements to Publish with Provenance

Publishers must meet these prerequisites:
- npm CLI version 9.5.0 or later
- Public repository URL in `package.json` (case-sensitive match)
- Cloud-hosted CI/CD automation
- Review of the Linux Foundation Immutable Record notice

## Required Permissions and Flags

**GitHub Actions workflow needs:**
- `permissions: id-token: write`
- `runs-on: ubuntu-latest`
- `npm publish --provenance` flag
- For first-time publishes: `npm publish --provenance --access public`

**GitLab CI/CD needs:**
- `id_tokens` with `SIGSTORE_ID_TOKEN` configured
- Same npm publish command with `--provenance` flag

## Alternative Configuration Methods

If using third-party publishing tools that don't invoke `npm publish` directly, configure provenance through:
- Environment variable: `NPM_CONFIG_PROVENANCE=true`
- `package.json` entry: `"publishConfig": { "provenance": true }`
- `.npmrc` file: `provenance=true`

## Verification by Consumers

Users verify provenance using: `npm audit signatures`

This command reports verified registry signatures and attestation counts across project dependencies.

## Key Limitations

- "When a package in the npm registry has established provenance, it does not guarantee the package has no malicious code."
- Provenance only provides "a verifiable link to the package's source code and build instructions, which developers can then audit and determine whether to trust it or not."
- Developers must still conduct their own security review despite provenance presence.

## Trusted Publishing

When using trusted publishing (OIDC-based), "provenance attestations are automatically generated for your packages without requiring the `--provenance` flag," eliminating the need for access tokens in CI/CD workflows.
