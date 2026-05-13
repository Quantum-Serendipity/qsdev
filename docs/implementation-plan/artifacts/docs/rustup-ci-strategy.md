# Rustup CI Testing Strategy
> Source: https://github.com/rust-lang/rustup/blob/main/ci/actions-templates/README.md
> Retrieved: 2026-05-12

## Platform Coverage
Tests all tier 1 and tier 2 with host tools targets per Rust's platform policy.

## Native vs Cross-Compilation
- Platforms with free GitHub runners: native build + full test suite
- Other platforms: cross-build without tests

## Install Script Testing
- Uses rustup-init.sh from the branch under test
- Starts from bare VMs, avoids preinstalled rust/rustup

## Linux Docker Strategy
- x86_64 Linux builds run inside Docker image from rust-lang/rust CI
- Ensures linking against same libc as Rust release

## Windows
- Uses mingw from Rust CI infrastructure
- Prefers Visual Studio 2017 image

## Workflow Structure
- Generated from templates (ci/actions-templates/)
- Built workflows in .github/workflows/
- Different triggers: always-run, PR, main branch, stable release
