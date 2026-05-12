# Minimum Release Age — Renovate Documentation

- **Source URL**: https://docs.renovatebot.com/key-concepts/minimum-release-age/
- **Retrieved**: 2026-05-12

## Overview

`minimumReleaseAge` is a Renovate feature requiring a waiting period before suggesting dependency updates. The goal is to "reduce risk supply chain security risks" rather than slow development cycles. This concept is sometimes called "dependency cooldown" in other ecosystems.

## Configuration Options

Three configuration parameters control this feature:
- `minimumReleaseAge` (formerly `stabilityDays`)
- `minimumReleaseAgeBehaviour`
- `internalChecksFilter`

## How It Works

### Default Behavior

"When the time passed since the release is _less_ than the set `minimumReleaseAge`: Renovate adds a 'pending' status check to that update's branch. After enough days have passed: Renovate replaces the 'pending' status with a 'passing' status check."

The feature waits for each separate version independently, not for the absence of releases during a time period.

### Release Timestamp Requirements

Renovate 42 changed timestamp handling. Without a release timestamp, updates are now treated as not yet passing minimum age checks — a safer default than previous versions that treated missing timestamps as already approved.

## npm Integration

Renovate automatically integrates with npm's package manager. "When `minimumReleaseAge` is configured, Renovate passes `--before=<date>` to npm commands during lock file generation."

Key behaviors:
- The `--before` date is calculated as `now - minimumReleaseAge`
- If `.npmrc` already contains stricter settings, Renovate uses the older date
- If existing lockfiles contain packages published after the cutoff, npm fails with `ETARGET`; Renovate automatically retries without `--before` and logs a warning

## Update Type Support

| Type | Support | Notes |
|------|---------|-------|
| major/minor/patch | Yes | Depends on manager, datasource, package |
| pin/pinDigest | No | Not yet supported |
| digest | Partial | Generally not supported |
| lockFileMaintenance/lockFileUpdate/rollback/bump/replacement | No | Not applicable or not yet supported |

## Security Updates

"Security updates bypass any `minimumReleaseAge` checks, and so will be raised as soon as Renovate detects them."

## Configuration Example: Opting Out Dependencies

```json
{
  "$schema": "https://docs.renovatebot.com/renovate-schema.json",
  "extends": ["security:minimumReleaseAgeNpm"],
  "packageRules": [
    {
      "description": "Disable minimum release age checks for internal dependencies",
      "matchPackageNames": ["@super-secret-organisation/*"],
      "minimumReleaseAge": null
    }
  ]
}
```

As of version 42.19.5, `minimumReleaseAge=0 days` equals `minimumReleaseAge=null`.

## Registry Support

Public registries with timestamp support include npm, PyPI, Maven Central, Docker Hub, GitHub, Go, Ruby, Terraform, and JSR. Custom registries require explicit configuration to expose timestamps.

## Custom Registry Configuration

**Maven example:**
```json
{
  "packageRules": [
    {
      "matchDatasources": ["maven"],
      "registryUrls": [
        "https://repo1.maven.org/maven2",
        "https://europe-maven.pkg.dev/org-artifacts/maven-virtual"
      ]
    }
  ]
}
```

**PyPI example:**
```json
{
  "packageRules": [
    {
      "matchDatasources": ["pypi"],
      "registryUrls": [
        "https://pypi.org/pypi/",
        "https://custom-registry.example.com/pypi/some-repo/simple/"
      ]
    }
  ]
}
```

## Recommended Settings

"The recommendation is to set `internalChecksFilter=strict` when using `minimumReleaseAge`, so Renovate will create neither branches (nor PRs) on updates that haven't yet met minimum release age checks."

## Transitive Dependencies

"Renovate does not currently manage any transitive dependencies — instead leaving that to package managers and `lockFileMaintenance`." This is why configuring minimum release age in both Renovate and your package manager is recommended.
