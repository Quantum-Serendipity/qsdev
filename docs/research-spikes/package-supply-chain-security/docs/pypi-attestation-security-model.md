<!-- Source: https://docs.pypi.org/attestations/security-model/ -->
<!-- Retrieved: 2026-05-12 -->

# PyPI Attestation Security Model

## Core Purpose
Attestations assert facts about Python packages, primarily that "the project was published by an authorized publisher such as a particular CI provider in a particular code repository." They serve two main functions: protecting against post-build modifications and enabling observation of changes to Trusted Publishers.

## What Attestations Verify vs. Don't Verify

**What they verify:** A valid attestation demonstrates that a package came from a specific identity without modification after building. The signature proves "access to that identity while the package was built."

**Critical limitation:** Attestations do **not** convey trustworthiness. As the documentation emphasizes, "a valid signature does **not** tell the verifying party...whether they should trust the identity that holds the key." They cannot verify whether malicious code was introduced before or during the build process.

The distinction matters practically: if a package has an attestation from `pypa/sampleproject`, users know where it came from but must independently decide whether to trust that source.

## Trust Model

The framework creates a temporal trust dimension. Users can establish baseline trust by accepting the first identity seen for a package name, then detect suspicious changes automatically if subsequent releases come from different sources or lack attestations entirely.

## Sigstore Integration

PyPI leverages "keyless" signing through Sigstore, replacing long-lived keys with identity-based short-lived credentials:

1. **OIDC token issuance:** The publishing actor receives an OIDC token from its identity provider
2. **Certificate binding:** This token goes to Fulcio (Sigstore's CA) along with an ephemeral public key; Fulcio issues an X.509 certificate binding the OIDC claims to that public key
3. **Signing:** The ephemeral private key signs artifacts, then gets discarded; the certificate permanently binds its public counterpart to the verified identity

This approach trades direct key management for reliance on Sigstore as an intermediary, similar to how HTTPS trusts certificate authorities.

## Security Considerations

**Transparency mechanisms reduce Fulcio trust:**
- A Certificate Transparency log records all issued certificates, enabling audits of whether Fulcio issued credentials only for authentic identities
- Rekor logs all signing events with Fulcio certificates, creating "artifact transparency"
- Both mechanisms make Fulcio's actions "cryptographically auditable and verifiable"

**Verification requirements:** PyPI requires attestations to include inclusion proofs from both Rekor and Fulcio's CT log to be considered verified.

**Trusted Publishing alignment:** Since attestations depend on Trusted Publishers, all associated security considerations apply -- they're "more secure and misuse-resistant than a password or long-lived API token" but aren't substitutes for controlling who triggers publishing workflows.
