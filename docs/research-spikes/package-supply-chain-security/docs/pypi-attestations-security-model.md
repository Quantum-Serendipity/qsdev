# PyPI Attestations Security Model
- **Source**: https://docs.pypi.org/attestations/security-model/
- **Retrieved**: 2026-05-12

## What Attestations Guarantee

Attestations provide two core guarantees:

1. **Origin verification**: They confirm a package came from an authorized publisher (e.g., a specific GitHub repository via CI/CD) without post-build modification.

2. **Change detection**: They enable observers to notice when a project's Trusted Publisher changes, potentially indicating malicious takeover.

The mechanism uses "keyless" signing where an identity is "cryptographically bound to a short-lived signing key via an OpenID Connect (OIDC) operation."

## What Attestations Don't Guarantee

Critically, attestations do **not** establish trustworthiness. As the documentation states: "An attestation will tell you **where** a PyPI package came from, but not **whether** you should trust it."

A valid attestation proves "proof of access to that identity while the package was built" but provides no assurance that the identity itself is trustworthy or that malicious code was injected during development.

## Verification Architecture

Verification relies on:

- **Trusted Publishers**: Identity-based authorization (Sigstore integration)
- **Fulcio CA**: Issues short-lived X.509 certificates binding identities to ephemeral keys
- **Certificate Transparency log**: Auditable record of certificate issuance
- **Rekor**: "Artifact transparency log, effectively recording every signing event"

PyPI requires both Rekor and CT log inclusion proofs for verification.

## Limitations

The documentation does not address:
- How end-users check attestations via pip or other tools (still under development)
- The percentage of PyPI packages with attestations
