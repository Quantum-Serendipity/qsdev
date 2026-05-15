# Bombon: Nix CycloneDX SBOM Tool

- **Source**: https://github.com/nikstur/bombon
- **Retrieved**: 2026-05-15

## Overview

Bombon automatically builds CycloneDX Software Bills of Materials (SBOMs) for Nix packages. It generates version 1.5 SBOMs designed to comply with German Technical Guideline TR-03183 v2.0.0 and the US Executive Order 14028.

## Installation Methods

**Using Flakes:**
Users can initialize a project with `nix flake init -t github:nikstur/bombon` or manually add bombon as a flake input, then call `bombon.lib.${system}.buildBom` with a package.

**Using Niv:**
The tool can be added via niv and imported through a default.nix configuration file.

## Handling Vendored Dependencies

A key feature addresses ecosystem-specific challenges: "Some language ecosystems in Nixpkgs (most notably Rust and Go) vendor dependencies." Rather than representing each as separate derivations, bombon reads SBOMs from other tools via a `bombonVendoredSbom` passthru attribute. The `passthruVendoredSbom.rust` function integrates these automatically.

## Configuration Options

The `buildBom` function accepts an optional attribute set with three parameters:

- **extraPaths**: Store paths for inclusion, useful when building images that "discard their references"
- **includeBuildtimeDependencies**: Boolean flag for compile-time dependencies
- **excludes**: Regex patterns to filter out specific store paths

## Usage Example

A basic invocation demonstrates flexibility:

```nix
bombon.lib.${system}.buildBom pkgs.hello {
  extraPaths = [ pkgs.git ];
  includeBuildtimeDependencies = true;
  excludes = [ "service" ];
}
```

The repository shows 142 stars and is written primarily in Rust (61.2%) with Nix (38.8%).
