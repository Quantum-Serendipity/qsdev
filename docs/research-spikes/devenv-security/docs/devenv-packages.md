# devenv Package Management
- **Source**: https://devenv.sh/packages/
- **Retrieved**: 2026-05-12

## Basic Package Declaration

Packages are specified in `devenv.nix` by referencing the `pkgs` input:

```nix
{ pkgs, ... }:
{
  packages = [
    pkgs.git
    pkgs.jq
    pkgs.libffi
    pkgs.zlib
  ];
}
```

"Packages allow you to add executables and libraries/headers to your environment." Added packages become accessible in your PATH upon shell activation.

## Package Discovery

The `devenv search <NAME>` command helps locate available packages with their versions and descriptions, searching against your pinned Nixpkgs input version.

For finding which package provides specific files: "If you'd like to see what package includes a specific file" use `nix run github:nix-community/nix-index-database <filename>`.

## Version Pinning

Not covered in detail on this page. Individual package version pinning is not a native feature -- devenv.lock pins the nixpkgs input revision. For specific package versions, you can:
- Use a different nixpkgs input revision that contains the desired version
- Use overlays to override package definitions
- Reference packages from multiple nixpkgs inputs
