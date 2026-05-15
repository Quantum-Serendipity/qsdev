<!-- Source: https://github.com/in-toto/attestation -->
<!-- Retrieved: 2026-05-15 -->

# in-toto Attestation Framework

## Core Overview

The in-toto Attestation Framework enables "verifiable claims about any aspect of how a piece of software is produced," allowing consumers to validate software origins and establish supply chain trust.

## Key Components

**Specification & Structure:**
The framework includes a formal specification (https://github.com/in-toto/attestation/tree/main/spec/v1) defining attestation formats and metadata structure.

**Predicate Types:**
The repository maintains vetted attestation predicates (https://github.com/in-toto/attestation/tree/main/spec/predicates) covering common use cases including:
- SLSA Provenance
- SBOM
- Other supply chain metadata

**Language Bindings:**
Protobuf definitions support multiple languages:
- Go (most mature)
- Python
- Rust (most recent)
- Java

## Governance & Community

The framework operates under CNCF as part of the in-toto project. Maintainers use `@in-toto/attestation-maintainers` tags. Community discussions occur in the Slack channel `#in-toto-attestations`.

## Note

The in-toto Attestation Framework is still under active development, with ongoing tooling integration efforts.

For comprehensive technical details, consult the official documentation at https://github.com/in-toto/attestation/tree/main/docs and specification files directly.
