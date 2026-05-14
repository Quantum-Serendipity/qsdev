<!-- Source: https://github.com/NixOS/nixpkgs/issues/384684 -->
<!-- Retrieved: 2026-05-14 -->

# libzim Build Failure in NixOS Nixpkgs

## Issue Details
**Issue #384684** reports that the `libzim` package fails to build in NixOS Unstable (version 25.05), which blocks the dependent package `goldendict-ng`.

## Root Cause
The linking step fails with numerous "undefined reference" errors related to ICU (International Components for Unicode) library symbols. The compiler cannot find ICU functions with the `_76` suffix.

This indicates a **missing or incompatible ICU library linkage** during the final shared object compilation.

## Affected Platform
- **System**: x86_64-linux
- **NixOS Version**: 25.05 (Warbler)
- **Build Tool**: Meson
- **Reproducible**: Yes, confirmed on Hydra build infrastructure

## Current Status
The issue is **Closed** (resolved). The fix involved correcting ICU library linkage.

## Implications for gdev
- libzim exists in nixpkgs but has had build stability issues
- kiwix-tools (which depends on libzim) is also in nixpkgs
- python-libzim PyPI wheels bundle their own libzim — this avoids the nixpkgs libzim build issues entirely
- For gdev, using PyPI wheels via uv/pip in a Nix-managed venv is more reliable than depending on nixpkgs libzim
