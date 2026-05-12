# Sbomnix: Software Supply Chain Tools for Nix
- **Source**: https://github.com/tiiuae/sbomnix
- **Retrieved**: 2026-05-12

## Overview

Sbomnix is a suite of command-line utilities designed to address software supply chain security challenges for Nix-based projects. The primary tool, `sbomnix`, generates Software Bill of Materials (SBOMs) from Nix flake references or store paths.

## Core Functionality

The repository hosts several complementary tools:

- **sbomnix**: Generates SBOMs from Nix targets
- **nixgraph**: Queries and visualizes dependency graphs
- **nixmeta**: Summarizes nixpkgs metadata attributes
- **vulnxscan**: Demonstrates vulnerability scanning using SBOMs
- **nix_outdated**: Identifies outdated dependencies prioritized by downstream impact
- **provenance**: Creates SLSA v1.0 compliant provenance attestation files
- **repology_cli/repology_cve**: Command-line clients to repology.org

## Supported SBOM Formats

Sbomnix outputs SBOMs in multiple industry-standard formats:

- **CycloneDX** (sbom.cdx.json)
- **SPDX** (sbom.spdx.json)
- **CSV** format for tabular analysis

## Dependency Tracking

The tools distinguish between two dependency types:

**Buildtime dependencies** encompass the complete closure needed to reproduce builds, including compilers and build infrastructure. Computing this requires only derivation evaluation, not building the target.

**Runtime dependencies** represent the subset actually needed at execution time. Determining these requires building the target, as Nix scans outputs for references to other store paths.

By default, tools analyze runtime dependencies, though the `--buildtime` flag enables buildtime analysis.

## Integration and Usage

Sbomnix integrates with Nix flakes and development shells. Users can execute it via `nix run` commands or within Nix development environments. The tool enriches SBOMs with nixpkgs metadata including descriptions, licenses, maintainers, and homepage links when targets are flake references.

The project includes 21+ releases and originates from the Ghaf Framework security initiative.
