<!-- Source: https://docs.npmjs.com/generating-provenance-statements/ -->
<!-- Retrieved: 2026-05-12 -->

# npm Provenance with Sigstore

## How Provenance Statements Work

npm provenance establishes supply-chain security by generating two attestation types:

1. **Provenance Attestation**: Publicly provides a link to a package's source code and build instructions from the build environment, enabling verification before download.

2. **Publish Attestation**: Registry-generated signatures created when authorized users publish packages.

When packages are published with provenance, they are signed by Sigstore public good servers and logged in a public transparency ledger.

## Sigstore's Role

Sigstore is a toolkit enabling ephemeral certificate signing. Its architecture includes:

- **Certificate Authority**: Federates with any OIDC provider that includes verifiable build information, verifying token integrity and issuing signing certificates containing build data.

- **Transparency Log (Rekor)**: Provides a public, verifiable, tamper-evident ledger of signed attestations, detecting tampering attempts if registry compromise occurs.

## Supported CI/CD Platforms

Currently supported providers:
- GitHub Actions
- GitLab CI/CD

Both require cloud-hosted runners to establish provenance.

## Key Prerequisites

- npm CLI version 9.5.0 or later
- Public repository configured in `package.json` (case-sensitive match required)
- Cloud CI/CD automation with supported provider
- OIDC token write permissions

## Verification Method

Verify provenance using: `npm audit signatures`

This command returns the count of verified registry signatures and verified attestations for all of the packages in a project.

## Important Limitations

Provenance does not guarantee the package has no malicious code. Instead, npm provenance provides a verifiable link to the package's source code and build instructions, which developers can then audit and determine whether to trust it or not.

## Additional Details (from related sources)

### Cosign Verification

Cosign v2.4.0 release allows verification of attestations in the bundle format used by npm provenance, GitHub Artifact Attestations, and Homebrew provenance.

### npm audit signatures --json

As of March 2026, `npm audit signatures` verifies sigstore attestation bundles but only reports pass/fail counts, with no way to extract the actual attestation bundles for downstream consumption. A feature request exists to add a `--include-attestations` flag for outputting raw sigstore attestation bundles in JSON format.

### Trusted Publishing

When publishing using trusted publishing from GitHub Actions or GitLab CI/CD, npm automatically generates and publishes provenance attestations for your package.
