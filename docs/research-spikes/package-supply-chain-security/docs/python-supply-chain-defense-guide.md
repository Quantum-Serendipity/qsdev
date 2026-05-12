# Defense in Depth: A Practical Guide to Python Supply Chain Security

- **Source**: https://bernat.tech/posts/securing-python-supply-chain/
- **Retrieved**: 2026-05-12

## Dependency Pinning & Hashing

The foundational protection involves moving from unpinned to hash-pinned dependencies:

- **Unpinned** (`flask>=2.0`): Accepts any version; vulnerable to malicious updates
- **Version pinned** (`flask==3.1.1`): Gets exact version but lacks integrity verification
- **Hash pinned** (`flask==3.1.1 --hash=sha256:...`): "Cryptographic fingerprint of the package file" that detects tampering

Tools like `uv pip compile --generate-hashes` and `pip-tools` automatically generate these checksums.

## Time-Based Defenses

Modern tools enable delayed package ingestion:

- **`uv --exclude-newer`**: "Only use packages published before a specific date"
- **`pip --uploaded-prior-to`** (v26+): Equivalent functionality, providing a buffer period for community threat detection

The article notes this strategy leverages the community as "canaries in the coal mine" by introducing a 6-7 day delay before using newly published packages.

## Vulnerability Scanning

**`pip-audit`** checks dependencies against multiple vulnerability databases (OSV, PyPA Advisories, GitHub Advisories, NVD). Recommended to run in CI to "catch known CVEs before they hit production."

## SBOMs (Software Bill of Materials)

**`cyclonedx-py`** generates inventory documents enabling rapid impact assessment.

## Package Verification

**Trusted Publishing** with OIDC eliminates long-lived API tokens, replacing them with short-lived credentials that "expire in minutes instead of forever." **Package attestations** via Sigstore create cryptographic proof linking packages back to source repositories.

## Linting & Code Security

**Ruff** with security rules catches hardcoded secrets (S105), weak cryptography (S324), missing timeouts (S113), and unsafe deserialization patterns before code is committed.

The article emphasizes that no single control is perfect — "layer your defenses" so that when one mechanism fails, others provide protection.
