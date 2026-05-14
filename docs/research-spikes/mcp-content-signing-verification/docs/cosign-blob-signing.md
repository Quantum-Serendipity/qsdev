<!-- Source: https://docs.sigstore.dev/cosign/signing/signing_with_blobs/ -->
<!-- Retrieved: 2026-05-14 -->

# Cosign Blob Signing Overview

## What is Blob Signing?

Cosign enables signing and verification of standard files and binary objects beyond container images. The `cosign sign-blob` command handles this functionality for non-container artifacts.

## Signing Methods

### Keyless Signing (Recommended)
The preferred approach uses identity-based signing with ephemeral keys linked to OpenID Connect providers:

```bash
cosign sign-blob <file> --bundle bundle.sigstore.json
```

**Key feature:** "The bundle contains verification metadata, including an artifact's signature, certificate and proof of transparency log inclusion."

### Key-Based Signing
For self-managed keys, users supply their own credentials:

```bash
cosign sign-blob --key cosign.key --bundle bundle.sigstore.json README.md
```

This approach requires entering the private key password and supports KMS providers and hardware tokens.

## Automated Signing

For CI/CD environments, the `--yes` flag eliminates confirmation prompts:

```bash
cosign sign-blob --yes --key cosign.key --bundle bundle.sigstore.json myimage:latest
```

## Output Format

The recommended bundle approach consolidates signature, certificate, and transparency log metadata into a single `.sigstore.json` file, streamlining verification workflows compared to managing separate signature and certificate files.

## Additional Capabilities

Blob signing inherits features from standard signing operations, including support for hardware tokens, KMS integrations, and keyless authentication methods.
