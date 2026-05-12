# RFC 0062: Content-Addressed Paths (Security-Relevant Sections)
- **Source**: https://github.com/NixOS/rfcs/blob/master/rfcs/0062-content-addressed-paths.md
- **Retrieved**: 2026-05-12

## Trust Model Improvements

The RFC presents a significant shift in Nix's trust architecture. Unlike input-addressed derivations that require signatures because there's no verifiable link between hash and content, content-addressed paths enable cryptographic verification: "If `/nix/store/123-foo` is content-addressed, then `123` is supposed to be a hash of the content of the path, and that can be easily verified."

A notable future capability involves enabling multi-user scenarios where trust becomes decoupled from storage: "Each user could be a 'trusted-user' for its own view of the store, without affecting the others," potentially allowing shared infrastructure with separate trust domains.

## Reduced Attack Surface

The model eliminates certain verification requirements. Content-addressed outputs don't need signatures for integrity verification -- the hash itself serves as proof. However, the RFC introduces realisation metadata that must be signed, creating a cleaner separation: derivation outputs require signatures only when their relationship to inputs isn't mathematically deterministic.

## Security Properties vs Input-Addressed Model

**Content-addressed advantages:**
- Direct cryptographic verification of output integrity
- Detection of supply-chain compromises through hash mismatches
- Reproducibility guarantees tied to actual content

**Trade-offs mentioned:**
The RFC acknowledges the "two-glibc issue" -- non-deterministic builds can create closure duplication when mixing sources, potentially leading to subtle runtime failures. The current mitigation prevents fetching incompatible realisations locally.

## Implementation Status

The document states: "The implementation of this RFC is already partially integrated into Nix, behind the `ca-derivation` experimental flag," indicating active but incomplete deployment as of the RFC's date.
