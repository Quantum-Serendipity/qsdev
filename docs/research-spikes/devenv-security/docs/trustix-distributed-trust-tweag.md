<!-- Source: https://www.tweag.io/blog/2020-12-16-trustix-announcement/ -->
<!-- Retrieved: 2026-05-12 -->

# Trustix: Distributed Trust for Binary Caches

## Overview

Trustix is a tool designed to decentralize trust in binary software distribution by comparing build outputs across independent providers rather than relying on single-point-of-failure signature schemes.

## The Problem with Traditional Binary Caches

Current package managers like Nix use centralized trust models where users verify binaries through cryptographic signatures from a single authority. This approach has critical limitations:

- **Single point of failure**: If the signing key is compromised, all cached binaries become untrusted
- **Binary trust/distrust**: Users must either fully trust or completely distrust a build machine with no middle ground
- **Weak input verification**: No inherent guarantee that stated build inputs actually produced the cached output

## Trustix Solution

Rather than verifying signatures, Trustix acts as a "proxy for a binary cache" that only exposes packages meeting configurable trustworthiness criteria. If a package fails verification, Nix simply rebuilds it from source.

The core innovation involves building "a Merkle tree-based append-only log that maps build inputs to build outputs," establishing consensus about whether specific inputs consistently produce identical outputs across multiple builders.

## Addressing Trust Issues

**Compromise resilience**: If one builder is compromised, the network majority can still establish binary trustworthiness. Old packages remain verifiable even when newer ones are suspect.

**Graduated trust levels**: Rather than absolute trust, users configure machines as more or less trusted based on community involvement or historical compromise patterns.

**Decentralized mapping**: Builders maintain individual logs aggregated locally, eliminating a central database as a failure point.

## Limitations and Future Applications

The model requires reproducible builds — non-reproducible outputs prevent consensus. However, Trustix can track reproducibility failures across entire package ecosystems more comprehensively than single-machine approaches like r13y.

The project was funded by NLNet and the European Commission's Next Generation Internet program.
