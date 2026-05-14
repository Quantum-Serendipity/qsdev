# Checking OpenPGP Signatures in Nix Builds

- **Source**: https://scottworley.com/blog/2022-09-20-checking-openpgp-signatures-in-nix-builds.html
- **Retrieved**: 2026-05-14

## Core Approach

The method involves three components fetched as separate fixed-output derivations:

1. **Fetching the signing key**: Downloads the project's public key from a trusted source (pinned by hash)
2. **Fetching the unverified resource**: Obtains the software artifact (pinned by hash)
3. **Fetching the signature**: Retrieves the corresponding detached signature file (pinned by hash)

## Implementation Pattern

The example uses Tails Linux, implementing a `verified-tails-iso` derivation that:

- Creates a temporary GPG home directory
- Imports the signing key: `"${gnupg}/bin/gpg --import ${tails-signing-key}"`
- Verifies the signature: `"${gnupg}/bin/gpg --verify ${tails-iso-signature}"`
- Symlinks the verified ISO upon successful verification

## Version Management

When updating software versions, only the hash values for the resource and signature files need updating. The signing key hash remains unchanged across updates, reducing the modification surface area.

## Trust Model

This approach makes signature verification "publicly-verifiable" by embedding it in the build configuration. Anyone can inspect the derivation and see which key is trusted, what signature is expected, and verify the chain independently.

## Significance for gdev

This pattern demonstrates that per-artifact GPG verification IS possible in Nix without any nixpkgs infrastructure changes -- it just requires writing a custom derivation that imports a key and runs `gpg --verify`. The approach works today.
