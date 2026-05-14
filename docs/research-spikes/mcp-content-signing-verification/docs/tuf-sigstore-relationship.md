<!-- Source: https://dlorenc.medium.com/using-the-update-framework-in-sigstore-dc393cfe6b52 -->
<!-- Retrieved: 2026-05-14 -->

# TUF and Sigstore Integration

## Roles and Relationship

TUF provides the foundational framework for securing cryptographic keys and metadata within Sigstore's infrastructure. Rather than competing technologies, they work together synergistically.

Sigstore integrates TUF to accomplish two distinct goals:
1. **Internal protection**: TUF secures Sigstore's own keys and infrastructure
2. **User enablement**: TUF tools become available to end users through Sigstore's utilities

## The "TUF Sandwich" Architecture

Dan Lorenc characterizes the integration as "a TUF sandwich," consisting of layered trust mechanisms:

**Foundation Layer**: The Sigstore root of trust, established through a public key ceremony in June 2021, uses five hardware keys controlled by community members. This root "contains the full set of keys used to sign release artifacts and metadata, _as well as_ the rules and policies used to update this root over time."

**Delegation Layer**: TUF's delegation feature allows the Sigstore root to authorize other projects' keys through custom roles, creating what Lorenc calls a "super root" — comparable to how root CAs in web PKI issue intermediate certificates.

## Comparative Use Cases

| Scenario | Approach |
|----------|----------|
| **Sigstore infrastructure** | TUF protects internal keys (Fulcio CA, Rekor transparency log) |
| **Open source projects** | Leverage Sigstore's root via TUF delegations |
| **Air-gapped systems** | "Detached TUF" using OCI registries without Sigstore infrastructure |
| **Small communities** | Lightweight policy manifests checked into Git repositories |

## Complementary Features

The systems address different but interconnected problems:

- **Sigstore's contribution**: Ephemeral identity certificates through OIDC/SPIFFE, enabling human-readable identities instead of raw public keys
- **TUF's contribution**: Secure metadata framework, key rotation policies, and graceful root updates

Sigstore merges these through its Fulcio CA, which issues certificates backed by the Sigstore trust root, allowing "TUF policies can be expressed in terms of these human-readable identities instead of just plain public keys."

## When to Use Which

Users don't typically choose TUF *or* Sigstore — they're integrated. The question becomes which Sigstore deployment model fits:
- Use Sigstore's established root for community projects
- Use detached TUF in OCI registries for isolated environments
- Use policy manifests with Rekor transparency logs for lightweight trust bootstrapping
