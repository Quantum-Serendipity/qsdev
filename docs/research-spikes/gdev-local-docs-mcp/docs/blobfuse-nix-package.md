<!-- Source: https://mynixos.com/nixpkgs/package/blobfuse -->
<!-- Retrieved: 2026-05-14 -->

# BlobFuse Nix Package

- **Package name**: blobfuse
- **Version**: 2.5.3
- **License**: MIT
- **Maintainer**: Jean-Baptiste Giraudeau
- **Description**: "Mount an Azure Blob storage as filesystem through FUSE."

## Executables Provided
- azure-storage-fuse
- health-monitor

## Platform Support
Supports 24 Linux platforms including:
- x86_64-linux
- aarch64-linux
- i686-linux
- ARM variants (armv5tel, armv6l, armv7a, armv7l)
- PowerPC, RISC-V, MIPS variants

## Status
Package is stable (not marked as broken, insecure, unfree, or unsupported).

## Source Location
pkgs/by-name/bl/blobfuse/package.nix

## Key Takeaway
BlobFuse2 is already packaged in nixpkgs as `blobfuse` (v2.5.3), making it directly usable on NixOS without custom packaging.
