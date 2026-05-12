# Maven Central Adds Sigstore Signature Validation
- **Source**: https://socket.dev/blog/maven-central-adds-sigstore-signature-validation
- **Retrieved**: 2026-05-12

## What Changed

Sonatype announced that the Maven Central Publisher Portal now validates Sigstore signatures alongside traditional artifacts. Publishers can include `.sigstore.json` files with their Java packages for cryptographic verification at publication time.

## How It Works Alongside PGP

The implementation maintains backward compatibility. As Maven Central stated: "We are monitoring adoption of Sigstore and may eventually make both Sigstore and PGP signatures required" but "have no intention of replacing PGP signatures." Currently, PGP remains the standard, with Sigstore operating as a supplementary option.

## Verification Process

**For Publishers:**
- Sigstore signing is currently optional
- The portal validates submitted signatures and provides warnings if verification fails
- Invalid signatures will eventually block publishing, but missing signatures do not

**For Consumers:**
- Developers can use Sigstore signatures to verify package provenance
- Signatures coexist with traditional PGP signatures for verification purposes

## What Publishers Need to Do

No immediate action is required. Publishers can voluntarily include Sigstore signatures with their artifacts. The technology uses "keyless signing" tied to identity providers like GitHub or Google, simplifying adoption compared to managing long-lived PGP keys.

## Future Enforcement Plans

Maven Central may eventually require both Sigstore and PGP signatures if community adoption increases. However, the organization guarantees "there will always be a way to cryptographically verify components downloaded from Maven Central."

## Current Adoption Status

Sigstore validation represents industry momentum across ecosystems. PyPI has introduced digital attestations, and npm added provenance capabilities in 2023. This broader shift suggests potential future standardization across package repositories.
