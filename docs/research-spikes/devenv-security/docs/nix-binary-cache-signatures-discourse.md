<!-- Source: https://discourse.nixos.org/t/what-guarantees-do-signatures-by-binary-caches-give/34802 -->
<!-- Retrieved: 2026-05-12 -->

# Binary Cache Signatures in Nix: Guarantees and Limitations

## What Signatures Cover

Binary cache signatures in Nix provide verification over multiple components. According to the discussion, signatures cover:

- The store path address (input hash)
- The NAR (Nix Archive) hash of contents
- References between paths
- The derivation that produced them

As one participant noted, the evidence lies in the codebase where "both, plus its references" are included in the signature fingerprint.

## Current Guarantees

The signatures primarily establish that "it was signed," confirming the cache owner trusts the path through either building it themselves or choosing to sign it. However, a critical limitation exists: **signatures do not cover the `Deriver:` field** in the `.narinfo` metadata.

This creates a significant gap. When multiple derivations can produce identical outputs (particularly with fixed-output derivations), there's no cryptographic link proving which specific derivation actually generated a given output.

## Identified Problems

A major concern raised involves situations where the `Deriver` field from cache.nixos.org doesn't match the locally-evaluated derivation. One developer stated: "the same store path can actually be created from different derivations, so that the link in that direction is not unique."

This manifests as a **provenance tracking gap**. Without signing over the derivation metadata, users cannot verify whether a builder compiled trustworthy code or potentially malicious versions that produce identical outputs.

## Trust Limitations

The current model requires trusting the substituter with the entire build process itself, regardless of whether the system uses input-addressed or content-addressed derivations.
