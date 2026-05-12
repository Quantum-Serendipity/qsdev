<!-- Source: https://docs.pypi.org/attestations/ + https://blog.pypi.org/posts/2024-11-14-pypi-now-supports-digital-attestations/ + https://peps.python.org/pep-0740/ -->
<!-- Retrieved: 2026-05-12 -->

# PyPI Digital Attestations (PEP 740)

## Overview

PEP 740 defines cryptographically verifiable attestations hosted by indices like PyPI. Digital attestations enable package maintainers as well as third parties (such as the index itself, external auditors, etc.) to cryptographically sign for uploaded packages.

## Supported Attestation Predicates

- SLSA Provenance
- PyPI Publish

The framework used is the in-toto Attestation Framework, which accepts attestations signed by:
- GitHub Actions
- GitLab CI/CD
- Google Cloud identities

## Key Features

- Attestations improve on traditional PGP signatures (which have been disabled on PyPI) by providing key usability, index verifiability, cryptographic strength, and provenance properties
- More than 20,000 attestations already published (as of late 2024)
- If you already publish packages to PyPI using Trusted Publishing, the official PyPI publishing workflow has attestation support built in, enabled by default as of v1.11.0

## API Access (Integrity API)

The Integrity API provides programmatic access to PyPI's implementation of PEP 740, operating on individual files and collecting all published attestations for a given file and returning them as a single response.

Endpoint format (from PEP 740):
```
GET /simple/<project>/<version>/<filename>/provenance
```

Returns attestation bundles in Sigstore bundle format.

## Verification

Verification can be done using:
- `pypi-attestations` Python package
- `sigstore-python` for lower-level verification
- pip does NOT yet verify attestations automatically (as of 2026)

## Limitations

- Attestation does not guarantee the package has no malicious code
- Only provides a verifiable link to source code and build instructions
- pip integration for automatic verification is still in progress
- Only packages published via Trusted Publishing from supported CI/CD platforms get attestations
