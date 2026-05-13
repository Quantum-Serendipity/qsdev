<!-- Source: https://docs.npmjs.com/cli/v11/commands/npm-audit/ -->
<!-- Retrieved: 2026-05-12 -->

# npm audit Documentation Summary

## Core Functionality

The `npm audit` command scans project dependencies for security vulnerabilities. It submits dependency information to a registry and returns a report of known issues. The command exits with code 0 if no vulnerabilities are found.

## Command Syntax

```
npm audit [fix|signatures]
```

### Primary Variants:
- `npm audit` -- Reports vulnerabilities without modifications
- `npm audit fix` -- Automatically applies compatible security updates
- `npm audit signatures` -- Verifies registry signatures and provenance attestations

## Severity Levels & Exit Codes

Five vulnerability severity tiers:
- info
- low
- moderate
- high
- critical

The `npm audit` command will exit with a 0 exit code if no vulnerabilities were found. The `npm audit fix` command will exit with 0 exit code if no vulnerabilities are found or if the remediation is able to successfully fix all vulnerabilities.

Exit codes for findings depend on the `--audit-level` configuration.

## Key Flags & Configuration Options

| Flag | Purpose |
|------|---------|
| `--audit-level` | Sets minimum severity triggering non-zero exit (null/"info"/"low"/"moderate"/"high"/"critical"/"none") |
| `--fix` | Applies automatic remediations |
| `--dry-run` | Shows changes without executing them |
| `--force` | Permits SemVer-major updates during remediation |
| `--json` | Outputs structured JSON results |
| `--package-lock-only` | Updates lock file without modifying node_modules |
| `--only=prod` | Excludes devDependencies from analysis |
| `--omit` | Excludes dependency types (dev/optional/peer) |
| `--include` | Specifies included dependency types |
| `--workspace` | Targets specific workspace(s) |
| `--workspaces` | Runs across all configured workspaces |
| `--include-attestations` | Includes sigstore bundles in JSON output |

## Audit Endpoints

**Bulk Advisory Endpoint** (Primary): Sends package names and versions to `/-/npm/v1/security/advisories/bulk`, optimized for speed since npm v7.

**Quick Audit Endpoint** (Fallback): Submits the full package tree with metadata if bulk endpoint fails. Considerably slower but more thorough.

Any packages in the tree that do not have a `version` field in their package.json file will be ignored.

## Signature Verification

Registry signatures employ ECDSA with SHA2-NISTP256. Each signature includes:
- `keyid` -- SHA256 fingerprint
- `sig` -- ECDSA signature matching template: `${package.name}@${package.version}:${package.dist.integrity}`

Public keys are retrieved from `registry-host.tld/-/npm/v1/keys` with expiration tracking and base64 encoding.

## Meta-Vulnerabilities

A "meta-vulnerability" is a dependency that is vulnerable by virtue of dependence on vulnerable versions of a vulnerable package. The system caches these calculations in `~/.npm` and re-evaluates upon advisory range changes or new package versions.

## Lock File Requirement

By default, npm requires `package-lock.json` or shrinkwrap for audits. The `--no-package-lock` flag bypasses this but may produce inconsistent results across runs.
