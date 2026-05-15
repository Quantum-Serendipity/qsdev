<!-- Source: https://spdx.github.io/spdx-ntia-sbom-howto/ -->
<!-- Retrieved: 2026-05-15 -->

# SPDX and NTIA Minimum Elements for SBOM: Complete Guide

## Overview

This guide explains how SPDX 2.x supports the NTIA Minimum Elements for Software Bills of Materials (SBOMs). SPDX is an ISO/IEC 5962:2021 international standard providing standardized formats for expressing software metadata in both machine-readable and human-readable formats.

## NTIA Minimum Elements and SPDX Field Mappings

| NTIA Element | SPDX 2.3 Field | Specification Reference |
|---|---|---|
| Supplier Name | `PackageSupplier` | Section 7.5 |
| Component Name | `PackageName` | Section 7.1 |
| Version of Component | `PackageVersion` | Section 7.3 |
| Other Unique Identifiers | `DocumentNamespace`, `SPDXID` | Sections 6.5, 7.2 |
| Dependency Relationship | `Relationship` (CONTAINS) | Section 11.1 |
| Author of SBOM Data | `Creator` | Section 6.8 |
| Timestamp | `Created` | Section 6.9 |

## Additional Mandatory SPDX Fields

Beyond NTIA requirements, SPDX 2.3 mandates these fields:

| Field | Required Value | Purpose |
|---|---|---|
| `SPDXVersion` | SPDX-2.3 | Specification version identifier |
| `DataLicense` | CC0-1.0 | SPDX Document licensing |
| `SPDXID` (Document) | SPDXRef-DOCUMENT | Document unique identifier |
| `DocumentName` | User-defined | Human-readable document name |
| `PackageDownloadLocation` | NOASSERTION or URL/VCS | Package source location |
| `FilesAnalyzed` | false (for minimal SBOM) | File-level analysis indicator |
| `Relationship` (DESCRIBES) | SPDXRef-DOCUMENT to primary package | Primary software designation |

## Key Concepts

### SPDX Identifiers

Each SPDX element requires a unique identifier composed of:
- **Document Namespace**: A unique URI defined once per document
- **Local Identifier**: Must begin with "SPDXRef-" followed by letters, numbers, periods, or hyphens

### Relationship Types for NTIA Compliance

**DESCRIBES Relationship**: Designates which package(s) the document primarily describes.
**CONTAINS Relationship**: Indicates dependency relationships between packages.

For unknown indirect dependencies, use CONTAINS NOASSERTION.

## Automation Support Requirements

NTIA specifies SBOMs should be expressed in "predictable implementation and data formats." SPDX supports this through:

- **Format Standardization**: Five supported formats (JSON, YAML, RDF/XML, tag-value, spreadsheet)
- **Machine-Parseable Structure**: Standardized field names and values
- **Validation Tools**: Online SPDX validator and NTIA conformance checker available

## Version-Specific Guidance

SPDX 2.3 (released November 2022) was the first version explicitly describing NTIA Minimum Elements compliance in its specification (Annex K.2). However, SPDX support for these elements dates to version 2.0 (2015).
