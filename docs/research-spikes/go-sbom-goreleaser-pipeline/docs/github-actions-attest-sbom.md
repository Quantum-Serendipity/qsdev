<!-- Source: https://github.com/actions/attest-sbom -->
<!-- Retrieved: 2026-05-15 -->

# GitHub Actions attest-sbom

## Core Purpose
The `actions/attest-sbom` action generates signed SBOM attestations for workflow artifacts using the in-toto format. It accepts SBOMs in SPDX or CycloneDX JSON-serialized format.

## Key Functionality
- **Attestation Creation**: Binds artifacts (with digests) to Software Bill of Materials documentation
- **Signing**: Uses short-lived Sigstore-issued certificates for verifiable signatures
- **Repository Handling**: Public repositories use Sigstore's public-good instance; private/internal repos use GitHub's private Sigstore instance
- **Storage**: Attestations are uploaded to the GitHub attestations API and associated with the originating repository

## Important Notice
The action is being deprecated in favor of `actions/attest`. The page notes that "actions/attest-sbom will continue to function as a wrapper on top of `actions/attest` for some period of time," though existing inputs remain compatible.

## Verification
Attestations can be verified using the `attestation` command in the GitHub CLI:

```bash
gh attestation verify <artifact> --owner <org>
```

## Availability
Artifact attestations are available in public repositories across all current GitHub plans, but require GitHub Enterprise Cloud for private/internal repositories.
