# NixOS Discourse: Interest in Checking Signatures While Building Packages

- **Source**: https://discourse.nixos.org/t/any-interest-in-checkings-signatures-while-building-packages/8918
- **Retrieved**: 2026-05-14

## Proposal

A developer suggested implementing signature verification during package builds in NixOS, using the Electrum package as an example. The idea was to embed upstream developers' public keys into package definitions and automatically verify cryptographic signatures during the build process.

## Arguments For

Proponents emphasized that signature checking would provide "a persistent security win" requiring only one-time effort. The approach would protect against compromised download servers or SSL certificate breaches -- scenarios where maintainers might unknowingly submit corrupted sha256 hashes. As one participant noted, this addresses situations where "the upstream archive changed and as a consequence the hash is broken," making it difficult to distinguish legitimate upstream modifications from actual compromises.

## Arguments Against

Critics pointed out that existing hash verification already provides sufficient protection. One developer observed that "all the sources require hashes," making signature verification redundant. Concerns also centered on implementation complexity -- adding per-package public keys would introduce "another dimension of complexity that very few people want to put up with."

## Key Insight: Mic92's Position

Mic92 (who later closed PR #43233) argued: "we should make this part of our tooling rather than the build process. Once we have our own checksum we no longer need to rely on public key cryptography." This represents the prevailing nixpkgs philosophy: hash pinning IS the security mechanism, and signature verification should happen in the update/review pipeline, not at build time.

## Community Consensus

The discussion concluded without strong consensus, with participants suggesting the proposal merited formal consideration as an RFC while acknowledging significant design questions remained unresolved. No RFC was ever filed.
