# GitHub Actions: attest-sbom

- **Source URL**: https://github.com/actions/attest-sbom
- **Retrieved**: 2026-05-15

---

## Overview

The `actions/attest-sbom` action generates signed SBOM (Software Bill of Materials) attestations for workflow artifacts. It binds some subject (a named artifact along with its digest) to a Software Bill of Materials using the in-toto format.

## Key Information

**Status:** This action is being deprecated in favor of `actions/attest`. The documentation notes that attest-sbom will continue to function as a wrapper on top of attest for some period of time, but applications should make plans to migrate.

**Supported SBOM Formats:**
- SPDX (JSON-serialized)
- CycloneDX (JSON-serialized)

**Signing Method:** Uses short-lived Sigstore-issued signing certificates. Public repositories use Sigstore's public-good instance, while private/internal repositories use GitHub's private Sigstore instance.

**Attestation Storage:** Once created and signed, attestations are uploaded to the GitHub attestations API and associated with the repository initiating the workflow.

**Verification:** Attestations can be verified using the attestation command in the GitHub CLI:
```bash
gh attestation verify <artifact> --owner <owner>
```

## Version 4 Changes

As of version 4, this action functions as a wrapper around `actions/attest`. Users should reference the `actions/attest` repository for current usage documentation.

## Availability

Artifact attestations require specific GitHub plans -- available in public repositories on all current plans, but only in private/internal repositories with GitHub Enterprise Cloud.
