# sbomnix: Software Bill of Materials for Nix

- **Source**: https://github.com/tiiuae/sbomnix
- **Retrieved**: 2026-05-15

## Overview

sbomnix is a command-line utility that "generates SBOMs given a Nix flake reference or store path."

The project encompasses several related tools:
- **sbomnix**: SBOM generation
- **nixgraph**: dependency visualization
- **nixmeta**: nixpkgs metadata summarization
- **vulnxscan**: vulnerability scanning
- **repology_cli/repology_cve**: Repology.org integration
- **nix_outdated**: outdated dependency detection
- **provenance**: SLSA v1.0 attestation generation

## Core Functionality

### Flake References vs. Store Paths

1. **Flake References** (recommended): Enable automatic nixpkgs metadata enrichment. Resolves package descriptions, licenses, maintainers, and homepage information.
2. **Store Paths**: Direct references to Nix store locations. Work but skip metadata enrichment by default.

### Output Formats

- CycloneDX JSON (`sbom.cdx.json`)
- SPDX JSON (`sbom.spdx.json`)
- CSV format (`sbom.csv`)

### Dependency Classification

**Runtime Dependencies**: Identified by scanning built outputs for store path references. Requires building the target. "Scans the given target and generates an SBOM including the runtime dependencies" by default.

**Buildtime Dependencies**: Computed from derivation closures without requiring a build. Represents "all the store paths Nix must have available to reproduce the build, including compilers, build tools, standard libraries."

## Usage Examples

### Basic SBOM Generation
```bash
$ sbomnix github:NixOS/nixpkgs?ref=nixos-unstable#wget
```

### With Buildtime Dependencies
```bash
$ sbomnix github:NixOS/nixpkgs/nixos-unstable#wget --buildtime
```

### From Store Paths
```bash
$ sbomnix /path/to/result
```

## Metadata Enrichment

- **Automatic** (flakeref targets): Derives nixpkgs version from target context
- **Explicit nixpkgs**: `--meta-nixpkgs <flakeref-or-path>`
- **NIX_PATH integration**: `--meta-nixpkgs nix-path`
- **Disabled**: `--exclude-meta`

Metadata fields recorded include: metadata source method, path, revision, flakeref, version, and descriptive messages.

## Technical Details

Written primarily in Python (97.5%) with supporting Nix code (2.0%). Requires the Nix command-line tool in the system PATH.
