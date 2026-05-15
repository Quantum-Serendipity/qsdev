<!-- Source: https://dev.to/kanywst/sigstore-deep-dive-unmasking-the-magic-behind-keyless-verification-lmh -->
<!-- Retrieved: 2026-05-15 -->

# Sigstore Keyless Verification: Technical Deep Dive

## Core Architecture

Sigstore comprises four integrated components:

| Component | Function | Traditional Equivalent |
|-----------|----------|----------------------|
| **Cosign** | Signing and verification CLI | `gpg sign`/`gpg verify` |
| **Fulcio** | Certificate authority issuing short-lived certs | DigiCert, traditional CAs |
| **Rekor** | Immutable transparency log | Certificate Transparency logs |
| **TUF** | Secure distribution of trust roots | Manual root cert installation |

## Fulcio: The 10-Minute Certificate Authority

### Why Short-Lived Certificates?

Traditional code signing relies on long-lived private keys, creating vulnerability windows. Fulcio eliminates revocation infrastructure by making certificates so brief that attackers lack exploitation time. "If we make the certificate's lifespan so short that an attacker has no time to exploit it, we don't need revocation management at all."

### OIDC-Based Identity Flow

1. Developer initiates signing (browser popup for manual signing, or automatic in CI)
2. OIDC provider (Google, GitHub) issues an ID Token -- a signed JWT containing authenticated identity
3. Developer creates ephemeral keypair and performs Proof of Possession by signing the token's `sub` claim
4. Fulcio verifies the OIDC token signature and PoP signature
5. Fulcio issues X.509 certificate with 10-minute validity window
6. Certificate is submitted to Certificate Transparency log

### Certificate Contents

The Fulcio certificate embeds identity information:

**Subject Alternative Name (SAN)** varies by OIDC provider:
- Google: `you@gmail.com`
- GitHub personal: `you@users.noreply.github.com`
- GitHub Actions: `https://github.com/org/repo/.github/workflows/build.yml@refs/heads/main`

**Custom OID Extensions** (Enterprise Number `1.3.6.1.4.1.57264`) encode CI/CD provenance:
- OIDC issuer URL
- Build signer URI
- Runner environment type
- Source repository URI and commit digest
- Workflow trigger type
- Run invocation URL

This enables "X.509-level guarantee" that a specific workflow in a specific repository triggered the signature.

### Certificate Chain Structure

```
Root CA (Fulcio Root, self-signed)
    |
Intermediate CA (constrained with pathlen: 0)
    |
Leaf Certificate (signing key, valid 10 minutes)
```

## Rekor: Transparency Log with Merkle Trees

### Solving the Temporal Problem

A critical gap exists: we need proof that the signature occurred *while the 10-minute certificate remained valid*. Rekor solves this by recording signing events with cryptographically signed timestamps.

### Merkle Tree Structure

Rekor organizes entries in a binary hash tree where:
- Leaf nodes contain entry hashes
- Parent nodes contain Hash(left_child + right_child)
- Root Hash encompasses the entire log
- Root Hash is signed by Rekor's private key

### Inclusion Proofs

Proving entry existence requires only O(log n) hashes rather than downloading the entire log. A million entries need only ~20 hashes for proof.

### Data Structure

Rekor entries contain:

```json
{
  "apiVersion": "0.0.1",
  "kind": "hashedrekord",
  "spec": {
    "data": {
      "hash": {
        "algorithm": "sha256",
        "value": "410dabcd6f1d..."
      }
    },
    "signature": {
      "content": "MEUCIQDx...(base64 signature)...",
      "publicKey": {
        "content": "LS0tLS1C...(base64 Fulcio cert)..."
      }
    }
  }
}
```

Server metadata adds:
- `logIndex`: Sequential position
- `integratedTime`: UNIX timestamp of recording
- `verification.inclusionProof`: Merkle sibling hashes and Root Hash
- `verification.signedEntryTimestamp`: Timestamp signed by Rekor key

## TUF: Defending the Root of Trust

### Four-Role Trust Model

TUF uses role-based metadata with threshold signatures:

- **root.json** - God-key defining all roles and signature thresholds (e.g., 3-of-5)
- **targets.json** - Records hashes/sizes of distributed files (certificates, keys)
- **snapshot.json** - Locks version numbers, preventing mix-and-match attacks
- **timestamp.json** - Updated frequently with short lifespan, preventing staleness

### Bootstrap Process

Cosign ships with a foundational `root.json`. On first run:
1. Fetch latest `root.json` from CDN
2. Verify signature using previous version's keys (chained verification)
3. Extract new role keys and thresholds
4. Fetch `timestamp.json`, verify with new keys
5. Fetch `snapshot.json` and `targets.json`
6. Retrieve `trusted_root.json` containing Fulcio CA chain and Rekor public key

## Complete Verification Algorithm

When running `cosign verify $IMAGE`:

```bash
cosign verify $IMAGE \
  --certificate-identity="you@gmail.com" \
  --certificate-oidc-issuer="https://accounts.google.com"
```

The verification process:
1. Pull signature manifest (`.sig` tag) from registry
2. Extract signature, Fulcio certificate, Rekor entry
3. Validate certificate chain up to Fulcio Root CA (obtained via TUF)
4. Confirm `integratedTime` falls within certificate's 10-minute validity window
5. Verify certificate SAN exactly matches `--certificate-identity`
6. Confirm certificate Issuer OID matches `--certificate-oidc-issuer`
7. Use certificate's public key to verify image digest signature
8. Verify Rekor's Inclusion Proof using Rekor public key (from TUF)

Success indicates irrefutable cryptographic proof that the specific identity signed the specific image at a specific moment.

## GitHub Actions Integration

GitHub Actions provides native OIDC (`https://token.actions.githubusercontent.com`). Minimal workflow:

```yaml
permissions:
  id-token: write  # Required for OIDC token
  packages: write  # Required for registry push

steps:
  - uses: sigstore/cosign-installer@v3
  - name: Sign with Cosign
    run: cosign sign --yes $IMAGE
```

Cosign automatically detects CI environment, bypasses browser popup, and performs headless signing.

## Key Advantages Over Legacy PKI

| Aspect | Legacy PKI | Sigstore |
|--------|-----------|----------|
| Key Lifecycle | Multi-year keys require HSM/Vault management | Ephemeral keys destroyed in seconds |
| Revocation | Complex CRL/OCSP infrastructure | Obsolete due to short lifespan |
| Identity | Organization-based (requires corporate entity) | Developer identity via OIDC (free) |
| Transparency | Completely opaque | All signatures in immutable Rekor log |
| OSS Compatibility | Key distribution nightmare | Natural fit with CI/CD automation |
| Cost | $200-500+/year per CA | Free (public instance) |
