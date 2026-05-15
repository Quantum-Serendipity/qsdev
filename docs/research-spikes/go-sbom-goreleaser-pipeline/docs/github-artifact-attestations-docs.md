# GitHub Artifact Attestations Documentation

- **Source URL**: https://docs.github.com/en/actions/concepts/security/artifact-attestations
- **Retrieved**: 2026-05-15

---

## What Attestations Are

Artifact attestations enable creation of cryptographically signed claims that establish your build's provenance and include workflow links, repository details, commit SHA, and environment information from OIDC tokens.

## Information Included

Attestations capture:
- Workflow association with the artifact
- Repository, organization, environment, and commit details
- Triggering event information
- OIDC token data
- Optional software bill of materials (SBOM) for dependency transparency

## Sigstore Integration

GitHub uses Sigstore, an open source project for signing and verifying software artifacts via attestations. The implementation differs by repository type:

**Public Repositories:** Use the Sigstore Public Good Instance with a copy of the generated Sigstore bundle stored with GitHub and written to a publicly readable transparency log.

**Private Repositories:** Use GitHub's Sigstore instance (same codebase but without transparency logging) that federates only with GitHub Actions.

## SLSA Framework Compliance

Artifact attestations provide SLSA v1.0 Build Level 2, linking artifacts to build instructions. Achieving Level 3 requires using reusable workflows that many repositories across your organization share.

## Verification Process

Consumers use the GitHub CLI command `gh attestation verify ...` to verify attestations and evaluate source code and build instructions.

## Important Limitation

Artifact attestations are NOT a guarantee that an artifact is secure. They establish provenance links requiring consumers to define security policies and make informed risk decisions.

## Signing Recommendations

**Sign:** Released software, binaries, packages, and manifests with content hashes.

**Don't Sign:** Frequent test builds or individual files like source code or documentation.
