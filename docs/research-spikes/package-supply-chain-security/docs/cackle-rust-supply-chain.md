# Cackle: Hardening Rust Supply Chain Security

- **Source**: https://davidlattimore.github.io/posts/2023/10/09/making-supply-chain-attacks-harder.html
- **Retrieved**: 2026-05-12

## Overview

David Lattimore's article introduces Cackle (cargo-acl), a tool designed to mitigate supply chain attacks in Rust projects.

## How Supply Chain Attacks Occur

The article identifies several attack vectors:
- Compromised crates.io credentials
- Abandoned projects handed to unknown maintainers
- Malicious code injected into established crates
- Protest-ware with unintended consequences

## Cackle's Approach

Cackle functions as a code ACL (Access Control List) checker, configured through a `cackle.toml` file. It classifies restricted API categories like "net," "fs," and "process," then verifies which dependencies can access them.

### API Definition Example

The tool uses inclusion/exclusion patterns. For instance, a "process" API might include `std::process` while excluding `std::process::exit`. This granular approach prevents false positives while catching suspicious activity.

### Configuration Permissions

Users specify which packages can access particular APIs: "The `rustyline` package is allowed to use filesystem APIs and also allowed to use unsafe code."

## Detection Mechanism

Rather than runtime monitoring, Cackle analyzes compiled binaries. It parses object files to map function references, then traces these back to source locations and originating packages. "Dead code is not considered with regard to API usage," meaning unused functions won't trigger false alarms.

## Sandboxing Implementation

Build scripts and procedural macros run within Bubblewrap sandboxes, with network/filesystem restrictions enforced during compilation. The tool also detects build script failures and suspicious linker instructions.

## Practical Limitations

The author acknowledges circumvention possibilities: incomplete configurations allow undetected API usage, granted permissions provide unrestricted access within that scope, platform-specific malware on Windows/macOS escapes detection, and implementation bugs might allow evasion.

## Integration and Observations

For CI pipelines, developers run `cargo acl -n`, with GitHub Actions support provided. Empirical analysis shows approximately half of dependencies require no special permissions, unsafe code dominates permission requests, while network APIs remain relatively uncommon.
