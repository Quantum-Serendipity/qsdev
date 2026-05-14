<!-- Source: https://docs.sigstore.dev/about/overview/ -->
<!-- Retrieved: 2026-05-14 -->

# Sigstore Architecture Overview

## Core Components

Sigstore comprises three primary technical modules working together:

**Cosign** (Client): A Sigstore client that initiates the signing workflow by creating ephemeral public/private key pairs and managing artifact signatures.

**Fulcio** (Certificate Authority): The code-signing certificate authority that processes signing requests. It verifies OpenID Connect (OIDC) identity tokens and issues short-lived certificates binding the public key to a verified identity (email, service account, or CI workflow information).

**Rekor** (Transparency Log): An immutable, append-only ledger that permanently records signing events, enabling public audit trails and verification that certificates were valid at signing time.

## Keyless Signing Workflow

The process eliminates long-lived key management:

1. Cosign generates an ephemeral key pair
2. A verifiable OIDC identity token is submitted with the certificate signing request
3. Fulcio validates the token and issues a short-lived certificate binding identity to the public key
4. The private key is discarded after single use
5. Artifact digest, signature, and certificate are logged in Rekor

As documented, "the signer ideally forgoes using long-lived keypairs" and "you don't have to manage signing keys, and Sigstore services never obtain your private key."

## Supported Artifact Types

The documentation references support for:
- Container images
- Release files
- Binaries
- Software bills of materials (SBOMs)
- Blobs (general data)
- Git commits (via Gitsign)
- Various other types through pluggable extensions

## Verification Process

Verification involves four complementary checks of the artifact's signature using the certificate's public key, confirming identity alignment, validating certificate signatures against Sigstore's root of trust, and verifying inclusion proof in Rekor.
