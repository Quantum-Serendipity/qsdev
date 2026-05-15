# GitHub Actions: attest-build-provenance

- **Source URL**: https://github.com/actions/attest-build-provenance
- **Retrieved**: 2026-05-15

---

## Overview

The `attest-build-provenance` action generates signed build provenance attestations for GitHub Actions workflow artifacts using the in-toto format and SLSA build provenance standards.

## Key Functionality

**Primary Purpose:** The action creates verifiable signatures binding artifacts to their build provenance by establishing a connection between a named artifact along with its digest and build metadata.

**Current Status:** As of version 4, this repository serves as a wrapper on top of the newer `actions/attest` action, with new implementations directed toward that action instead.

## Signing & Cryptography

**Certificate Generation:** The tool leverages short-lived signing certificates issued by Sigstore to authenticate attestations cryptographically.

**Environment-Based Signing:**
- Public repositories use Sigstore's public-good instance
- Private/internal repositories use GitHub's private Sigstore instance

## Attestation Storage & Verification

Signed attestations are uploaded to the GitHub attestations API and associated with the repository from which the workflow was initiated.

Users can verify attestations using the `attestation` command in the GitHub CLI:
```bash
gh attestation verify <artifact> --owner <owner>
```

## Availability Requirements

- Attestations are available in public repositories across all current GitHub plans
- Private/internal repositories require GitHub Enterprise Cloud for attestation functionality
- Legacy plans (Bronze, Silver, Gold) don't support this feature

## Important Note

As of v4, users should consult `actions/attest` for current usage documentation, as attest-build-provenance is now a wrapper around that action.
