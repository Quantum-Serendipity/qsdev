<!-- Source: https://scottworley.com/blog/2022-09-20-checking-openpgp-signatures-in-nix-builds.html -->
<!-- Retrieved: 2026-05-14 -->

# GPG/OpenPGP Signature Verification in Nix Builds

## Approach

The method involves performing signature validation as an explicit step during the build process, making verification publicly auditable rather than implicit.

## Key Nix Primitives

The approach uses three core Nix functions:

1. **`fetchurl`** - Downloads three separate resources: the signing key, the ISO file, and its signature file
2. **`runCommand`** - Executes the verification logic as a build derivation
3. **`gnupg`** - Provides the GPG tooling for importing keys and verifying signatures

## Implementation Pattern

The example demonstrates this workflow:
- Import the signing key into a temporary GPG home directory
- Execute `gpg --verify` to validate the signature against the downloaded ISO
- If verification succeeds, symlink the unverified ISO to the output, effectively marking it verified

As the documentation notes: "Version bumps change the fetch hashes of the signed resource and the signature, _but not the signing key_" -- only the ISO and signature hashes require updates during version upgrades.

## Key Management

The signing key is treated as a separately fetchable, hash-verified artifact with its own `sha256` value. This allows key rotation to be explicit and tracked in version control.

## Implicit Challenge

The approach requires trust in the initial key hash; there's no built-in mechanism to establish that hash's authenticity beyond the Nix repository's own integrity.
