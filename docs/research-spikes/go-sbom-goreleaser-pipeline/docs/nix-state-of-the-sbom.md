# Nix State of the SBOM

- **Source**: https://arnout.engelen.eu/blog/nix-state-of-the-sbom/
- **Retrieved**: 2026-05-15

## Overview

Examines how Software Bills of Materials (SBOMs) work in the Nix ecosystem, comparing three main tools and their effectiveness at capturing package dependencies and metadata.

## Main SBOM Tools Analyzed

**bombon**: Works at the `.nix` level, capturing metadata from the `meta` block. However, it "cannot discover dependency relations defined via string interpolation," resulting in missing references to assets and google-fonts in the example.

**genealogos**: Also operates at the `.nix` level using nixtract. Provides hierarchical output but shares bombon's limitation with string-interpolated dependencies.

**sbomnix**: Functions at the `.drv` representation level. Despite lacking direct access to metadata sections, it successfully identifies the google-fonts reference and attempts metadata enrichment through nixpkgs attributes.

## Practical Example

The article walks through a nethogs derivation, demonstrating how each tool handles:
- License and metadata extraction
- Dependency discovery (including indirect references via `fetchFromGitea` and `fetchFromGitHub`)
- Hierarchy representation in output

## Key Challenges

**Missing Dependencies**: All tools struggle with identifying resources copied into artifacts during build phases, particularly those referenced through string interpolation.

**Tree Pruning**: Runtime SBOMs remain incomplete — they fail to capture data incorporated from build-time dependencies into final outputs.

**Component Identification**: Tools need both precise Nix derivation paths and fuzzy identifiers like Package URLs (PURLs) and CPEs for vulnerability matching.

## Comparative Strengths

Nix's advantages include comprehensive system definitions as expressions. Traditional distributions struggle to document software installed via `curl | sh` or package managers, requiring filesystem analysis tools like syft.

## Recommendations

- Enriching nixpkgs metadata with explicit PURLs
- Recording component type information
- Improving language-specific dependency bundle integration
- Accessing the #nixpkgs-sbom Matrix channel for ongoing work
