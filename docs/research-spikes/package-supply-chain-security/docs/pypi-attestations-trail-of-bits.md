# Attestations: A New Generation of Signatures on PyPI
- **Source**: https://blog.trailofbits.com/2024/11/14/attestations-a-new-generation-of-signatures-on-pypi/
- **Retrieved**: 2026-05-12

## How They Work

PyPI attestations create a cryptographic chain linking package distributions to their provenance. The process integrates three key components:

**Trusted Publishing** establishes machine identities (e.g., GitHub workflows) that can publish packages via OpenID Connect, eliminating manual API tokens.

**Sigstore** accepts OIDC credentials and issues short-lived X.509 certificates binding ephemeral signing keys to these machine identities through its Fulcio certificate authority.

**Package signing** uses those ephemeral keys to create attestations that cryptographically bind a package's identity (filename and digest) to its production provenance.

"The certificate issued by Sigstore is bound to the Trusted Publishing identity, but it doesn't itself sign for the thing being published (i.e., the actual Python package distribution)."

## PEP 740 Specification

PEP 740 defines the complete attestation framework, including:
- Fixed attestation payload structure based on the in-toto Attestation Framework
- Storage mechanism via `provenance` keys in the JSON simple API
- "Provenance objects" containing rollups of attestation objects with Trusted Publisher identity verification information

## Consumer Verification

Currently, downstream verification remains incomplete. "It tells PyPI how to receive and verify attestations for its own purposes as well as how to redistribute them on the public index endpoints, but it doesn't mandate (or even define) a verification flow for installing clients."

Three user groups can verify attestations today:
- Researchers studying supply chain security
- Incident responders tracking artifacts to source
- Projects with complete build system control

## Adoption Statistics

- ~20,000 packages now produce attestations by default through Trusted Publishing
- 5% of the 360 most-downloaded packages have attestations
- Approximately two-thirds of top packages haven't released updates since attestation enablement became standard (October 29, 2024)

## Future Pip Integration

Trail of Bits is developing a plugin architecture for pip that would enable attestation verification during installation, with planned "trust on first use" identity tracking through standardized lockfiles (PEP 751).
