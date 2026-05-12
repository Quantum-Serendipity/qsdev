# devenv.sh Rust Language Configuration
- **Source**: https://devenv.sh/supported-languages/rust/
- **Retrieved**: 2026-05-12

## Overview

The `languages.rust` module provides comprehensive support for Rust development with flexible toolchain management through two approaches: nixpkgs channel (default) or rust-overlay channels (stable, beta, nightly).

## Toolchain Management Approaches

**1. Nixpkgs Channel (Default)** — uses Rust version from current nixpkgs revision
**2. Rust-Overlay Channels** — supports "stable", "beta", or "nightly" with cross-compilation target support

## Complete Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `languages.rust.enable` | boolean | false | Enable Rust development tools |
| `languages.rust.channel` | "nixpkgs"\|"stable"\|"beta"\|"nightly" | "nixpkgs" | Rustup toolchain to install |
| `languages.rust.components` | list of string | ["rustc", "cargo", "clippy", "rustfmt", "rust-analyzer"] | Rustup components to install |
| `languages.rust.cranelift.enable` | boolean | false | Use Cranelift as codegen backend for dev builds |
| `languages.rust.cranelift.excludePackages` | list of string | [] | Crate names using LLVM instead of Cranelift |
| `languages.rust.cranelift.forceBuildScriptsLlvm` | boolean | false | Force build scripts/proc macros to use LLVM |
| `languages.rust.import` | function | -- | Import Cargo project using crate2nix |
| `languages.rust.lld.enable` | boolean | false | Use lld as the linker |
| `languages.rust.lsp.enable` | boolean | true | Enable Rust Language Server |
| `languages.rust.lsp.package` | package | Depends on channel | Language server package |
| `languages.rust.mold.enable` | boolean | false | Use mold as faster linker |
| `languages.rust.rustflags` | string | "" | Extra flags for Rust compiler |
| `languages.rust.targets` | list of string | [] | Extra compilation targets |
| `languages.rust.toolchain.cargo` | null or package | pkgs.cargo | Cargo package |
| `languages.rust.toolchain.clippy` | null or package | pkgs.clippy | Clippy package |
| `languages.rust.toolchain.rust-analyzer` | null or package | pkgs.rust-analyzer | Rust-analyzer package |
| `languages.rust.toolchain.rustc` | null or package | pkgs.rustc | Rustc package |
| `languages.rust.toolchain.rustfmt` | null or package | pkgs.rustfmt | Rustfmt package |
| `languages.rust.toolchainFile` | null or path | null | Path to rust-toolchain.toml |
| `languages.rust.toolchainPackage` | package | -- | Aggregated toolchain package (auto-set) |
| `languages.rust.version` | string | "latest" | Rust version (only with non-nixpkgs channel) |
| `languages.rust.wild.enable` | boolean | false | Use wild as fast Linux linker |

## Key Features

- Git hooks integration with rustfmt and clippy
- Cross-compilation support via targets
- Toolchain file support (rust-toolchain.toml)
- Component management beyond defaults
- Linker options: lld, mold, or wild
