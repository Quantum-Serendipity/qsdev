# Nix fetchurl/default.nix Analysis

- **Source**: https://github.com/NixOS/nixpkgs/blob/master/pkgs/build-support/fetchurl/default.nix
- **Retrieved**: 2026-05-14
- **Note**: Content summarized via WebFetch; raw source is Nix code

## Hash/SRI Verification Parameters

The file supports multiple hash specification methods:

- **SRI hash**: `hash ? ""`
- **Legacy formats**: `sha1`, `sha256`, `sha512` parameters
- **Explicit output hash**: `outputHash` and `outputHashAlgo` pair
- **Recursive hashing**: `recursiveHash ? false` for directory contents

The hash resolution logic prioritizes inputs: SRI hash takes precedence, followed by explicit `outputHash`/`outputHashAlgo`, then legacy SHA variants.

## Fixed-Output Derivation Mechanism

The implementation uses structured attributes (`__structuredAttrs = true`) and converts hash inputs into standardized format: `"${algorithm}:${hash}"`. When hash information is missing but `cacert` exists, the system uses `lib.fakeHash` as a placeholder.

## Integrity Checking Process

The file enforces strict validation:

> "fetchurl requires a hash for fixed-output derivation"

This throws an error if no hash variant is provided. The system prevents multiple hashes simultaneously and requires `outputHashAlgo` when using explicit `outputHash`.

## Notable Omissions

**No GPG/signature verification options appear in this code.** The document focuses exclusively on cryptographic hash validation. TLS certificate verification is conditionally disabled when using fake hashes or authentication credentials (`netrcPhase`).

The `postFetch` parameter allows custom validation logic post-download, offering extensibility for signature checking if implemented downstream.
