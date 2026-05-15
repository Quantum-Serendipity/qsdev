<!-- Source: https://anchore.com/blog/how-syft-scans-software-to-generate-sboms/ -->
<!-- Retrieved: 2026-05-15 -->

# Syft SBOM Scanning Architecture & Go Support Analysis

## Overall Scanning Architecture

Syft employs a four-step process for SBOM generation:

1. **Input Detection**: Identifies source type (container images, directories, archives, single files)
2. **Pluggable Cataloger Orchestration**: Deploys ecosystem-specific scanners
3. **Component Aggregation**: Consolidates discovered packages into standardized format
4. **Format Output**: Serializes to Syft JSON, SPDX, or CycloneDX formats

## Cataloger System

The architecture relies on decoupled, specialized catalogers. The document notes: "Rather than one monolithic scanner, Syft delegates scanning to a collection of catalogers, each focused on a specific software ecosystem."

Named catalogers include:
- Alpine (apk-db-cataloger)
- Debian (dpkg-db-cataloger)
- RPM packages (rpm-db-cataloger)
- Python, Java archives, Node/NPM modules

**Critical Gap**: The article makes **no mention** of Go-specific catalogers or Go binary handling. Go is mentioned only once, in the limitations section: "Syft also has room to grow in terms of programming language support. While it covers major ecosystems like Java and Python well, more work is needed to cover languages like Go, Rust, and Swift completely."

## Go-Specific Details

The document provides **zero technical details** about Go module discovery, Go binary scanning capabilities, or Go-specific metadata extraction. Go support is explicitly identified as an area requiring additional development work.

## Source vs. Binary Scanning

Syft handles both but faces inherent challenges with source-built packages: "When Syft scans the source code directory or docker image, it won't find any already built C++ libraries to detect as packages."

## Metadata Extraction

Per-package metadata captured includes name, version, type, associated files, source information (repository, URL), and file digests.
