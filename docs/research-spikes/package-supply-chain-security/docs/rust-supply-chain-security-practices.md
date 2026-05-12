<!-- Source: https://blog.ortham.net/posts/2025-10-02-rust-supply-chain-security/ -->
<!-- Retrieved: 2026-05-12 -->

# Improving Rust Projects' Supply Chain Security

## Trusted Publishing for Crate Releases

Migrated from local API token-based publishing to crates.io's Trusted Publishing implementation. Key advantages include:

- Automatic token creation/revocation with 30-minute lifespans
- Tokens scoped to specific crates only
- Tokens accessible exclusively through designated GitHub Actions workflows
- Optional restriction to specific GitHub environments

A significant motivation: "crates.io doesn't support multi-factor authentication for publishing," making tokens the sole authentication factor. Trusted Publishing eliminates local token storage and links published versions back to GitHub Actions runs for transparency.

**Current limitation**: First-version crate publishing via Trusted Publishing remains unimplemented, requiring temporary tokens for new crate releases.

## Dependency Review Workflow

**Selective Updates**: Skipping non-essential releases reduces churn and avoids potentially compromised versions without active intervention. Introducing deliberate delays (approximately one week) allows upstream issues to surface.

**Code Inspection**: Using diff.rs instead of Dependabot's Git comparisons, reviewing diffs for:
- Obfuscated code patterns
- Unexpected network requests in parsing logic
- Build script modifications
- New unsafe keyword usage
- Dependency tree changes

**Dependency Acquisition**: Avoiding linked repositories, instead using docs.rs source views or diff.rs to prevent accidental build script execution before code review.

## Cargo-Vet Integration

For tracking reviews across repositories, using cargo-vet (over cargo-crev due to Windows build issues). The tool enables:

- Recording exemptions for reviewed versions
- Importing third-party audits (Google, Mozilla, Bytecode Alliance, ISRG)
- Importing audits totaling 206 records across dependencies
- Delta audits between versions
- Publisher-level trust declarations

For esplugin specifically: 57 dependencies fully audited, 19 exempted, with CI enforcing audit coverage for all non-exempted dependencies.

**Limitations acknowledged**:
- Trusted Publishing support unreleased at time of writing
- No target-specific filtering (affects irrelevant platform dependencies)
- Optional dependencies processed regardless of actual use
- Rough backlog estimation including non-code lines

## GitHub Actions Security Hardening

**Token permissions**: Enforced read-only defaults for `GITHUB_TOKEN` across all repositories.

**Third-party action pinning**: Official GitHub Actions use major version tags; third-party actions pinned to commit hashes with version comments for maintainability.

**Cargo install practices**: When installing tools in CI:
- Passes `--locked` to respect lockfiles
- Installs from Git repositories with commit hash verification rather than crates.io
- Checks SHA-256 hashes for downloaded files before use

## Existing Security Foundations

- Strong unique passwords and 2FA for accounts
- Committed Cargo.lock files in all projects
- Minimal third-party GitHub Actions usage
