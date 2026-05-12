<!-- Source: https://raw.githubusercontent.com/tiiuae/sbomnix/main/README.md -->
<!-- Retrieved: 2026-05-12 -->

# sbomnix: Software Supply Chain Tools for Nix - Full README

## Project Description
The sbomnix repository contains command-line tools and Python libraries addressing software supply chain security challenges. The project "aim[s] to help with software supply chain challenges" through multiple integrated utilities.

## Core Tools

**sbomnix**: Generates Software Bill of Materials (SBOMs) from Nix flake references or store paths.
**nixgraph**: Enables querying and visualizing dependency graphs for Nix packages.
**nixmeta**: Summarizes nixpkgs meta-attributes from specified nixpkgs versions.
**vulnxscan**: A vulnerability scanner demonstrating SBOM usage in security scanning workflows.
**repology_cli and repology_cve**: Command-line clients interfacing with repology.org for package information.
**nix_outdated**: Identifies outdated Nix dependencies, prioritizing by downstream impact.
**provenance**: Generates SLSA v1.0 compliant provenance attestation files in JSON format.

## Installation Methods

Users can run sbomnix as a Nix flake directly:
```bash
nix run github:tiiuae/sbomnix#sbomnix -- --help
```

Alternatively, users may clone the repository and use the development shell:
```bash
git clone https://github.com/tiiuae/sbomnix
cd sbomnix
nix develop
```

## SBOM Output Formats
- **CycloneDX** (sbom.cdx.json)
- **SPDX** (sbom.spdx.json)
- **CSV** (sbom.csv)

## Key Usage Examples

### Basic SBOM Generation
```bash
sbomnix github:NixOS/nixpkgs/nixos-unstable#wget
```

### Including Buildtime Dependencies
```bash
sbomnix github:NixOS/nixpkgs/nixos-unstable#wget --buildtime
```

### Dependency Visualization
```bash
nixgraph github:NixOS/nixpkgs/nixos-unstable#wget --depth=2
```

## Dependency Classification
Runtime dependencies represent "the transitive set of those recorded references: the store paths the built output actually needs at runtime." Buildtime dependencies encompass the complete closure required for reproducible builds, typically including compilers and build infrastructure.

## Verbosity Control
All tools support consistent verbosity flags: no flag or `--verbose=0` displays INFO output, `-v` or `--verbose=1` enables verbose progress details, `-vv` or `--verbose=2` enables debugging details, and `-vvv` or `--verbose=3` enables spam-level output.

## Development Requirements
The project requires the Nix command-line tool in `$PATH` and modern Nix supporting `nix-command` and `--json-format 1`. Development workflows use the flakes-based development shell for testing and contribution.

## Licensing
Apache-2.0 license. The repository also includes acknowledgments recognizing code origin from the vulnix project.
