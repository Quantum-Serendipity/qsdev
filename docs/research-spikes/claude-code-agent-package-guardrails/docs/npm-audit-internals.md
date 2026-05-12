<!-- Source: https://docs.npmjs.com/cli/v11/commands/npm-audit/ -->
<!-- Retrieved: 2026-05-12 -->

# npm audit: Complete Technical Documentation

## Core Functionality

The `npm audit` command performs security vulnerability scanning by submitting dependency information to a registry endpoint. It identifies known vulnerabilities and calculates remediations. The command exits with code 0 if no vulnerabilities are found; otherwise, the exit code depends on the `audit-level` configuration.

## Registry Endpoints

### Bulk Advisory Endpoint (Primary)
npm posts a JSON payload containing package names and versions to `/-/npm/v1/security/advisories/bulk`. This endpoint is the default approach as of npm v7 due to superior performance. The registry responds with advisory objects containing vulnerability metadata for matching dependencies.

### Quick Audit Endpoint (Fallback)
If the Bulk endpoint fails or returns invalid data, npm attempts the slower Quick Audit endpoint. This submits the complete `package-lock.json` tree along with system metadata (npm version, node version, platform, architecture, environment).

## Advisory Response Format

The registry returns advisory objects including: `name`, `url`, `id`, `severity`, `vulnerable_versions`, and `title`. npm then calculates "meta-vulnerabilities" -- dependencies that are vulnerable through transitive dependencies on vulnerable packages.

## Exit Codes & Audit Levels

The command supports granular failure thresholds: "info", "low", "moderate", "high", or "critical". The exit code reflects whether detected vulnerabilities meet the specified minimum severity level, allowing CI environments to customize failure behavior independently from report output.

## Configuration Options

**Key flags:**
- `--audit-level`: Sets minimum vulnerability severity for non-zero exit
- `--dry-run`: Reports remediation actions without applying changes
- `--force`: Permits SemVer-major version updates during fixes
- `--json`: Outputs structured JSON instead of human-readable format
- `--package-lock-only`: Modifies lock file without touching node_modules
- `--omit`: Excludes dependency types (dev, optional, peer)

## Audit Signatures Feature

`npm audit signatures` verifies ECDSA registry signatures and provenance attestations. The command validates packages against public keys available at `registry-host.tld/-/npm/v1/keys`. Keys contain expiration metadata, keyid (SHA256 fingerprint), keytype, scheme, and base64-encoded public key material. The `--include-attestations` flag includes full sigstore bundles (DSSE envelopes and transparency logs) in JSON output.

## Remediation & Fix Behavior

When `npm audit fix` runs, it executes a full-fledged `npm install` operation, so all installer configurations apply. Automatic fixes update packages to non-vulnerable versions when possible. If the vulnerability chain extends to the root project and cannot be resolved without expanding dependency ranges, `--force` is required.

## Programmatic Considerations

npm internally uses the `@npmcli/metavuln-calculator` module to transform advisories into vulnerability objects. Meta-vulnerability calculations are cached in `~/.npm` and re-evaluated only when advisory ranges change or new package versions publish. The package-lock file is required by default; `--no-package-lock` bypasses this requirement but may produce inconsistent results across runs.
