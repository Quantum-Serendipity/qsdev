<!-- Source: https://github.com/sigstore/cosign -->
<!-- Retrieved: 2026-05-15 -->

# Cosign: Code Signing and Transparency for Containers and Binaries

## Overview

Cosign enables signing OCI containers and other artifacts using Sigstore. It aims to make signatures "invisible infrastructure."

## Key Features

**Signing Methods:**
- Keyless signing via Fulcio CA and Rekor transparency log (default)
- Hardware and KMS-based signing
- Encrypted private/public keypair signing
- Bring-your-own PKI support

**Supported Artifacts:**
- Container images
- Blobs and binaries
- Tekton Bundles
- WebAssembly modules
- eBPF modules
- In-toto attestations

## Core Commands

### Container Signing and Verification

**Sign:** `cosign sign $IMAGE` -- prompts OIDC authentication, requests code-signing certificate from Fulcio, stores signature in registry.

**Verify:** `cosign verify $IMAGE --certificate-identity=$IDENTITY --certificate-oidc-issuer=$OIDC_ISSUER`

### Blob Operations

`cosign upload blob` -- publish artifacts to OCI registries with digest verification.

### Attestation Commands

**Attest:** `cosign attest --predicate <file> --key cosign.key $IMAGE_URI_DIGEST`

**Verify Attestation:** `cosign verify-attestation --key cosign.pub $IMAGE_URI`

### Blob Signing

**Sign-blob:** Enables keyless blob signing without requiring `--key` flag.

**Verify-blob:** Validates blobs against expected signer identity using certificate information.

## Key Management

- **Generated keypairs:** Encrypted with scrypt KDF and nacl/secretbox in PEM format
- **Hardware tokens:** Direct integration
- **KMS providers:** Hashicorp Vault, AWS KMS, GCP KMS, Azure Key Vault
- **Keyless:** Uses OIDC providers for ephemeral certificate generation

## Keyless Signing Workflow

1. User initiates signing without local keys
2. OIDC authentication redirects to browser-based login
3. Fulcio issues ephemeral code-signing certificate matching authenticated email
4. Signature and certificate stored in Rekor transparency log
5. Signature uploaded to OCI registry alongside image

## Verification Checks

- Cosign claims validation
- Presence in transparency log
- Signature integration timing verification
- Cryptographic signature validation
- Certificate authority validation

## Registry Support

Tested with AWS ECR, GCP Artifact Registry, Docker Hub, Azure Container Registry, Harbor, Quay, and others.

## Air-Gapped Verification

- Save images locally using `cosign save`
- Use bundle annotations containing verification materials
- Verify with `cosign verify --offline=true --trusted-root <path>`

## DSSE Signing

For in-toto attestations, uses DSSE (Dead Simple Signing Envelope) signing specification.

## Development Status

Future Cosign development will be focused on the next major release based on sigstore-go. Current 2.x releases remain stable.
