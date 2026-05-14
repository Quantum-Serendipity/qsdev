# WACZ Signing and Verification Specification (v0.1.0)

- **Source URL**: https://specs.webrecorder.net/wacz-auth/0.1.0/
- **Retrieved**: 2026-05-14

## datapackage-digest.json Structure

```json
{
  "path": "datapackage.json",
  "hash": "sha256:<hash_value>",
  "signedData": <SignatureData>
}
```

## SignatureData Formats

### Anonymous Signature
```json
{
  "hash": "<sha256 hash>",
  "created": "<ISO 8601 Date>",
  "software": "<string>",
  "version": "<string>",
  "signature": "<base64 encoded>",
  "publicKey": "<base64 encoded public key (ECDSA)>"
}
```

### Domain-Ownership + Timestamp Signature
```json
{
  "hash": "<sha256 hash>",
  "created": "<ISO 8601 Date>",
  "software": "<string>",
  "version": "<string>",
  "signature": "<base64 signature by domainCert>",
  "domain": "<valid hostname>",
  "domainCert": "<PEM certificate chain>",
  "timeSignature": "<base64 RFC 3161 signature>",
  "timestampCert": "<PEM certificate chain>",
  "crossSignedCert": "<PEM certificate chain (optional)>"
}
```

## Signing Algorithms

- **Algorithm**: ECDSA (Elliptic Curve Digital Signature Algorithm)
- **Hash function**: SHA-256
- **Timestamp standard**: RFC 3161

## Domain-Ownership Verification Workflow

1. Creator generates ECDSA private key
2. Creates Certificate Signing Request (CSR)
3. Obtains TLS certificate from trusted CA (e.g., LetsEncrypt)
4. Optionally creates backup cross-signed certificate via secondary CA
5. Signs hash using private key
6. Uses RFC 3161 timestamp server to sign the signature

The domain certificate proves the creator owns the specified domain, discoverable via Certificate Transparency logs.

## Complete Validation Flow

**For all WACZ files:**
1. Validate WACZ as Frictionless Data Package (hashes match contents)
2. Confirm hash in `datapackage-digest.json` matches `datapackage.json`
3. Verify `signedData` conforms to specification

**For anonymous signatures:**
4. Validate signature against hash using public key (external key validation required)

**For domain-ownership signatures:**
4. Validate `signature` of hash using first certificate's public key in `domainCert`
5. Confirm `domain` matches certificate subject name
6. Validate `timeSignature` as valid RFC 3161 timestamp of `signature`
7. Check `created` date within 10 minutes of signed timestamp
8. Verify both certificate chains against trusted roots (optionally check Certificate Transparency)
9. If `crossSignedCert` provided: verify matching public key and validate alternative trust path

**For partial WACZ loading:**
Verify individual WARC record hashes against CDXJ index instead of full validation.

## Tools & Implementations

- **Webrecorder**: Created the WACZ and WACZ Auth specifications
- **js-wacz** (Harvard LIL): JavaScript module/CLI for WACZ with signing support
- **ReplayWeb.page**: Primary viewer with verification badge display
