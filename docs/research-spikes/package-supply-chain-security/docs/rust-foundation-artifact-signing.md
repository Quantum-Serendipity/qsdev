# Rust Foundation: Improving Supply Chain Security Through Artifact Signing
- **Source**: https://rustfoundation.org/media/improving-supply-chain-security-for-rust-through-artifact-signing/
- **Retrieved**: 2026-05-12

## What's Planned

The Rust Foundation aims to implement cryptographic verification for Rust releases and crates through a three-part approach:

**Index Signing**: Rather than immediately signing individual crate files, the foundation will sign the crate index (both sparse and Git formats). Since the index already contains SHA-256 checksums, this approach provides security guarantees while reducing complexity. "Our plan is to use another delegated certificate to sign each index entry along with the index as a whole."

**Release Signing**: The team will create delegated certificates within a new PKI to sign release artifacts, enabling `rustup` to verify signatures without relying solely on system certificate stores.

## Infrastructure Being Built

The foundation is establishing public key infrastructure (PKI) to manage certificates for signing, including protocols for delegation, rotation, and revocation. This PKI will be managed by the Rust Project's infrastructure team with Rust Foundation support.

## Timeline

The document indicates work began immediately as of December 2023, with RFCs planned in sequence: first the foundational PKI RFC, then RFCs for release and crates components thereafter.

## Sigstore Relationship

The document makes no mention of Sigstore or any relationship to it.

## Current Status

The article (December 21, 2023) represents an announcement of intentions rather than completed implementation, with actual development dependent on RFC reviews and community feedback.
