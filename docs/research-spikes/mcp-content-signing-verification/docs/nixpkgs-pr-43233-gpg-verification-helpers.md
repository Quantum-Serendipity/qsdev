# nixpkgs PR #43233: GPG Signature Verification Helpers

- **Source**: https://github.com/NixOS/nixpkgs/pull/43233
- **Retrieved**: 2026-05-14

## Overview

PR #43233 proposed infrastructure for verifying PGP signatures on source tarballs and binaries in nixpkgs. It was **closed without merging** on August 20, 2020 by Mic92. The branch was deleted on February 17, 2022.

## Proposed Components

### 1. fetchpgpkey (`pkgs/build-support/fetchpgpkey/default.nix`)

A utility that downloads PGP public keys and verifies their fingerprints:
- Downloads a key from a specified URL
- Validates the fingerprint matches expected value
- Returns the key for use in signature verification
- Requires both sha256 hash and fingerprint for security

### 2. verifySignatureHook (`pkgs/build-support/setup-hooks/verify-signature.sh`)

A setup hook providing helper functions:

- `_importPublicKey()` - Adds public key to GPG keyring
- `verifySignature SIGFILE DATAFILE [UNCOMPRESS]` - Verifies detached signatures
- `verifySrcSignature()` - Automatically checks source signatures before unpacking

The workflow automatically creates a temporary GPG home directory and integrates into the `preUnpackHooks` lifecycle.

### 3. Example Packages

Three packages demonstrated the API:

1. **1password**: Binary signature verification during build
2. **tor-browser-bundle-bin**: Tarball signature validation
3. **samba4**: Source tarball signature checking

Example pattern from 1password:
```nix
signaturePublicKey = fetchpgpkey {
  url = https://keybase.io/1password/pgp_keys.asc;
  fingerprint = "3FEF9748469ADBE15DA7CA80AC2D62742012EA22";
  sha256 = "1v9gic59...";
};
```

## Review Comments and Reasons for Rejection

### Concerns Raised

1. **Security architecture issue** - roconnor-blockstream argued against integrating verification into `fetchurl`, stating: "placing it as part of `fetchurl` would enable an attack where an attacker tricks a victim to load a trojan-horse."

2. **Redundancy objection** - Mic92 concluded that "we should make this part of our tooling rather than the build process. Once we have our own checksum we no longer need to rely on public key cryptography."

3. **Implementation questions** - Reviewers questioned handling of expired keys and key updates over time.

### Fundamental Disagreement

Reviewers believed signature verification should occur *outside* the build derivation rather than as a setup hook, to maintain cleaner separation of concerns and avoid security implications of verifying within fixed-output contexts. Alternative approaches (separate verification derivations) were suggested but never implemented.
