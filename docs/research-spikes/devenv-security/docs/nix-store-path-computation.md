# Nix Store Path Computation: Technical Overview
- **Source**: https://nixos.org/guides/nix-pills/18-nix-store-paths
- **Retrieved**: 2026-05-12

## Core Mechanism

Nix computes store paths through a three-step hashing process that ensures deterministic, content-addressed storage. The system relies on integrity hashes to prevent collisions and maintain verification.

## Hash Computation Process

**Step 1: NAR Serialization and Initial Hash**

Nix uses NAR (Nix ARchive) format to serialize file contents. The process begins by computing a SHA256 hash of the NAR serialization: `"nix-hash --type sha256 myfile"` or `"nix-store --dump myfile|sha256sum"`. This standardized format handles both flat files and recursive directory structures.

**Step 2: String Description Construction**

A special string combines the hash type, content hash, store location, and filename. For source paths, this follows the pattern: `"source:sha256:[hash]:/nix/store:[filename]"`. This fingerprints the derivation's identity.

**Step 3: Final Path Generation**

The store path itself derives from truncating the first 160 bits of a SHA256 hash of the description string, then encoding in base-32. This produces the characteristic short identifiers like `"xv2iccirbrvklck36f1g7vldn5v58vck"`.

## Path Types

**Source Paths**: Hash regular files or directories placed in the store directly.

**Output Paths**: Generated for derivations. Computed by replacing output paths in `.drv` files with empty strings, then hashing the modified derivation file using the `"output:out"` type prefix.

**Fixed-Output Paths**: Used for downloads with pre-declared integrity hashes. The string format is `"fixed:out:sha256:[hash]:"`, enabling content verification without building.

## Integrity Maintenance

Input derivation references within `.drv` files are recursively replaced by their computed hashes, creating a deterministic dependency chain. This ensures reproducibility -- the output path depends only on inputs, known before building. Fixed-output derivations anchor verification to declared hashes rather than build processes, fundamental to nixpkgs' source tarball verification system.

## Collision Prevention

The original reason for computing names this way was to prevent name collisions for security purposes -- the thinking was that it shouldn't be feasible to come up with a derivation whose output path collides with the path for a copied source, with derivation-produced data having an inner-fingerprint starting with "output:out:" while manually-hashed content would start with "source:".
