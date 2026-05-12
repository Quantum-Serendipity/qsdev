<!-- Source: https://github.com/rustsec/rustsec/blob/main/cargo-audit/README.md -->
<!-- Retrieved: 2026-05-12 -->

# cargo-audit Documentation

## Overview
cargo-audit audits Rust project dependencies against the RustSec Advisory Database for known vulnerabilities.

## Installation

**Requirements:** Rust 1.74 or later

**Primary method:**
```
$ cargo install cargo-audit
```

**Package managers:**
- Alpine Linux: `apk add cargo-audit`
- Arch Linux: `pacman -S cargo-audit`
- macOS: `brew install cargo-audit`
- OpenBSD: `pkg_add cargo-audit`

## Core Commands

**Basic audit:**
```
$ cargo audit
```

**With ignore option:**
```
$ cargo audit --ignore RUSTSEC-2017-0001
```

## Subcommands

### `cargo audit fix` (Experimental)
Install with: `cargo install cargo-audit --features=fix`
Automatically updates Cargo.toml to address vulnerable dependencies.
Use `--dry-run` flag to preview changes.

### `cargo audit bin`
```
$ cargo audit bin [path/to/binary]
```
Audits compiled binaries for vulnerabilities. Works optimally with binaries compiled using `cargo auditable`.

## Database

- Advisory Database: https://github.com/RustSec/advisory-db/
- Also exported to OSV format, available on osv.dev
- Local git clone updated before each audit

## Programmatic Usage (rustsec crate)

The `rustsec` crate provides a Rust library for programmatic access:
- Fetches advisory-db git repository
- Audits Cargo.lock files against it
- Available on crates.io

## Configuration

Create an `audit.toml` file for persistent configuration of ignored advisories.

## CI/CD Integration

GitHub Actions: Use the dedicated `audit-check` action.
Travis CI: Install cargo-audit in before_script, run `cargo audit` in script.

## Licensing

Dual-licensed under Apache License 2.0 OR MIT license.
