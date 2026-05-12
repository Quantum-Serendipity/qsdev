<!-- Source: https://github.com/DeterminateSystems/flake-checker -->
<!-- Retrieved: 2026-05-12 -->

# Nix Flake Checker: Validation Rules, Configuration, and Usage

## Validation Rules (Default Checks)

Flake-checker performs three primary health validations on flake.lock files:

1. **Supported Git refs**: Ensures explicit Nixpkgs Git references are in the supported branches list
2. **Recency verification**: Confirms Nixpkgs dependencies are under 30 days old
3. **Upstream ownership**: Validates the NixOS organization owns the GitHub dependency (prevents forks/variants)

## Configuration Options

**Command-line flags and environment variables:**

| Flag | Env Variable | Purpose | Default |
|------|--------------|---------|---------|
| `--check-outdated` | `NIX_FLAKE_CHECKER_CHECK_OUTDATED` | Age validation | enabled |
| `--check-owner` | `NIX_FLAKE_CHECKER_CHECK_OWNER` | Ownership verification | enabled |
| `--check-supported` | `NIX_FLAKE_CHECKER_CHECK_SUPPORTED` | Branch support validation | enabled |

**CEL (Common Expression Language) policies:**

Custom conditions can be applied via `--condition` flag. Available variables include `gitRef`, `numDaysOld`, `owner`, `supportedRefs`, and `refStatuses`. Recommended: "supportedRefs.contains(gitRef) && numDaysOld < 30 && owner == 'NixOS'"

## How to Run

**CLI execution:**
```
nix run github:DeterminateSystems/flake-checker
nix run github:DeterminateSystems/flake-checker /path/to/flake.lock
```

**GitHub Actions:**
Integrate via DeterminateSystems/flake-checker-action

**Git hooks:** Not mentioned in provided documentation, but can be configured as a custom hook.

## Security Relevance

Catches outdated, unsupported, or forked Nixpkgs dependencies -- reducing exposure to unpatched vulnerabilities by enforcing use of current, official upstream sources.

## Performance

Not specified in documentation. Written in Rust, so expected to be fast. Checks are lightweight (reads flake.lock JSON, no network calls for basic checks).

## Telemetry

Enabled by default; disable with `--no-telemetry` or `FLAKE_CHECKER_NO_TELEMETRY=true`.
