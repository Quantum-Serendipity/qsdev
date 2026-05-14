<!-- Source: https://github.com/sigstore/cosign/blob/main/specs/SIGNATURE_SPEC.md -->
<!-- Retrieved: 2026-05-14 -->

# Cosign Signature Specification: Technical Details

## What Gets Signed

Cosign uses **detached signatures** where "the payload MUST contain the digest of the image it references, in a well-known location." Rather than signing full file content, the system signs a digest representation. Specifically, signatures are created through this chain:

`Sign(sha256(SimpleSigningPayload(sha256(Image Manifest))))`

This two-hop approach means the actual image manifest is always referenced by SHA256, establishing the cryptographic link without signing raw content.

## Signature Format

Signatures are "base64-encoded and stored as an annotation on the layer" with the key `dev.cosignproject.cosign/signature`. The specification requires support for **ECDSA-P256** with SHA256 hashing, though other schemes may be supported.

## Bundle Format

Optional bundle objects contain a JSON structure with:
- **SignedEntryTimestamp**: A rekor-signed signature over log metadata
- **Payload** fields including the body (base64-encoded Rekor log entry), integratedTime (UNIX timestamp), logIndex, and logID (SHA256 hash of the transparency log's public key)

## Blob Signing Protocol

Payloads are "uploaded to an OCI registry as a blob, and are referenced by digest, size and mediaType." The blob receives its own content-addressable storage reference. This design enables "validating the signature against the payload without fetching the blob, because the blob's digest is also present in the manifest."

## Hash Algorithm Support

The specification mandates "use the same hash algorithm used by the underlying registry to reference the payload." In practice, this pins implementations to **SHA256**, eliminating algorithm agility for stored signatures while maintaining compatibility with registry standards.
